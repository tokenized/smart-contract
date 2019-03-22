package vote

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"go.opencensus.io/trace"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Vote not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in its proper form")
)

// Retrieve gets the specified vote from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, voteID *protocol.TxId) (*state.Vote, error) {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Retrieve")
	defer span.End()

	// Find vote in storage
	c, err := Fetch(ctx, dbConn, contractPKH, voteID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return c, nil
}

// Update the vote
func Update(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, voteID *protocol.TxId, upd *UpdateVote, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Update")
	defer span.End()

	// Find vote
	v, err := Fetch(ctx, dbConn, contractPKH, voteID)
	if err != nil {
		return ErrNotFound
	}

	// TODO(srg) New protocol spec - This double up in logic is reserved for using
	// conditional pointers where only some fields are updated on the object.
	// v.VoteName = upd.VoteName
	// v.VotingSystem = upd.VotingSystem

	// if upd.IssuerType == string(0x0) {
	// 	v.IssuerType = ""
	// }

	if err := Save(ctx, dbConn, contractPKH, v); err != nil {
		return err
	}

	return nil
}

// ResultMaximum returns the maximum result
func ResultMaximum(r state.Result) uint64 {
	max := uint64(0)

	for _, v := range r {
		if v >= max {
			max = v
		}
	}

	return max
}
