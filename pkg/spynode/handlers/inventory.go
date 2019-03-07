package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// InvHandler exists to handle the inv command.
type InvHandler struct {
	state   *data.State
	tracker *data.TxTracker
	memPool *data.MemPool
}

// NewInvHandler returns a new InvHandler
func NewInvHandler(state *data.State, tracker *data.TxTracker, memPool *data.MemPool) *InvHandler {
	result := InvHandler{
		state:   state,
		tracker: tracker,
		memPool: memPool,
	}
	return &result
}

// Handle implements the Handler interface.
func (handler *InvHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgInv)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	// We don't care about tx announcments until we are in sync
	if !handler.state.IsReady() {
		return nil, nil
	}

	response := []wire.Message{}
	invRequest := wire.NewMsgGetData()

	for _, item := range msg.InvList {
		switch item.Type {
		case wire.InvTypeTx:
			alreadyHave, shouldRequest := handler.memPool.AddRequest(&item.Hash)
			if !alreadyHave {
				if shouldRequest {
					// Request
					if err := invRequest.AddInvVect(item); err != nil {
						// Too many requests for one message
						response = append(response, invRequest) // Append full message
						invRequest = wire.NewMsgGetData()       // Start new message

						// Try to add it again
						if err := invRequest.AddInvVect(item); err != nil {
							return response, errors.Wrap(err, "Failed to add tx to get data request")
						}
					}
				} else {
					handler.tracker.Add(&item.Hash)
				}
			}

		// The trusted node shouldn't get block inventories because new blocks will be announced
		//   with headers since we sent a "sendheaders" message.
		case wire.InvTypeBlock:
		default:
		}
	}

	if len(invRequest.InvList) > 0 {
		response = append(response, invRequest)
	}

	return response, nil
}
