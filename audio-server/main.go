package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type packet struct {
	seq  uint32
	data []byte
}

const (
	broker            = "tcp://0.0.0.0:1883"
	topicWildcard     = "sensors/audio/%d"
	pcmFile           = "output/sensor-%d.pcm"
	logBytesThreshold = 100 * 1014
)

var streams map[int]chan packet

func subscribeCb(sensorId int, mu *sync.Mutex) func(mqtt.Client, mqtt.Message) {
	return func(_ mqtt.Client, msg mqtt.Message) {
		payload := msg.Payload()
		if len(payload) < 4 {
			fmt.Println("invalid payload")
			return
		}
		seq := binary.BigEndian.Uint32(payload[:4])
		data := make([]byte, len(payload)-4)
		copy(data, payload[4:])

		mu.Lock()
		ch, exists := streams[sensorId]
		if !exists {
			log.Printf("Connected sendor %d", sensorId)
			f, err := os.Create(fmt.Sprintf(pcmFile, sensorId))
			if err != nil {
				log.Fatal(err)
			}

			ch = make(chan packet, 10)
			streams[sensorId] = ch
			go handleStream(sensorId, ch, f)
		}
		mu.Unlock()

		select {
		case ch <- packet{seq, data}:
		default:
			log.Printf("[sensor-%d] dropping packet %d (stream too slow)\n", sensorId, seq)
		}
	}
}

func main() {
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID("go-audio-server")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer client.Disconnect(250)
	fmt.Printf("Connected to %s\n", broker)

	streams = make(map[int]chan packet)
	var mu sync.Mutex

	sensorIds := []int{1, 2}
	for _, id := range sensorIds {
		topic := fmt.Sprintf(topicWildcard, id)
		client.Subscribe(topic, 1, subscribeCb(id, &mu))
		fmt.Printf("Subscribed to topic: %s\n", topic)

	}

	select {}
}

func handleStream(sensorId int, ch chan packet, f *os.File) {
	defer f.Close()
	defer log.Printf("[sensor-%d] stream handler exiting\n", sensorId)

	var (
		expectedSeq     uint32
		totalBytesWrite int64
		nextLogPoint    = int64(logBytesThreshold)
	)

	for pkt := range ch {
		if pkt.seq != expectedSeq {
			log.Printf("[sensor-%d] seq mismatch: got %d, expected %d\n", sensorId, pkt.seq, expectedSeq)
			expectedSeq = pkt.seq
		}

		if _, err := f.Write(pkt.data); err != nil {
			log.Printf("write error: %v", err)
		}

		n := int64(len(pkt.data))
		totalBytesWrite += n
		expectedSeq++

		if totalBytesWrite >= nextLogPoint {
			log.Printf("[sensor-%d] total bytes written: %d\n", sensorId, totalBytesWrite)
			nextLogPoint += logBytesThreshold
		}
	}
}
