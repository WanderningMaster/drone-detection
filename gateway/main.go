package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"main/grpc"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Location struct {
	Lat float64 `bson:"lat" json:"lat"`
	Lon float64 `bson:"lon" json:"lon"`
}

type Sensor struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	SensorID int32              `bson:"sensor_id" json:"sensor_id"`
	Name     string             `bson:"name" json:"name"`
	Location Location           `bson:"location" json:"location"`
	Status   string             `bson:"status" json:"status"` // "online" or "offline"
}

type AnalysisRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	SensorID  int32              `bson:"sensor_id" json:"sensor_id"`
	Coeff     float64            `bson:"coeff" json:"coeff"`
	Lag       float64            `bson:"lag" json:"lag"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	Recording string             `bson:"recording_path" json:"recording_path"`
}

type AnalysisRecordAgg struct {
	ID        int32     `bson:"_id,omitempty" json:"-"`
	SensorID  int32     `bson:"sensor_id" json:"sensor_id"`
	Coeff     float64   `bson:"coeff" json:"coeff"`
	Lag       float64   `bson:"lag" json:"lag"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Recording string    `bson:"recording_path" json:"recording_path"`
}

type messageFromQueue struct {
	SensorID  int32   `json:"sensor_id"`
	Coeff     float64 `json:"coeff"`
	Lag       float64 `json:"lag"`
	WavBuffer string  `json:"wav_buffer"`
}

var (
	mongoClient     *mongo.Client
	sensorsColl     *mongo.Collection
	analysisColl    *mongo.Collection
	rabbitMQChannel *amqp.Channel
)

const (
	mongoURI      = "mongodb://0.0.0.0:27017"
	databaseName  = "audio_app"
	sensorsCollN  = "sensors"
	analysisCollN = "analysis_records"

	rabbitMQURL   = "amqp://guest:guest@0.0.0.0:5672/"
	rabbitMQQueue = "analysis_queue"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("[FATAL] MongoDB connect error: %v", err)
	}
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("[FATAL] MongoDB ping error: %v", err)
	}
	log.Println("[Mongo] Connected to", mongoURI)

	sensorsColl = mongoClient.Database(databaseName).Collection(sensorsCollN)
	analysisColl = mongoClient.Database(databaseName).Collection(analysisCollN)

	rabbitConn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("[FATAL] RabbitMQ dial error: %v", err)
	}
	ch, err := rabbitConn.Channel()
	if err != nil {
		log.Fatalf("[FATAL] RabbitMQ channel error: %v", err)
	}
	rabbitMQChannel = ch

	_, err = rabbitMQChannel.QueueDeclare(
		rabbitMQQueue, // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Fatalf("[FATAL] RabbitMQ queue declare error: %v", err)
	}
	log.Printf("[RabbitMQ] Connected, queue %q declared", rabbitMQQueue)

	go consumeAnalysisMessages()

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("[FATAL] gRPC listen error: %v", err)
	}
	grpcServer := grpc.NewGrpcServer(sensorsColl)
	go func() {
		log.Println("[gRPC] GatewayService listening on 0.0.0.0:50052")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[FATAL] gRPC serve error: %v", err)
		}
	}()

	e := echo.New()
	e.Use(middleware.CORS())

	e.GET("/sensors", getSensorsHandler)
	e.GET("/data", getDataHandler)
	e.GET("/data/latest", getLatestPerSensorHandler)
	e.POST("/sensors", createSensorHandler)

	e.Static("/recordings", "./recordings")

	e.Logger.Fatal(e.Start(":8080"))
}

func consumeAnalysisMessages() {
	msgs, err := rabbitMQChannel.Consume(
		rabbitMQQueue, // queue
		"",            // consumer tag (empty → let server pick)
		false,         // autoAck = false (we’ll ack manually)
		false,         // exclusive
		false,         // noLocal
		false,         // noWait
		nil,           // args
	)
	if err != nil {
		log.Fatalf("[FATAL] RabbitMQ consume error: %v", err)
	}

	log.Printf("[RabbitMQ] Waiting for messages on %q ...", rabbitMQQueue)
	for d := range msgs {
		var msg messageFromQueue
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("[WARN] Failed to unmarshal RabbitMQ message: %v\n  Raw: %s", err, string(d.Body))
			d.Ack(false)
			continue
		}
		fmt.Printf("[RabbitMQ] Recieved: %v", msg)

		ts := time.Now().Unix()
		filename := fmt.Sprintf("%d_%d.wav", msg.SensorID, ts)
		filepath := filepath.Join("recordings", filename)
		record := AnalysisRecord{
			SensorID:  msg.SensorID,
			Coeff:     msg.Coeff,
			Lag:       msg.Lag,
			Timestamp: time.Now().UTC(),
		}
		if msg.WavBuffer != "" {
			wavBytes, err := base64.StdEncoding.DecodeString(msg.WavBuffer)
			if err != nil {
				log.Printf("[ERROR] Base64 decode failed for sensor %d: %v", msg.SensorID, err)
			} else {

				if err := os.WriteFile(filepath, wavBytes, 0644); err != nil {
					log.Printf("[ERROR] Failed to write WAV file for sensor %d: %v", msg.SensorID, err)
				} else {
					log.Printf("[FS] Wrote WAV file: %s", filepath)
					record.Recording = filepath
				}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := analysisColl.InsertOne(ctx, record)
		cancel()
		if err != nil {
			log.Printf("[ERROR] MongoDB insert analysis record error: %v", err)
		} else {
			log.Printf("[Mongo] Inserted analysis: sensor_id=%d, coeff=%.4f, lag=%.4f",
				record.SensorID, record.Coeff, record.Lag)
		}

		d.Ack(false)
	}
}

func getLatestPerSensorHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{
			{Key: "sensor_id", Value: 1},
			{Key: "timestamp", Value: -1},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$sensor_id"},
			{Key: "sensor_id", Value: bson.D{{Key: "$first", Value: "$sensor_id"}}},
			{Key: "coeff", Value: bson.D{{Key: "$first", Value: "$coeff"}}},
			{Key: "lag", Value: bson.D{{Key: "$first", Value: "$lag"}}},
			{Key: "timestamp", Value: bson.D{{Key: "$first", Value: "$timestamp"}}},
			{Key: "recording_path", Value: bson.D{{Key: "$first", Value: "$recording_path"}}},
		}}},
	}

	cursor, err := analysisColl.Aggregate(ctx, pipeline)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("MongoDB aggregation error: %v", err))
	}
	defer cursor.Close(ctx)

	latest := []AnalysisRecordAgg{}
	if err := cursor.All(ctx, &latest); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Cursor decode error: %v", err))
	}

	return c.JSON(http.StatusOK, latest)
}

func createSensorHandler(c echo.Context) error {
	var s Sensor
	if err := c.Bind(&s); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid JSON body: %v", err))
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	filter := bson.M{"sensor_id": s.SensorID}
	update := bson.M{"$set": bson.M{
		"name":     s.Name,
		"location": s.Location,
		"status":   "offline", // default status on create/update
	}}
	opts := options.Update().SetUpsert(true)

	if _, err := sensorsColl.UpdateOne(ctx, filter, update, opts); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("MongoDB upsert sensor error: %v", err))
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": fmt.Sprintf("Sensor %d created/updated", s.SensorID),
	})
}

func getSensorsHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	cursor, err := sensorsColl.Find(ctx, bson.M{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("MongoDB find error: %v", err))
	}
	defer cursor.Close(ctx)

	sensors := []Sensor{}
	if err := cursor.All(ctx, &sensors); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Cursor decode error: %v", err))
	}

	return c.JSON(http.StatusOK, sensors)
}

func getDataHandler(c echo.Context) error {
	pageStr := c.QueryParam("page")
	sizeStr := c.QueryParam("pageSize")

	page := 1
	pageSize := 50
	var err error

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid 'page' parameter")
		}
	}
	if sizeStr != "" {
		pageSize, err = strconv.Atoi(sizeStr)
		if err != nil || pageSize < 1 || pageSize > 1000 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid 'pageSize' parameter (must be between 1 and 1000)")
		}
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	totalCount, err := analysisColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("MongoDB count error: %v", err))
	}

	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := analysisColl.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("MongoDB find error: %v", err))
	}
	defer cursor.Close(ctx)

	records := []AnalysisRecord{}
	if err := cursor.All(ctx, &records); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Cursor decode error: %v", err))
	}

	resp := struct {
		TotalCount int64            `json:"totalCount"`
		Page       int              `json:"page"`
		PageSize   int              `json:"pageSize"`
		Records    []AnalysisRecord `json:"records"`
	}{
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		Records:    records,
	}

	return c.JSON(http.StatusOK, resp)
}
