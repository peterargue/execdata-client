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

func followBlocks(ctx context.Context, client access.AccessAPIClient, execClient executiondata.ExecutionDataAPIClient) {
	lastHeight := uint64(0)

	// get initial height
	header, err := client.GetLatestBlockHeader(ctx, &access.GetLatestBlockHeaderRequest{IsSealed: true})
	if err != nil {
		log.Fatalf("could not get latest block header: %v", err)
	}

	lastHeight = header.Block.Height

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// get the next block, blocking until it's available
		header, err := client.GetBlockHeaderByHeight(ctx, &access.GetBlockHeaderByHeightRequest{Height: lastHeight + 1})
		if status.Code(err) == codes.NotFound {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err != nil {
			log.Fatalf("could not get block header for height %d: %v", lastHeight+1, err)
		}

		// if header.Block.Height <= lastHeight {
		// 	time.Sleep(500 * time.Millisecond)
		// 	continue
		// }
		lastHeight = header.Block.Height

		log.Printf("%d: %x", header.Block.Height, header.Block.Id)

		var events []flow.Event
		for {
			events, err = getExecutionData(ctx, header.Block.Id, execClient)
			if err != nil {
				if status.Code(err) == codes.NotFound || strings.Contains(err.Error(), "not found") {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				log.Fatalf("failed to get execution data: %v", err)
			}

			break
		}

		for _, event := range events {
			log.Printf("%d %v %d %v", event.TransactionIndex, event.TransactionID, event.EventIndex, event.Type)
		}
	}
}

func getExecutionData(ctx context.Context, blockID []byte, client executiondata.ExecutionDataAPIClient) ([]flow.Event, error) {
	resp, err := client.GetExecutionDataByBlockID(ctx, &executiondata.GetExecutionDataByBlockIDRequest{BlockId: blockID})
	if err != nil {
		return nil, fmt.Errorf("could not get execution data: %w", err)
	}

	log.Printf("got execution data: %v", resp.GetBlockExecutionData())

	events, err := extractEvents(resp.GetBlockExecutionData())
	if err != nil {
		return nil, fmt.Errorf("could not convert execution data: %w", err)
	}

	return events, nil
}

func bytesToID(b []byte) flow.Identifier {
	var id flow.Identifier
	copy(id[:], b)
	return id
}

func extractEvents(m *entities.BlockExecutionData) ([]flow.Event, error) {
	if m == nil {
		return nil, convert.ErrEmptyMessage
	}

	events := []flow.Event{}
	for _, chunk := range m.GetChunkExecutionData() {
		e := convert.MessagesToEvents(chunk.GetEvents())
		events = append(events, e...)
	}

	return events, nil
}
