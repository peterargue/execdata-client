package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"
	executiondata "github.com/onflow/flow/protobuf/go/flow/executiondata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// This app demonstrates how to use the Execution Data API to poll for BlockExecutionData.
// It uses the execution data to get a list of accounts that were modified during the block.

const (
	accessURL = "access-003.devnet43.nodes.onflow.org:9000"
)

type Tracker struct {
	accessClient access.AccessAPIClient
	execClient   executiondata.ExecutionDataAPIClient

	chain flow.Chain
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := grpc.Dial(accessURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to access api server: %v", err)
	}

	t := &Tracker{
		accessClient: access.NewAccessAPIClient(conn),
		execClient:   executiondata.NewExecutionDataAPIClient(conn),
	}

	// get the network's chainID
	resp, err := t.accessClient.GetNetworkParameters(ctx, &access.GetNetworkParametersRequest{})
	if err != nil {
		log.Fatalf("could not get network parameters: %v", err)
	}
	t.chain = flow.ChainID(resp.ChainId).Chain()

	err = t.FollowBlocks(ctx)
	if err != nil {
		log.Fatalf("could not follow blocks: %v", err)
	}
}

func (t *Tracker) FollowBlocks(ctx context.Context) error {
	// get initial height
	header, err := t.accessClient.GetLatestBlockHeader(ctx, &access.GetLatestBlockHeaderRequest{IsSealed: true})
	if err != nil {
		log.Fatalf("could not get latest block header: %v", err)
	}

	lastHeight := header.Block.Height

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// get the next block, blocking until it's available
		header, err := t.accessClient.GetBlockHeaderByHeight(ctx, &access.GetBlockHeaderByHeightRequest{Height: lastHeight + 1})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			return fmt.Errorf("could not get block header for height %d: %w", lastHeight+1, err)
		}

		lastHeight = header.Block.Height

		log.Printf("%d: %x", header.Block.Height, header.Block.Id)

		var accounts []flow.Address
		for {
			resp, err := t.execClient.GetExecutionDataByBlockID(ctx, &executiondata.GetExecutionDataByBlockIDRequest{BlockId: header.Block.Id})
			if err != nil {
				if status.Code(err) == codes.NotFound || strings.Contains(err.Error(), "not found") {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				return fmt.Errorf("could not get execution data: %w", err)
			}

			accounts, err = getModifiedAccounts(resp.GetBlockExecutionData(), t.chain)
			if err != nil {
				return fmt.Errorf("failed to get execution data: %w", err)
			}

			break
		}

		log.Printf("modified accounts: %d", len(accounts))
		// for _, address := range accounts {
		// 	fmt.Printf("0x%s\n", address)
		// }
		time.Sleep(800 * time.Millisecond)
	}
}

func getModifiedAccounts(executionData *entities.BlockExecutionData, chain flow.Chain) ([]flow.Address, error) {
	updates, err := extractTrieUpdates(executionData, chain)
	if err != nil {
		return nil, fmt.Errorf("could not convert execution data: %w", err)
	}

	accounts := map[flow.Address]struct{}{}
	for _, update := range updates {
		for _, payload := range update.Payloads {
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

func extractTrieUpdates(m *entities.BlockExecutionData, chain flow.Chain) ([]*ledger.TrieUpdate, error) {
	if m == nil {
		return nil, convert.ErrEmptyMessage
	}

	updates := []*ledger.TrieUpdate{}
	for i, c := range m.GetChunkExecutionData() {
		chunk, err := convert.MessageToChunkExecutionData(c, chain)
		if err != nil {
			return nil, fmt.Errorf("could not convert chunk %d: %w", i, err)
		}

		if chunk.TrieUpdate != nil {
			updates = append(updates, chunk.TrieUpdate)
		}
	}

	return updates, nil
}
