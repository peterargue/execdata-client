package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/onflow/flow-go/model/flow"
)

type rawEventsResponse struct {
	BlockID string
	Height  uint64
	Events  []struct {
		Type             string
		TransactionID    string
		TransactionIndex uint32
		EventIndex       uint32
		Payload          string
	}
}

type RestClient struct {
	address string
}

func NewRestClient(address string) (*RestClient, error) {
	return &RestClient{
		address: address,
	}, nil
}

func (c *RestClient) SubscribeEvents(
	ctx context.Context,
	startBlockID flow.Identifier,
	startHeight uint64,
	filter EventFilter,
) (*Subscription[EventsResponse], error) {
	if startBlockID != flow.ZeroID && startHeight > 0 {
		return nil, fmt.Errorf("cannot specify both start block ID and start height")
	}

	url := url.URL{
		Scheme: "ws",
		Host:   c.address,
		Path:   "/v1/subscribe_events",
	}

	query := url.Query()

	if startBlockID != flow.ZeroID {
		query.Set("start_block_id", startBlockID.String())
	}
	if startHeight > 0 {
		query.Set("height", fmt.Sprint(startHeight))
	}
	if len(filter.EventTypes) > 0 {
		query.Set("event_types", strings.Join(filter.EventTypes, ","))
	}
	if len(filter.Addresses) > 0 {
		query.Set("addresses", strings.Join(filter.Addresses, ","))
	}
	if len(filter.Contracts) > 0 {
		query.Set("contracts", strings.Join(filter.Contracts, ","))
	}

	url.RawQuery = query.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return nil, err
	}

	sub := NewSubscription[EventsResponse]()
	go func() {
		defer close(sub.ch)
		defer conn.Close()

		for {
			var resp *rawEventsResponse
			err := conn.ReadJSON(&resp)
			if err == io.EOF {
				return
			}
			if err != nil {
				sub.err = fmt.Errorf("error receiving execution data: %w", err)
				return
			}

			eventsResponse, err := convertEventResponse(resp)
			if err != nil {
				sub.err = err
				return
			}

			sub.ch <- *eventsResponse
		}
	}()

	return sub, nil
}

func convertEventResponse(raw *rawEventsResponse) (*EventsResponse, error) {
	blockID, err := flow.HexStringToIdentifier(raw.BlockID)
	if err != nil {
		return nil, fmt.Errorf("error parsing block ID: %w", err)
	}

	events := make([]flow.Event, 0, len(raw.Events))
	for _, event := range raw.Events {
		txID, err := flow.HexStringToIdentifier(event.TransactionID)
		if err != nil {
			return nil, fmt.Errorf("error parsing transaction ID: %w", err)
		}

		payload, err := base64.StdEncoding.DecodeString(event.Payload)
		if err != nil {
			return nil, fmt.Errorf("error base64 decoding event payload: %w", err)
		}

		events = append(events, flow.Event{
			Type:             flow.EventType(event.Type),
			TransactionID:    txID,
			TransactionIndex: event.TransactionIndex,
			EventIndex:       event.EventIndex,
			Payload:          payload,
		})
	}

	return &EventsResponse{
		Height:  raw.Height,
		BlockID: blockID,
		Events:  events,
	}, nil
}
