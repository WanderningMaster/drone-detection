package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"main/apipb"
	"main/service"
	"strconv"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type packet struct {
	seq  uint32
	data []byte
}

const (
	broker              = "tcp://0.0.0.0:1883"
	topicWildcard       = "sensors/audio/%d"
	statusTopicWildcard = "sensors/status/#"
	pcmFile             = "output/sensor-%d.pcm"
	logBytesThreshold   = 100 * 1014
	SAMPLE_RATE         = 16000
)

var streams map[int]chan packet
var mu sync.Mutex

func subscribeCb(sensorId int) func(mqtt.Client, mqtt.Message) {
	return func(_ mqtt.Client, msg mqtt.Message) {
		payload := msg.Payload()
		if len(payload) < 4 {
			fmt.Println("invalid payload")
			return
		}
		seq := binary.BigEndian.Uint32(payload[:4])
		data := make([]byte, len(payload)-4)
		copy(data, payload[4:])

		ch, _ := streams[sensorId]

		select {
		case ch <- packet{seq, data}:
		default:
			log.Printf("[sensor-%d] dropping packet %d (stream too slow)\n", sensorId, seq)
		}
	}
}

func main() {
	ctx := context.Background()
	analyzer := service.NewAnalyzerService("0.0.0.0:50051")
	gateway := service.NewGatewayService("0.0.0.0:50052")
	streams = make(map[int]chan packet)

	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID("go-audio-server")
	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("[MQTT] Connected (or reconnected) to %s", broker)
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer client.Disconnect(250)
	fmt.Printf("Connected to %s\n", broker)

	client.Subscribe(statusTopicWildcard, 1, func(c mqtt.Client, m mqtt.Message) {
		parts := strings.Split(m.Topic(), "/")
		sensorId := parts[len(parts)-1]
		id, _ := strconv.Atoi(sensorId)

		topic := fmt.Sprintf(topicWildcard, id)

		if string(m.Payload()) == "online" {
			mu.Lock()
			ch, exists := streams[id]
			if !exists {
				log.Printf("Sensor connected: sensor-%d", id)
				ch = make(chan packet, 10)
				streams[id] = ch
				go handleStream(ctx, analyzer, id, ch)
			}
			mu.Unlock()

			client.Subscribe(topic, 1, subscribeCb(id))
		} else {
			mu.Lock()
			delete(streams, id)
			log.Printf("Sensor disconnected: sensor-%d", id)
			mu.Unlock()

			client.Unsubscribe(topic)
		}
		gateway.UpdateStatus(ctx, &apipb.StatusRequest{
			SensorId: int32(id),
			Status:   string(m.Payload()),
		})
	})

	select {}
}

func handleStream(ctx context.Context, analyzer apipb.AnalyzerServiceClient, sensorId int, ch chan packet) {
	defer log.Printf("[sensor-%d] stream handler exiting\n", sensorId)

	var (
		expectedSeq     uint32
		totalBytesWrite int64
		buffer          []byte
		batchSize       = 10 * SAMPLE_RATE * 2 // 10s * sample rate * 1 channel * 16bit pcm
	)
	stream, err := analyzer.Analyze(ctx)
	if err != nil {
		log.Fatalf("analyze rpc error: %v", err)
	}

	for pkt := range ch {
		if pkt.seq != expectedSeq {
			log.Printf("[sensor-%d] seq mismatch: got %d, expected %d\n", sensorId, pkt.seq, expectedSeq)
			expectedSeq = pkt.seq
		}

		n := int64(len(pkt.data))
		totalBytesWrite += n
		buffer = append(buffer, pkt.data...)
		expectedSeq++

		if totalBytesWrite >= int64(batchSize) {
			log.Printf("[sensor-%d] total bytes written: %d\n", sensorId, totalBytesWrite)

			_ = stream.Send(&apipb.AudioBuf{
				SensorId:  int32(sensorId),
				SeqOffset: expectedSeq,
				Pcm:       buffer,
			})
			buffer = buffer[:0]
			totalBytesWrite = 0
		}
	}
}
