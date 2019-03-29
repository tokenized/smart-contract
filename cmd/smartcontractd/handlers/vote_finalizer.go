package handlers

import (
	"context"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// VoteFinalizer is a Scheduler job that compiles the vote result when the vote expires.
type VoteFinalizer struct {
	handler    protomux.Handler
	voteTx     *inspector.Transaction
	expiration protocol.Timestamp
	finished   bool
}

func NewVoteFinalizer(handler protomux.Handler, voteTx *inspector.Transaction, expiration protocol.Timestamp) *VoteFinalizer {
	result := VoteFinalizer{
		handler:    handler,
		voteTx:     voteTx,
		expiration: expiration,
	}
	return &result
}

// IsReady returns true when a job should be executed.
func (vf *VoteFinalizer) IsReady(ctx context.Context) bool {
	return uint64(time.Now().UnixNano()) > vf.expiration.Nano()
}

// Run executes the job.
func (vf *VoteFinalizer) Run(ctx context.Context) {
	vf.handler.Trigger(ctx, protomux.REPROCESS, vf.voteTx)
	vf.finished = true
}

// IsComplete returns true when a job should be removed from the scheduler.
func (vf *VoteFinalizer) IsComplete(ctx context.Context) bool {
	return vf.finished
}
