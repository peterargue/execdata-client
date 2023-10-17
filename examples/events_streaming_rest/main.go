package main

import (
	"context"
	"log"

	"github.com/onflow/flow-go/model/flow"

	"github.com/peterargue/execdata-client/client"
)

// This app demonstrates how to use the Execution Data API to stream Events.
// It requests all events from the localnet FlowToken contract.

const (
	accessURL = "localhost:4004" // REST endpoint on localnet AN1
)

func main() {
	ctx := context.Background()

	restClient, err := client.NewRestClient(accessURL)
	if err != nil {
		log.Fatalf("could not create execution data client: %v", err)
	}

	sub, err := restClient.SubscribeEvents(ctx, flow.ZeroID, 0, client.EventFilter{
		Contracts: []string{"A.0ae53cb6e3f42a79.FlowToken"},
	})
	if err != nil {
		log.Fatalf("could not subscribe to execution data: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case response, ok := <-sub.Channel():
			if sub.Err() != nil {
				log.Fatalf("error in subscription: %v", sub.Err())
			}
			if !ok {
				log.Fatalf("subscription closed")
			}

			if len(response.Events) > 0 {
				log.Printf("block %d %s:", response.Height, response.BlockID)
				for _, event := range response.Events {
					log.Printf("  %s", event.Type)
				}
			}
		}
	}
}
