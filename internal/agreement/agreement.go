package agreement

import (
	"context"
	"errors"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/protocol"

	"go.opencensus.io/trace"
)

var (
	ErrNotFound = errors.New("Not Found")
)

// Retrieve gets the specified agreement from the database.
func Retrieve(ctx context.Context, dbConn *db.DB,
	contractAddress bitcoin.RawAddress) (*state.Agreement, error) {

	ctx, span := trace.StartSpan(ctx, "internal.agreement.Retrieve")
	defer span.End()

	// Find agreement in storage
	a, err := Fetch(ctx, dbConn, contractAddress)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Create the agreement
func Create(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	nu *NewAgreement, now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.agreement.Create")
	defer span.End()

	// Set up agreement
	var a state.Agreement

	// Get current state
	err := node.Convert(ctx, &nu, &a)
	if err != nil {
		return err
	}

	a.Revision = 0
	a.CreatedAt = now

	return Save(ctx, dbConn, contractAddress, &a)
}

// Update the agreement
func Update(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	upd *UpdateAgreement, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.agreement.Update")
	defer span.End()

	// Find agreement
	a, err := Fetch(ctx, dbConn, contractAddress)
	if err != nil {
		return ErrNotFound
	}

	// Update fields
	if upd.Chapters != nil {
		a.Chapters = *upd.Chapters
	}
	if upd.Definitions != nil {
		a.Definitions = *upd.Definitions
	}
	if upd.Revision != nil {
		a.Revision = *upd.Revision
	}
	if upd.Timestamp != nil {
		a.Timestamp = *upd.Timestamp
	}

	a.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, a)
}
