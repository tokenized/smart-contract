package listeners

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/tokenized/inspector"
	"github.com/tokenized/pkg/scheduler"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"

	"github.com/tokenized/specification/dist/golang/protocol"
)

// TransferTimeout is a Scheduler job that rejects a multi-contract transfer if all contracts don't
//   approve or reject within a specified time.
type TransferTimeout struct {
	handler    protomux.Handler
	transferTx *inspector.Transaction
	expiration protocol.Timestamp
	finished   bool
	lock       sync.Mutex
}

func NewTransferTimeout(handler protomux.Handler, transferTx *inspector.Transaction,
	expiration protocol.Timestamp) *TransferTimeout {

	result := TransferTimeout{
		handler:    handler,
		transferTx: transferTx,
		expiration: expiration,
	}
	return &result
}

// IsReady returns true when a job should be executed.
func (tt *TransferTimeout) IsReady(ctx context.Context) bool {
	tt.lock.Lock()
	defer tt.lock.Unlock()

	return uint64(time.Now().UnixNano()) > tt.expiration.Nano()
}

// Run executes the job.
func (tt *TransferTimeout) Run(ctx context.Context) {
	tt.lock.Lock()
	defer tt.lock.Unlock()

	node.Log(ctx, "Timing out transfer : %s", tt.transferTx.Hash.String())
	tt.handler.Reprocess(ctx, tt.transferTx)
	tt.finished = true
}

// IsComplete returns true when a job should be removed from the scheduler.
func (tt *TransferTimeout) IsComplete(ctx context.Context) bool {
	tt.lock.Lock()
	defer tt.lock.Unlock()

	return tt.finished
}

// Equal returns true if another job matches it. Used to cancel jobs.
func (tt *TransferTimeout) Equal(other scheduler.Task) bool {
	tt.lock.Lock()
	defer tt.lock.Unlock()

	otherTT, ok := other.(*TransferTimeout)
	if !ok {
		return false
	}

	otherTT.lock.Lock()
	defer otherTT.lock.Unlock()

	return bytes.Equal(tt.transferTx.Hash[:], otherTT.transferTx.Hash[:])
}
