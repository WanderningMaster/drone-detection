package service

import (
	"log"
	"main/apipb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAnalyzerService(host string) apipb.AnalyzerServiceClient {
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(time.Second*2))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := apipb.NewAnalyzerServiceClient(conn)

	return client
}

func NewGatewayService(host string) apipb.GatewayServiceClient {
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(time.Second*2))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := apipb.NewGatewayServiceClient(conn)

	return client
}
