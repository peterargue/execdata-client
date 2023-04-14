# execdata-client

This is a PoC for using the Execution State Streaming API. There are examples under `cmd/` for each of the endpoints.

## Usage

Note: This assumes you have a SSH tunnel between your local machine and the AN running the State Stream API.

```
go run cmd/events_streaming/*.go
```