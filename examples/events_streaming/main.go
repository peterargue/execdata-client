package main

import (
	"context"
	"fmt"
	"log"

	"github.com/onflow/flow-go/model/flow"

	"github.com/peterargue/execdata-client/client"
)

// This app demonstrates how to use the Execution Data API to stream Events.
// It requests all events from the testnet FlowToken contract.

const (
	accessURL = "access-003.devnet43.nodes.onflow.org:9000"
)

type Tracker struct {
	execClient *client.ExecutionDataClient
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chain, err := client.GetChain(ctx, accessURL)
	if err != nil {
		log.Fatalf("could not get chain: %v", err)
	}

	execClient, err := client.NewExecutionDataClient(accessURL, chain)
	if err != nil {
		log.Fatalf("could not create execution data client: %v", err)
	}

	t := &Tracker{
		execClient: execClient,
	}

	err = t.FollowBlocks(ctx)
	if err != nil {
		log.Fatalf("could not follow blocks: %v", err)
	}
}

func (t *Tracker) FollowBlocks(ctx context.Context) error {

	sub, err := t.execClient.SubscribeEvents(ctx, flow.ZeroID, 0, client.EventFilter{
		Contracts: []string{"A.7e60df042a9c0868.FlowToken"},
	})
	if err != nil {
		return fmt.Errorf("could not subscribe to execution data: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case response, ok := <-sub.Channel():
			if sub.Err() != nil {
				return fmt.Errorf("error in subscription: %w", sub.Err())
			}
			if !ok {
				return fmt.Errorf("subscription closed")
			}

			log.Printf("block %d %s:", response.Height, response.BlockID)
			for _, event := range response.Events {
				log.Printf("  %s", event.Type)
			}
		}
	}
}
