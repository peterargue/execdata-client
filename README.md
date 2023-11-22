# execdata-client

This is a PoC for using the Execution State Streaming API. There are examples under `examples/` for each of the endpoints.

## Usage

This will connect to `access-001.devnet49.nodes.onflow.org:9000` and stream all `FlowToken` events.
```
go run cmd/demo/*.go --host access-001.devnet49.nodes.onflow.org:9000
```