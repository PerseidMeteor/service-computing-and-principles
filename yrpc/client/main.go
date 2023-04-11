package main

import (
	"context"
	"log"

	message "yrpc/pb"

	"google.golang.org/grpc"
)

func main() {

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	c := message.NewMessageSenderClient(conn)

	r, err := c.Send(context.Background(), &message.MessageRequest{Reqsome: "nihao"})

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetRespsome())

}
