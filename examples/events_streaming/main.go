package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/onflow/flow-go/model/flow"

	"github.com/peterargue/execdata-client/client"
)

// This app demonstrates how to use the Execution Data API to stream Events.
// It requests all events from the testnet FlowToken contract.

type Tracker struct {
	execClient *client.ExecutionDataClient
}

const (
	mainnetFlowToken = "A.1654653399040a61.FlowToken"
	devnetFlowToken  = "A.7e60df042a9c0868.FlowToken"
	canaryFlowToken  = "A.0ae53cb6e3f42a79.FlowToken"
)

func main() {
	var accessURL string
	flag.StringVar(&accessURL, "host", "access-003.devnet43.nodes.onflow.org:9000", "execution data api url.")
	flag.Parse()

	filterContract := ""
	switch {
	case strings.Contains(accessURL, "mainnet"):
		filterContract = mainnetFlowToken
	case strings.Contains(accessURL, "devnet"):
		filterContract = devnetFlowToken
	case strings.Contains(accessURL, "canary"):
		filterContract = canaryFlowToken
	default:
		log.Fatal("could not determine FlowToken contract for host's network")
	}

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

	err = t.FollowBlocks(ctx, filterContract)
	if err != nil {
		log.Fatalf("could not follow blocks: %v", err)
	}
}

func (t *Tracker) FollowBlocks(ctx context.Context, filterContract string) error {

	sub, err := t.execClient.SubscribeEvents(ctx, flow.ZeroID, 0, client.EventFilter{
		Contracts: []string{filterContract},
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
