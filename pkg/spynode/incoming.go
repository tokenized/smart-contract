package spynode

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// Process an inbound message
func handleMessage(ctx context.Context, msgHandlers map[string]handlers.CommandHandler, msg wire.Message, outgoingLock *sync.Mutex, outgoingOpen *bool, outgoing chan wire.Message) error {
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
		outgoingLock.Lock()
		if !*outgoingOpen {

			outgoingLock.Unlock()
			break
		}
		outgoing <- response
		outgoingLock.Unlock()
	}

	return nil
}
