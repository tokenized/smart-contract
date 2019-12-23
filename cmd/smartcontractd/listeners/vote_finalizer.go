package listeners

import (
	"bytes"
	"context"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"

	"github.com/tokenized/specification/dist/golang/protocol"
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
	node.Log(ctx, "Finalizing vote : %s", vf.voteTx.Hash.String())
	vf.handler.Reprocess(ctx, vf.voteTx)
	vf.finished = true
}

// IsComplete returns true when a job should be removed from the scheduler.
func (vf *VoteFinalizer) IsComplete(ctx context.Context) bool {
	return vf.finished
}

// Equal returns true if another job matches it. Used to cancel jobs.
func (vf *VoteFinalizer) Equal(other scheduler.Job) bool {
	otherVF, ok := other.(*VoteFinalizer)
	if !ok {
		return false
	}
	return bytes.Equal(vf.voteTx.Hash[:], otherVF.voteTx.Hash[:])
}
