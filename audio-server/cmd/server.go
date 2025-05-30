package cmd

import (
	"context"
	"log"
	"main/apipb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func StartServer() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := apipb.NewHealthServiceClient(conn)

	// Prepare a request with a 5-second deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Healthcheck(ctx, &apipb.Payload{SensorId: "sensor-1"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

}
