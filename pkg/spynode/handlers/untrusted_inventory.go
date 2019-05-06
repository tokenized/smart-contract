package handlers

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/spynode/handlers/data"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// InvHandler exists to handle the inv command.
type UntrustedInvHandler struct {
	state   *data.UntrustedState
	tracker *data.TxTracker
	memPool *data.MemPool
}

// NewUntrustedInvHandler returns a new UntrustedInvHandler with the given Config.
func NewUntrustedInvHandler(state *data.UntrustedState, tracker *data.TxTracker, memPool *data.MemPool) *UntrustedInvHandler {
	result := UntrustedInvHandler{
		state:   state,
		tracker: tracker,
		memPool: memPool,
	}
	return &result
}

// Handle implements the Handler interface.
func (handler *UntrustedInvHandler) Handle(ctx context.Context, m wire.Message) ([]wire.Message, error) {
	msg, ok := m.(*wire.MsgInv)
	if !ok {
		return nil, errors.New("Could not assert as *wire.Msginv")
	}

	// We don't care about tx announcments until the peer is verified
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
					// Track to ensure previous request is successful and if not, this node can request.
					handler.tracker.Add(&item.Hash)
				}
			}

		// Untrusted nodes don't care about block announcements
		case wire.InvTypeBlock:
		default:
		}
	}

	if len(invRequest.InvList) > 0 {
		response = append(response, invRequest)
	}

	return response, nil
}
