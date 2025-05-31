package grpc

import (
	"context"
	"fmt"
	"log"
	"main/apipb"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	apipb.UnimplementedGatewayServiceServer
	coll *mongo.Collection
}

func NewGrpcServer(coll *mongo.Collection) *grpc.Server {
	srv := &GrpcServer{
		coll: coll,
	}
	grpcServer := grpc.NewServer()
	apipb.RegisterGatewayServiceServer(grpcServer, srv)

	return grpcServer
}

func (s *GrpcServer) UpdateStatus(ctx context.Context, req *apipb.StatusRequest) (*apipb.StatusResponse, error) {
	sensorID := req.GetSensorId()
	status := req.GetStatus()

	if status != "online" && status != "offline" {
		return &apipb.StatusResponse{Success: false}, fmt.Errorf("invalid status: %s", status)
	}

	filter := bson.M{"sensor_id": sensorID}
	update := bson.M{"$set": bson.M{"status": status}}
	opts := options.Update().SetUpsert(true)

	ctxDB, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := s.coll.UpdateOne(ctxDB, filter, update, opts)
	if err != nil {
		log.Printf("[ERROR][gRPC] UpdateStatus failed: %v", err)
		return &apipb.StatusResponse{Success: false}, err
	}

	if res.MatchedCount == 0 && res.UpsertedCount == 1 {
		log.Printf("[gRPC] Status upserted for new sensor %d → %s", sensorID, status)
	} else {
		log.Printf("[gRPC] Status updated for sensor %d → %s", sensorID, status)
	}

	return &apipb.StatusResponse{Success: true}, nil
}
