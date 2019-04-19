package listeners

import (
	"bytes"
	"context"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TransferTimeout is a Scheduler job that rejects a multi-contract transfer if all contracts don't
//   approve or reject within a specified time.
type TransferTimeout struct {
	handler    protomux.Handler
	transferTx *inspector.Transaction
	expiration protocol.Timestamp
	finished   bool
}

func NewTransferTimeout(handler protomux.Handler, transferTx *inspector.Transaction, expiration protocol.Timestamp) *TransferTimeout {
	result := TransferTimeout{
		handler:    handler,
		transferTx: transferTx,
		expiration: expiration,
	}
	return &result
}

// IsReady returns true when a job should be executed.
func (tt *TransferTimeout) IsReady(ctx context.Context) bool {
	return uint64(time.Now().UnixNano()) > tt.expiration.Nano()
}

// Run executes the job.
func (tt *TransferTimeout) Run(ctx context.Context) {
	tt.handler.Trigger(ctx, protomux.REPROCESS, tt.transferTx)
	tt.finished = true
}

// IsComplete returns true when a job should be removed from the scheduler.
func (tt *TransferTimeout) IsComplete(ctx context.Context) bool {
	return tt.finished
}

// Equal returns true if another job matches it. Used to cancel jobs.
func (tt *TransferTimeout) Equal(other scheduler.Job) bool {
	otherTT, ok := other.(*TransferTimeout)
	if !ok {
		return false
	}
	return bytes.Equal(tt.transferTx.Hash[:], otherTT.transferTx.Hash[:])
}
