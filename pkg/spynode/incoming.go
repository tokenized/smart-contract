package spynode

import (
	"context"

	"bitbucket.org/tokenized/nexus-api/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// Process an inbound message
func handleMessage(ctx context.Context, msgHandlers map[string]handlers.CommandHandler, msg wire.Message, outgoing chan wire.Message) error {
	h, ok := msgHandlers[msg.Command()]
	if !ok {
		// no handler for this command
		return nil
	}

	responses, err := h.Handle(ctx, msg)
	if err != nil {
		return err
	}

	// Queue messages to be sent in response
	for _, response := range responses {
		outgoing <- response
	}

	return nil
}
