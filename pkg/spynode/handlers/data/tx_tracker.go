package data

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/tokenized/nexus-api/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	txids map[chainhash.Hash]time.Time
	mutex sync.Mutex
}

func NewTxTracker() *TxTracker {
	result := TxTracker{
		txids: make(map[chainhash.Hash]time.Time),
	}

	return &result
}

// Adds a txid to tracker to be monitored for expired requests
func (tracker *TxTracker) Add(txid *chainhash.Hash) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	tracker.txids[*txid] = time.Now()
}

// Call when a tx is received to cancel tracking
func (tracker *TxTracker) Remove(ctx context.Context, txids []chainhash.Hash) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	// Iterate tracker ids first since that list should be much smaller
	for txid, _ := range tracker.txids {
		for _, removeid := range txids {
			if txid == removeid {
				logger.Log(ctx, logger.Verbose, "Removing tracking for tx : %s", txid.String())
				delete(tracker.txids, txid)
			}
		}
	}
}

// Called periodically to request any txs that have not been received yet
func (tracker *TxTracker) Check(ctx context.Context, mempool *MemPool) ([]wire.Message, error) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	response := []wire.Message{}
	invRequest := wire.NewMsgGetData()
	for txid, _ := range tracker.txids {
		newTxId := txid // Make a copy to ensure the value isn't overwritten by the next iteration
		alreadyHave, shouldRequest := mempool.AddRequest(&newTxId)
		if alreadyHave {
			delete(tracker.txids, txid) // Remove since we have received tx
		} else {
			if shouldRequest {
				logger.Log(ctx, logger.Verbose, "Requesting tx : %s", newTxId.String())
				item := wire.NewInvVect(wire.InvTypeTx, &newTxId)
				// Request
				if err := invRequest.AddInvVect(item); err != nil {
					// Too many requests for one message
					response = append(response, invRequest) // Append full message
					invRequest = wire.NewMsgGetData()       // Start new message

					// Try to add it again
					if err := invRequest.AddInvVect(item); err != nil {
						return response, errors.Wrap(err, "Failed to add tx to get data request")
					} else {
						delete(tracker.txids, txid) // Remove since we requested
					}
				} else {
					delete(tracker.txids, txid) // Remove since we requested
				}
			} // else wait and check again later
		}
	}

	if len(invRequest.InvList) > 0 {
		response = append(response, invRequest)
	}

	return response, nil
}
