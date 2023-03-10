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

const (
	// Mainnet (must be running an ssh tunnel to an7)
	accessURL        = "access-007.mainnet21.nodes.onflow.org:9000"
	executiondataURL = "localhost:9003"

	// Localnet
	// accessURL        = "localhost:3569"
	// executiondataURL = "localhost:3709"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := grpc.Dial(accessURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to access api server: %v", err)
	}

	accessClient := access.NewAccessAPIClient(conn)

	conn, err = grpc.Dial(
		executiondataURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)),
	)
	if err != nil {
		log.Fatalf("could not connect to exec data api server: %v", err)
	}

	execClient := executiondata.NewExecutionDataAPIClient(conn)

	followBlocks(ctx, accessClient, execClient)
}

func followBlocks(ctx context.Context, accessClient access.AccessAPIClient, execClient executiondata.ExecutionDataAPIClient) {
	resp, err := accessClient.GetNetworkParameters(ctx, &access.GetNetworkParametersRequest{})
	if err != nil {
		log.Fatalf("could not get network parameters: %v", err)
	}
	chain := flow.ChainID(resp.ChainId).Chain()

	// get initial height
	header, err := accessClient.GetLatestBlockHeader(ctx, &access.GetLatestBlockHeaderRequest{IsSealed: true})
	if err != nil {
		log.Fatalf("could not get latest block header: %v", err)
	}

	lastHeight := header.Block.Height

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// get the next block, blocking until it's available
		header, err := accessClient.GetBlockHeaderByHeight(ctx, &access.GetBlockHeaderByHeightRequest{Height: lastHeight + 1})
		if status.Code(err) == codes.NotFound {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err != nil {
			log.Fatalf("could not get block header for height %d: %v", lastHeight+1, err)
		}

		lastHeight = header.Block.Height

		log.Printf("%d: %x", header.Block.Height, header.Block.Id)

		var accounts []flow.Address
		for {
			accounts, err = getModifiedAccounts(ctx, header.Block.Id, execClient, chain)
			if err != nil {
				if status.Code(err) == codes.NotFound || strings.Contains(err.Error(), "not found") {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				log.Fatalf("failed to get execution data: %v", err)
			}

			break
		}

		for _, address := range accounts {
			fmt.Printf("0x%s\n", address)
		}
	}
}

func getModifiedAccounts(ctx context.Context, blockID []byte, client executiondata.ExecutionDataAPIClient, chain flow.Chain) ([]flow.Address, error) {
	resp, err := client.GetExecutionDataByBlockID(ctx, &executiondata.GetExecutionDataByBlockIDRequest{BlockId: blockID})
	if err != nil {
		return nil, fmt.Errorf("could not get execution data: %w", err)
	}

	updates, err := extractTrieUpdates(resp.GetBlockExecutionData(), chain)
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
