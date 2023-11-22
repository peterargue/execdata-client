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

const (
	mainnetFlowToken = "A.1654653399040a61.FlowToken"
	devnetFlowToken  = "A.7e60df042a9c0868.FlowToken"
	canaryFlowToken  = "A.0ae53cb6e3f42a79.FlowToken"
)

func main() {
	var accessURL,
		filterEvents,
		filterContracts,
		filterAddresses string

	flag.StringVar(&accessURL, "host", "access-001.devnet49.nodes.onflow.org:9000", "execution data api url.")
	flag.StringVar(&filterEvents, "events", "", "comma separated list of events to filter for.")
	flag.StringVar(&filterContracts, "contracts", "", "comma separated list of contracts to filter events by.")
	flag.StringVar(&filterAddresses, "addresses", "", "comma separated list of addresses to filter events by.")
	flag.Parse()

	filter := defaultFilter(accessURL)
	if filterEvents != "" || filterContracts != "" || filterAddresses != "" {
		filter = buildFilter(filterEvents, filterContracts, filterAddresses)
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

	err = followBlocks(ctx, execClient, filter)
	if err != nil {
		log.Fatalf("could not follow blocks: %v", err)
	}
}

func followBlocks(ctx context.Context, execClient *client.ExecutionDataClient, filter client.EventFilter) error {
	sub, err := execClient.SubscribeEvents(ctx, flow.ZeroID, 0, filter)
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

func buildFilter(events, contracts, addresses string) client.EventFilter {
	filter := client.EventFilter{}

	if events != "" {
		filter.EventTypes = strings.Split(events, ",")
	}

	if contracts != "" {
		filter.Contracts = strings.Split(contracts, ",")
	}

	if addresses != "" {
		filter.Addresses = strings.Split(addresses, ",")
	}

	return filter
}

func defaultFilter(accessURL string) client.EventFilter {
	filter := client.EventFilter{}

	switch {
	case strings.Contains(accessURL, "mainnet"):
		filter.Contracts = []string{mainnetFlowToken}

	case strings.Contains(accessURL, "devnet"):
		filter.Contracts = []string{devnetFlowToken}

	case strings.Contains(accessURL, "canary"):
		filter.Contracts = []string{canaryFlowToken}

	default:
		log.Fatal("could not determine FlowToken contract for host's network")
	}

	return filter
}
