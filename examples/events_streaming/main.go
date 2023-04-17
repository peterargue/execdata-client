package main

import (
	"context"
	"log"

	"github.com/onflow/flow-go/model/flow"

	"github.com/peterargue/execdata-client/client"
)

// This app demonstrates how to use the Execution Data API to stream Events.
// It requests all events from the testnet FlowToken contract.

const (
	accessURL = "access-003.devnet43.nodes.onflow.org:9000"
)

func main() {
	ctx := context.Background()

	chain, err := client.GetChain(ctx, accessURL)
	if err != nil {
		log.Fatalf("could not get chain: %v", err)
	}

	execClient, err := client.NewExecutionDataClient(accessURL, chain)
	if err != nil {
		log.Fatalf("could not create execution data client: %v", err)
	}

	sub, err := execClient.SubscribeEvents(ctx, flow.ZeroID, 0, client.EventFilter{
		Contracts: []string{"A.7e60df042a9c0868.FlowToken"},
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

			log.Printf("block %d %s:", response.Height, response.BlockID)
			for _, event := range response.Events {
				log.Printf("  %s", event.Type)
			}
		}
	}
}
