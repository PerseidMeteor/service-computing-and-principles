package main

import (
	"context"
	"log"
	"net"

	message "yrpc/pb"

	"google.golang.org/grpc"
)

type server struct {
	*message.UnimplementedMessageSenderServer
}

func (s *server) Send(ctx context.Context, in *message.MessageRequest) (*message.MessageResponse, error) {
	log.Printf("Received: %v", in.Reqsome)
	return &message.MessageResponse{Respsome: "Hello, Client!"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// message.RegisterExampleServiceServer(s, &server{})
	message.RegisterMessageSenderServer(s, &server{})
	log.Println("starting server...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
