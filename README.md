# execdata-client

This is a PoC for using the Execution State Streaming API. There are examples under `examples/` for each of the endpoints.

## Usage

This will connect to `access-003.devnet43.nodes.onflow.org` and stream all `FlowToken` events.
```
go run examples/events_streaming/*.go --host access-003.devnet43.nodes.onflow.org:9000
```