package holdings

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Holding not found")

	// ErrInsufficientHoldings occurs when the address doesn't hold enough tokens for the operation.
	ErrInsufficientHoldings = errors.New("Holdings insufficient")

	// ErrHoldingsFrozen occurs when the address holdings are frozen.
	ErrHoldingsFrozen = errors.New("Holdings are frozen")

	// ErrHoldingsLocked occurs when the address holdings are locked for a multi-contract transfer.
	ErrHoldingsLocked = errors.New("Holdings are locked")

	// ErrDuplicateEntry occurs when more than one send or receive is specified for an address.
	ErrDuplicateEntry = errors.New("Holdings duplicate entry")
)

const (
	FreezeCode               = byte('F')
	DebitCode                = byte('S')
	DepositCode              = byte('R')
	MultiContractDebitCode   = byte('-')
	MultiContractDepositCode = byte('+')
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

// AppendBallots adds ballot quantities to the ballot map.
func AppendBallots(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	as *state.Asset, ballots *map[bitcoin.Hash20]state.Ballot, applyMultiplier bool,
	now protocol.Timestamp) error {

	if !as.VotingRights {
		return nil
	}

	holdings, err := FetchAll(ctx, dbConn, contractAddress, as.Code)
	if err != nil {
		return errors.Wrap(err, "fetch all holdings")
	}

	for _, h := range holdings {
		hash, err := h.Address.Hash()
		if err != nil {
			return errors.Wrap(err, "address hash")
		}

		quantity := h.FinalizedBalance
		if applyMultiplier {
			quantity *= uint64(as.VoteMultiplier)
		}

		_, exists := (*ballots)[*hash]
		if exists {
			ballot := (*ballots)[*hash]
			ballot.Quantity += quantity
			(*ballots)[*hash] = ballot
		} else {
			(*ballots)[*hash] = state.Ballot{
				Address:  h.Address,
				Quantity: quantity,
			}
		}
	}

	return nil
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
// When a holding status does not exist to finalize, like when in recovery mode, the balance is just
//   set to the specified balance.
func FinalizeTx(h *state.Holding, txid *protocol.TxId, balance uint64, now protocol.Timestamp) error {
	h.UpdatedAt = now

	hs, exists := h.HoldingStatuses[*txid]
	if !exists {
		// Status may not exist during recovery, but the settlement is valid either way.
		h.FinalizedBalance = balance
		h.PendingBalance = balance
		return nil
	}

	switch hs.Code {
	case DebitCode:
		fallthrough
	case MultiContractDebitCode:
		h.FinalizedBalance -= hs.Amount
		delete(h.HoldingStatuses, *txid)
	case DepositCode:
		fallthrough
	case MultiContractDepositCode:
		h.FinalizedBalance += hs.Amount
		delete(h.HoldingStatuses, *txid)
	default:
		return fmt.Errorf("Unknown holding status code : %c", hs.Code)
	}

	return nil
}

// AddDebit adds a pending send amount to a holding.
func AddDebit(h *state.Holding, txid *protocol.TxId, amount uint64, isSingleContract bool,
	now protocol.Timestamp) error {

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

	// Check if there is a pending multi-contract holding status.
	for _, status := range h.HoldingStatuses {
		if status.Code == MultiContractDepositCode || status.Code == MultiContractDebitCode {
			return ErrHoldingsLocked
		}
	}

	h.PendingBalance -= amount
	h.UpdatedAt = now

	hs := state.HoldingStatus{
		Code:           DebitCode,
		Amount:         amount,
		TxId:           txid,
		SettleQuantity: h.PendingBalance,
	}

	if !isSingleContract {
		hs.Code = MultiContractDebitCode
	}

	h.HoldingStatuses[*txid] = &hs
	return nil
}

// AddDeposit adds a pending receive amount to a holding.
func AddDeposit(h *state.Holding, txid *protocol.TxId, amount uint64, isSingleContract bool,
	now protocol.Timestamp) error {

	_, exists := h.HoldingStatuses[*txid]
	if exists {
		return ErrDuplicateEntry
	}

	// Check if there is a pending multi-contract holding status.
	for _, status := range h.HoldingStatuses {
		if status.Code == MultiContractDepositCode || status.Code == MultiContractDebitCode {
			return ErrHoldingsLocked
		}
	}

	h.PendingBalance += amount
	h.UpdatedAt = now

	hs := state.HoldingStatus{
		Code:           DepositCode,
		Amount:         amount,
		TxId:           txid,
		SettleQuantity: h.PendingBalance,
	}

	if !isSingleContract {
		hs.Code = MultiContractDepositCode
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

	if hs.Code != DebitCode && hs.Code != MultiContractDebitCode {
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

	if hs.Code != DepositCode && hs.Code != MultiContractDepositCode {
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
		fallthrough
	case MultiContractDebitCode:
		h.PendingBalance += hs.Amount
	case DepositCode:
		fallthrough
	case MultiContractDepositCode:
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
