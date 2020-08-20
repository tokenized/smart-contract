package filters

import (
	"bytes"
	"context"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// Filters for transactions with tokenized.com op return scripts.
type TxFilter struct {
	pubkeys [][]byte
	pkhs    [][]byte
	tracer  *Tracer
	isTest  bool
	lock    sync.RWMutex
}

func NewTxFilter(tracer *Tracer, isTest bool) *TxFilter {
	return &TxFilter{
		tracer: tracer,
		isTest: isTest,
	}
}

func (filter *TxFilter) AddPubKey(ctx context.Context, contractPubKey []byte) {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	filter.pubkeys = append(filter.pubkeys, contractPubKey)
	filter.pkhs = append(filter.pkhs, bitcoin.Hash160(contractPubKey))
}

func (filter *TxFilter) RemovePubKey(ctx context.Context, contractPubKey []byte) {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	for i, pubkey := range filter.pubkeys {
		if bytes.Equal(pubkey, contractPubKey) {
			filter.pubkeys = append(filter.pubkeys[:i], filter.pubkeys[i+1:]...)
			break
		}
	}

	contactPKH := bitcoin.Hash160(contractPubKey)
	for i, pkh := range filter.pkhs {
		if bytes.Equal(contactPKH, pkh) {
			filter.pkhs = append(filter.pkhs[:i], filter.pkhs[i+1:]...)
			break
		}
	}
}

func (filter *TxFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
	filter.lock.RLock()
	defer filter.lock.RUnlock()

	if filter.tracer.Contains(ctx, tx) {
		logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 4,
			"Matches Tracer : %s", tx.TxHash().String())
		return true
	}

	// Check if relevant to contract
	for _, output := range tx.TxOut {
		// Check for C2
		action, err := protocol.Deserialize(output.PkScript, filter.isTest)
		if err == nil && action.Code() == actions.CodeContractFormation {
			return true
		}

		// Check for filtered pub keys
		address, err := bitcoin.RawAddressFromLockingScript(output.PkScript)
		if err != nil {
			continue
		}
		hash, err := address.Hash()
		if err != nil {
			continue
		}
		for _, pkh := range filter.pkhs {
			if bytes.Equal(hash.Bytes(), pkh) {
				logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 4,
					"Matches PKH : %s", tx.TxHash().String())
				return true
			}
		}
	}

	// Check if txin is from contract
	// Reject responses don't go to the contract. They are from contract to request sender.
	for _, input := range tx.TxIn {
		pk, err := bitcoin.PublicKeyFromUnlockingScript(input.SignatureScript)
		if err != nil {
			continue
		}

		for _, cpk := range filter.pubkeys {
			if bytes.Equal(pk, cpk) {
				logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 4,
					"Matches Pub Key : %s", tx.TxHash().String())
				return true
			}
		}
	}

	return false
}
