package data

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// TxTracker saves txids that have been announced by a specific node, but not requested yet.
// For example, the node announces the tx, but another node has already requested it.
// But if the node it was requested from never actually gives the tx, then it needs to be
//   re-requested from a node that has previously announced it.
// So TxTracker remembers all the txids that we don't have the tx for so we can re-request if
//   necessary.

// When a block is confirmed, its txids are fed back through an interface to the node class which
//   uses that data to call back to all active TxTrackers to remove any txs being tracked that are
//   now confirmed.

type TxTracker struct {
	txids map[bitcoin.Hash32]time.Time
	stop  atomic.Value
	mutex sync.Mutex
}

func NewTxTracker() *TxTracker {
	result := TxTracker{
		txids: make(map[bitcoin.Hash32]time.Time),
	}

	result.stop.Store(false)

	return &result
}

func (tracker *TxTracker) Stop() {
	tracker.stop.Store(true)
}

// Adds a txid to tracker to be monitored for expired requests
func (tracker *TxTracker) Add(txid bitcoin.Hash32) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if _, exists := tracker.txids[txid]; !exists {
		tracker.txids[txid] = time.Now()
	}
}

// Remove removes the tx from the tracker.
func (tracker *TxTracker) Remove(ctx context.Context, txid bitcoin.Hash32) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if _, exists := tracker.txids[txid]; exists {
		delete(tracker.txids, txid)
	}
}

// RemoveList removes all the txs from the tracker.
func (tracker *TxTracker) RemoveList(ctx context.Context, txids []*bitcoin.Hash32) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	for _, removeid := range txids {
		if _, exists := tracker.txids[*removeid]; exists {
			delete(tracker.txids, *removeid)
		}
	}
}

type MessageTransmitter interface {
	TransmitMessage(wire.Message) bool
}

// Called periodically to request any txs that have not been received yet
func (tracker *TxTracker) Check(ctx context.Context, mempool *MemPool, transmitter MessageTransmitter) error {
	val := tracker.stop.Load()
	s, ok := val.(bool)
	if !ok || s {
		return nil
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	invRequest := wire.NewMsgGetData()
	requestCount := 0
	for txid, addedTime := range tracker.txids {
		val := tracker.stop.Load()
		s, ok := val.(bool)
		if !ok || s {
			break
		}

		alreadyHave, shouldRequest := mempool.AddRequest(ctx, txid, false)
		if alreadyHave {
			delete(tracker.txids, txid) // Remove since we have received tx
		} else if shouldRequest {
			logger.Debug(ctx, "Re-Requesting tx (announced %s) : %s",
				addedTime.Format("15:04:05.999999"), txid.String())
			newTxId := txid // Make a copy to ensure the value isn't overwritten by the next iteration
			item := wire.NewInvVect(wire.InvTypeTx, &newTxId)

			// Request
			if err := invRequest.AddInvVect(item); err != nil {
				// Too many requests for one message
				if !transmitter.TransmitMessage(invRequest) {
					break // node stopped
				}
				invRequest = wire.NewMsgGetData() // Start new message

				// Try to add it again
				if err := invRequest.AddInvVect(item); err != nil {
					return errors.Wrap(err, "Failed to add tx to get data request")
				} else {
					requestCount++
					delete(tracker.txids, txid) // Remove since we requested
				}
			} else {
				requestCount++
				delete(tracker.txids, txid) // Remove since we requested
			}

			if requestCount > 100 {
				if !transmitter.TransmitMessage(invRequest) {
					break // node stopped
				}
				invRequest = wire.NewMsgGetData() // Start new message
				requestCount = 0
			}

		} // else wait and check again later
	}

	return nil
}
