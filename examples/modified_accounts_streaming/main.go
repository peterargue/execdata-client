package main

import (
	"context"
	"fmt"
	"log"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/executiondatasync/execution_data"

	"github.com/peterargue/execdata-client/client"
)

// This app demonstrates how to use the Execution Data API to stream BlockExecutionData.
// It uses the execution data to get a list of accounts that were modified during the block.

const (
	accessURL = "access-001.devnet49.nodes.onflow.org:9000"
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

	sub, err := execClient.SubscribeExecutionData(ctx, flow.ZeroID, 0)
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

			accounts, err := getModifiedAccounts(response.ExecutionData)
			if err != nil {
				log.Fatalf("failed to get execution data: %v", err)
			}

			log.Printf("modified accounts: %d", len(accounts))
		}
	}
}

func getModifiedAccounts(executionData *execution_data.BlockExecutionData) ([]flow.Address, error) {
	accounts := map[flow.Address]struct{}{}
	for _, chunk := range executionData.ChunkExecutionDatas {
		if chunk.TrieUpdate == nil {
			continue
		}
		for _, payload := range chunk.TrieUpdate.Payloads {
			key, err := payload.Key()
			if err != nil {
				return nil, fmt.Errorf("could not get key: %w", err)
			}

			address := flow.BytesToAddress(key.KeyParts[0].Value)
			accounts[address] = struct{}{}
		}
	}

	addresses := make([]flow.Address, 0, len(accounts))
	for address := range accounts {
		addresses = append(addresses, address)
	}

	return addresses, nil
}
