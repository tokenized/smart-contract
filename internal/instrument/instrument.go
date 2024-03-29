package instrument

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Instrument not found")
)

// Retrieve gets the specified instrument from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20) (*state.Instrument, error) {

	ctx, span := trace.StartSpan(ctx, "internal.instrument.Retrieve")
	defer span.End()

	// Find instrument in storage
	a, err := Fetch(ctx, dbConn, contractAddress, instrumentCode)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Create the instrument
func Create(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20, nu *NewInstrument, now protocol.Timestamp) error {

	ctx, span := trace.StartSpan(ctx, "internal.instrument.Create")
	defer span.End()

	// Set up instrument
	var a state.Instrument

	// Get current state
	err := node.Convert(ctx, &nu, &a)
	if err != nil {
		return err
	}

	a.Code = instrumentCode
	a.Revision = 0
	a.CreatedAt = now
	a.UpdatedAt = now

	// a.Holdings = make(map[protocol.PublicKeyHash]*state.Holding)
	// a.Holdings[nu.AdministrationPKH] = &state.Holding{
	// 	PKH:              nu.AdministrationPKH,
	// 	PendingBalance:   nu.TokenQty,
	// 	FinalizedBalance: nu.TokenQty,
	// 	CreatedAt:        a.CreatedAt,
	// 	UpdatedAt:        a.UpdatedAt,
	// }

	if a.InstrumentPayload == nil {
		a.InstrumentPayload = []byte{}
	}

	return Save(ctx, dbConn, contractAddress, &a)
}

// Update the instrument
func Update(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	instrumentCode *bitcoin.Hash20, upd *UpdateInstrument, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.instrument.Update")
	defer span.End()

	// Find instrument
	a, err := Fetch(ctx, dbConn, contractAddress, instrumentCode)
	if err != nil {
		return ErrNotFound
	}

	// Update fields
	if upd.Revision != nil {
		a.Revision = *upd.Revision
	}
	if upd.Timestamp != nil {
		a.Timestamp = *upd.Timestamp
	}

	if upd.InstrumentPermissions != nil {
		a.InstrumentPermissions = *upd.InstrumentPermissions
	}
	if upd.TradeRestrictions != nil {
		a.TradeRestrictions = *upd.TradeRestrictions
	}
	if upd.EnforcementOrdersPermitted != nil {
		a.EnforcementOrdersPermitted = *upd.EnforcementOrdersPermitted
	}
	if upd.VoteMultiplier != nil {
		a.VoteMultiplier = *upd.VoteMultiplier
	}
	if upd.AdministrationProposal != nil {
		a.AdministrationProposal = *upd.AdministrationProposal
	}
	if upd.HolderProposal != nil {
		a.HolderProposal = *upd.HolderProposal
	}
	if upd.InstrumentModificationGovernance != nil {
		a.InstrumentModificationGovernance = *upd.InstrumentModificationGovernance
	}
	if upd.AuthorizedTokenQty != nil {
		a.AuthorizedTokenQty = *upd.AuthorizedTokenQty
	}
	if upd.InstrumentPayload != nil {
		a.InstrumentPayload = *upd.InstrumentPayload
	}
	if upd.FreezePeriod != nil {
		a.FreezePeriod = *upd.FreezePeriod
	}

	a.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, a)
}

// ValidateVoting returns an error if voting is not allowed.
func ValidateVoting(ctx context.Context, as *state.Instrument, initiatorType uint32,
	votingSystem *actions.VotingSystemField) error {

	switch initiatorType {
	case 0: // Administration
		if !as.AdministrationProposal {
			return errors.New("Administration proposals not allowed")
		}
	case 1: // Holder
		if !as.HolderProposal {
			return errors.New("Holder proposals not allowed")
		}
	}

	return nil
}

func timeString(t uint64) string {
	return time.Unix(int64(t)/1000000000, 0).String()
}

// IsTransferable returns an error if the instrument is non-transferable.
func IsTransferable(ctx context.Context, as *state.Instrument, now protocol.Timestamp) error {
	if as.FreezePeriod.Nano() > now.Nano() {
		return node.NewError(actions.RejectionsInstrumentFrozen,
			fmt.Sprintf("Instrument frozen until %s", as.FreezePeriod.String()))
	}

	instrumentData, err := instruments.Deserialize([]byte(as.InstrumentType), as.InstrumentPayload)
	if err != nil {
		return node.NewError(actions.RejectionsMsgMalformed, err.Error())
	}

	switch data := instrumentData.(type) {
	case *instruments.Membership:
		if data.ExpirationTimestamp != 0 && data.ExpirationTimestamp < now.Nano() {
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				fmt.Sprintf("Membership expired at %s", timeString(data.ExpirationTimestamp)))
		}

	case *instruments.ShareCommon:

	case *instruments.CasinoChip:
		if data.ExpirationTimestamp != 0 && data.ExpirationTimestamp < now.Nano() {
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				fmt.Sprintf("CasinoChip expired at %s", timeString(data.ExpirationTimestamp)))
		}

	case *instruments.Coupon:
		if data.ExpirationTimestamp != 0 && data.ExpirationTimestamp < now.Nano() {
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				fmt.Sprintf("Coupon expired at %s", timeString(data.ExpirationTimestamp)))
		}

	case *instruments.LoyaltyPoints:
		if data.ExpirationTimestamp != 0 && data.ExpirationTimestamp < now.Nano() {
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				fmt.Sprintf("LoyaltyPoints expired at %s", timeString(data.ExpirationTimestamp)))
		}

	case *instruments.TicketAdmission:
		if data.EventEndTimestamp != 0 && data.EventEndTimestamp < now.Nano() {
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				fmt.Sprintf("TicketAdmission expired at %s", timeString(data.EventEndTimestamp)))
		}
	}

	return nil
}
