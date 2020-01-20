package storage

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
)

const (
	unconfirmedTxSize = bitcoin.Hash32Size + 11
)

var (
	TrueData  = []byte{0xff}
	FalseData = []byte{0x00}
)

// Mark an unconfirmed tx as unsafe
// Returns true if the tx was marked relevant
func (repo *TxRepository) MarkUnsafe(ctx context.Context, txid bitcoin.Hash32) (bool, error) {
	repo.unconfirmedLock.Lock()
	defer repo.unconfirmedLock.Unlock()

	if tx, exists := repo.unconfirmed[txid]; exists {
		tx.unsafe = true
		return true, nil
	}
	return false, nil
}

// Mark an unconfirmed tx as being verified by a trusted node.
// Returns true if the tx was marked relevant
func (repo *TxRepository) MarkTrusted(ctx context.Context, txid bitcoin.Hash32) (bool, error) {
	repo.unconfirmedLock.Lock()
	defer repo.unconfirmedLock.Unlock()

	if tx, exists := repo.unconfirmed[txid]; exists && !tx.trusted {
		logger.Verbose(ctx, "Tx marked trusted : %s", txid.String())
		tx.time = time.Now() // Reset so the "safe" delay is from when the trusted node verified.
		tx.trusted = true
		return true, nil
	}
	return false, nil
}

// Returns all transactions not marked as unsafe or safe that have a "seen" time before the
//   specified time.
// Also marks all returned txs as safe
func (repo *TxRepository) GetNewSafe(ctx context.Context, beforeTime time.Time) ([]bitcoin.Hash32, error) {
	repo.unconfirmedLock.Lock()
	defer repo.unconfirmedLock.Unlock()

	result := make([]bitcoin.Hash32, 0)
	for hash, tx := range repo.unconfirmed {
		if !tx.safe && !tx.unsafe && tx.trusted && tx.time.Before(beforeTime) {
			tx.safe = true
			result = append(result, hash)
		}
	}

	return result, nil
}

type unconfirmedTx struct { // Tx ID hash is key of map containing this struct
	time    time.Time // Time first seen
	unsafe  bool      // Conflict seen
	safe    bool      // Safe notification sent
	trusted bool      // Verified by trusted node
}

func newUnconfirmedTx(trusted, safe bool) *unconfirmedTx {
	result := unconfirmedTx{
		time:    time.Now(),
		unsafe:  false,
		safe:    safe,
		trusted: trusted,
	}
	return &result
}

func (tx *unconfirmedTx) Write(out io.Writer, txid *bitcoin.Hash32) error {
	var err error

	// TxID
	_, err = out.Write(txid[:])
	if err != nil {
		return err
	}

	// Time
	err = binary.Write(out, binary.LittleEndian, int64(tx.time.UnixNano())/1e6) // Milliseconds
	if err != nil {
		return err
	}

	// Unsafe
	if tx.unsafe {
		_, err = out.Write(TrueData[:])
	} else {
		_, err = out.Write(FalseData[:])
	}
	if err != nil {
		return err
	}

	// Safe
	if tx.safe {
		_, err = out.Write(TrueData[:])
	} else {
		_, err = out.Write(FalseData[:])
	}
	if err != nil {
		return err
	}

	// Trusted
	if tx.trusted {
		_, err = out.Write(TrueData[:])
	} else {
		_, err = out.Write(FalseData[:])
	}
	if err != nil {
		return err
	}

	return nil
}

func readUnconfirmedTx(in io.Reader, version uint8) (bitcoin.Hash32, *unconfirmedTx, error) {
	var txid bitcoin.Hash32
	var tx unconfirmedTx
	var err error

	_, err = in.Read(txid[:])
	if err != nil {
		return txid, &tx, err
	}

	// Time
	var milliseconds int64
	err = binary.Read(in, binary.LittleEndian, &milliseconds) // Milliseconds
	if err != nil {
		return txid, &tx, err
	}
	tx.time = time.Unix(0, milliseconds*1e6)

	// Unsafe
	value := []byte{0x00}
	_, err = in.Read(value[:])
	if err != nil {
		return txid, &tx, err
	}
	if value[0] == 0x00 {
		tx.unsafe = false
	} else {
		tx.unsafe = true
	}

	// Safe
	_, err = in.Read(value[:])
	if err != nil {
		return txid, &tx, err
	}
	if value[0] == 0x00 {
		tx.safe = false
	} else {
		tx.safe = true
	}

	// Trusted
	_, err = in.Read(value[:])
	if err != nil {
		return txid, &tx, err
	}
	if value[0] == 0x00 {
		tx.trusted = false
	} else {
		tx.trusted = true
	}

	return txid, &tx, nil
}
