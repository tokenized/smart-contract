package holdings

import (
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Holding not found")

	// ErrInsufficientHoldings occurs when the address doesn't hold enough tokens for the operation.
	ErrInsufficientHoldings = errors.New("Holdings insufficient")

	// ErrHoldingsFrozen occurs when the address holdings are frozen.
	ErrHoldingsFrozen = errors.New("Holdings are frozen")

	// ErrDuplicateEntry occurs when more than one send or receive is specified for an address.
	ErrDuplicateEntry = errors.New("Holdings duplicate entry")
)

const (
	FreezeCode  = byte('F')
	DebitCode   = byte('S')
	DepositCode = byte('R')
)

// GetHolding returns the holding data for a PKH.
func GetHolding(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	assetCode *protocol.AssetCode, address bitcoin.RawAddress, now protocol.Timestamp) (*state.Holding, error) {

	result, err := Fetch(ctx, dbConn, contractAddress, assetCode, address)
	if err == nil {
		return result, nil
	}
	if err != nil && err != ErrNotFound {
		return result, err
	}

	result = &state.Holding{
		Address:         address,
		CreatedAt:       now,
		UpdatedAt:       now,
		HoldingStatuses: make(map[protocol.TxId]*state.HoldingStatus),
	}
	return result, nil
}

// VotingBalance returns the balance for a PKH holder
func VotingBalance(as *state.Asset, h *state.Holding, applyMultiplier bool,
	now protocol.Timestamp) uint64 {

	if !as.VotingRights {
		return 0
	}

	if applyMultiplier {
		return h.FinalizedBalance * uint64(as.VoteMultiplier)
	}
	return h.FinalizedBalance
}

func SafeBalance(h *state.Holding) uint64 {
	if h.PendingBalance < h.FinalizedBalance {
		return h.PendingBalance
	}
	return h.FinalizedBalance
}

func UnfrozenBalance(h *state.Holding, now protocol.Timestamp) uint64 {
	result := h.FinalizedBalance
	if h.PendingBalance < h.FinalizedBalance {
		result = h.PendingBalance
	}

	for _, status := range h.HoldingStatuses {
		if status.Code != FreezeCode {
			continue
		}
		if statusExpired(status, now) {
			continue
		}
		if status.Amount > result {
			return 0
		} else {
			result -= status.Amount
		}
	}

	return result
}

// FinalizeTx finalizes any pending changes involved with a tx.
func FinalizeTx(h *state.Holding, txid *protocol.TxId, now protocol.Timestamp) error {
	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		return fmt.Errorf("Missing status to finalize : %s", txid.String())
	}

	h.UpdatedAt = now

	switch hs.Code {
	case DebitCode:
		h.FinalizedBalance -= hs.Amount
		delete(h.HoldingStatuses, *txid)
	case DepositCode:
		h.FinalizedBalance += hs.Amount
		delete(h.HoldingStatuses, *txid)
	default:
		return fmt.Errorf("Unknown holding status code : %c", hs.Code)
	}

	return nil
}

// AddDebit adds a pending send amount to a holding.
func AddDebit(h *state.Holding, txid *protocol.TxId, amount uint64, now protocol.Timestamp) error {
	_, exists := h.HoldingStatuses[*txid]
	if exists {
		return ErrDuplicateEntry
	}

	if SafeBalance(h) < amount {
		return ErrInsufficientHoldings
	}

	if UnfrozenBalance(h, now) < amount {
		return ErrHoldingsFrozen
	}

	h.PendingBalance -= amount
	h.UpdatedAt = now

	hs := state.HoldingStatus{
		Code:           DebitCode,
		Amount:         amount,
		TxId:           txid,
		SettleQuantity: h.PendingBalance,
	}
	h.HoldingStatuses[*txid] = &hs
	return nil
}

// AddDeposit adds a pending receive amount to a holding.
func AddDeposit(h *state.Holding, txid *protocol.TxId, amount uint64, now protocol.Timestamp) error {
	_, exists := h.HoldingStatuses[*txid]
	if exists {
		return ErrDuplicateEntry
	}

	h.PendingBalance += amount
	h.UpdatedAt = now

	hs := state.HoldingStatus{
		Code:           DepositCode,
		Amount:         amount,
		TxId:           txid,
		SettleQuantity: h.PendingBalance,
	}
	h.HoldingStatuses[*txid] = &hs
	return nil
}

// AddFreeze adds a freeze amount to a holding.
func AddFreeze(h *state.Holding, txid *protocol.TxId, amount uint64,
	timeout protocol.Timestamp, now protocol.Timestamp) error {

	_, exists := h.HoldingStatuses[*txid]
	if exists {
		return ErrDuplicateEntry
	}

	h.PendingBalance += amount
	h.UpdatedAt = now

	hs := state.HoldingStatus{
		Code:    FreezeCode, // Freeze
		Expires: timeout,
		Amount:  amount,
		TxId:    txid,
	}
	h.HoldingStatuses[*txid] = &hs
	return nil
}

// CheckDebit checks that the debit amount matches that specified.
func CheckDebit(h *state.Holding, txid *protocol.TxId, amount uint64) (uint64, error) {
	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		return 0, errors.New("Missing settlement")
	}

	if hs.Code != DebitCode {
		return 0, errors.New("Wrong settlement type")
	}

	if hs.Amount != amount {
		return 0, errors.New("Wrong settlement amount")
	}

	return hs.SettleQuantity, nil
}

// CheckDeposit checks that the deposit amount matches that specified
func CheckDeposit(h *state.Holding, txid *protocol.TxId, amount uint64) (uint64, error) {
	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		return 0, errors.New("Missing settlement")
	}

	if hs.Code != DepositCode {
		return 0, errors.New("Wrong settlement type")
	}

	if hs.Amount != amount {
		return 0, errors.New("Wrong settlement amount")
	}

	return hs.SettleQuantity, nil
}

func CheckFreeze(h *state.Holding, txid *protocol.TxId, amount uint64) error {
	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		return fmt.Errorf("Missing freeze : %s", txid.String())
	}

	if hs.Code != FreezeCode {
		return errors.New("Wrong freeze type")
	}

	if hs.Amount != amount {
		return errors.New("Wrong freeze amount")
	}

	return nil
}

// RevertStatus removes a holding status for a specific txid.
func RevertStatus(h *state.Holding, txid *protocol.TxId) error {
	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		return errors.New("Status not found") // No status to revert
	}

	switch hs.Code {
	case DebitCode:
		h.PendingBalance += hs.Amount
	case DepositCode:
		h.PendingBalance -= hs.Amount
	}

	delete(h.HoldingStatuses, *txid)
	return nil
}

// statusExpired checks to see if a holding status has expired
func statusExpired(hs *state.HoldingStatus, now protocol.Timestamp) bool {
	if hs.Expires.Nano() == 0 {
		return false
	}

	// Current time is after expiry, so this order has expired.
	if now.Nano() > hs.Expires.Nano() {
		return true
	}
	return false
}
