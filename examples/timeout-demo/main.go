package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect("127.0.0.1:4222")
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	fmt.Println("NATS service running on 127.0.0.1:4222")
	fmt.Println("Subscribing to subjects:")
	fmt.Println("  api.fast.*      → 1s response")
	fmt.Println("  api.slow.*      → 3s response")
	fmt.Println("  api.very-slow.* → 8s response")
	fmt.Println("  api.custom.*    → 9s response")

	// Subscribe to fast endpoint (1s response)
	nc.Subscribe("api.fast.*", func(msg *nats.Msg) {
		time.Sleep(1 * time.Second)
		msg.Respond([]byte(`{"message": "Fast response in 1s"}`))
	})

	// Subscribe to slow endpoint (3s response)
	nc.Subscribe("api.slow.*", func(msg *nats.Msg) {
		time.Sleep(3 * time.Second)
		msg.Respond([]byte(`{"message": "Slow response in 3s"}`))
	})

	// Subscribe to very slow endpoint (8s response - will timeout with 7s default)
	nc.Subscribe("api.very-slow.*", func(msg *nats.Msg) {
		time.Sleep(8 * time.Second)
		msg.Respond([]byte(`{"message": "Very slow response in 8s"}`))
	})

	// Subscribe to custom endpoint (9s response - will succeed with 9s timeout)
	nc.Subscribe("api.custom.*", func(msg *nats.Msg) {
		time.Sleep(9 * time.Second)
		msg.Respond([]byte(`{"message": "Custom response in 9s"}`))
	})

	// Keep the service running
	select {}
}
