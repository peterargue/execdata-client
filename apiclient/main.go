package apiclient

import (
	"context"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/executiondatasync/execution_data"
	executiondata "github.com/onflow/flow/protobuf/go/flow/executiondata"
	"google.golang.org/grpc"
)

type ExecutionDataClient struct {
	client executiondata.ExecutionDataAPIClient
	chain  flow.Chain
}

func NewExecutionDataClient(address string, chain flow.Chain, opts ...grpc.DialOption) (*ExecutionDataClient, error) {
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}

	return &ExecutionDataClient{
		client: executiondata.NewExecutionDataAPIClient(conn),
		chain:  chain,
	}, nil
}

func (c *ExecutionDataClient) GetExecutionDataForBlockID(
	ctx context.Context,
	blockID flow.Identifier,
	opts ...grpc.CallOption,
) (*execution_data.BlockExecutionData, error) {
	req := &executiondata.GetExecutionDataByBlockIDRequest{
		BlockId: blockID[:],
	}
	resp, err := c.client.GetExecutionDataByBlockID(ctx, req, opts...)
	if err != nil {
		return nil, err
	}

	execData, err := convert.MessageToBlockExecutionData(resp.GetBlockExecutionData(), c.chain)
	if err != nil {
		return nil, err
	}

	return execData, nil
}
