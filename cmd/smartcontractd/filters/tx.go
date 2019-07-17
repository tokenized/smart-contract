package filters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"sync"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/crypto/ripemd160"
)

// Filters for transactions with tokenized.com op return scripts.
type TxFilter struct {
	chainParams *chaincfg.Params
	pubkeys     [][]byte
	pkhs        [][]byte
	tracer      *Tracer
	isTest      bool
	lock        sync.RWMutex
}

func NewTxFilter(chainParams *chaincfg.Params, contractPubKeys []*btcec.PublicKey, tracer *Tracer, isTest bool) *TxFilter {
	result := TxFilter{
		chainParams: chainParams,
		tracer:      tracer,
		isTest:      isTest,
	}

	result.pubkeys = make([][]byte, 0, len(contractPubKeys))
	result.pkhs = make([][]byte, 0, len(contractPubKeys))
	for _, pubkey := range contractPubKeys {
		pubkeyData := pubkey.SerializeCompressed()

		// Hash public key
		hash160 := ripemd160.New()
		hash256 := sha256.Sum256(pubkeyData)
		hash160.Write(hash256[:])

		result.pubkeys = append(result.pubkeys, pubkeyData)
		result.pkhs = append(result.pkhs, hash160.Sum(nil))
	}

	return &result
}

func (filter *TxFilter) AddPubKey(ctx context.Context, contractPubKey *btcec.PublicKey) {
	pubkeyData := contractPubKey.SerializeCompressed()

	// Hash public key
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(pubkeyData)
	hash160.Write(hash256[:])

	filter.lock.Lock()
	defer filter.lock.Unlock()

	filter.pubkeys = append(filter.pubkeys, pubkeyData)
	filter.pkhs = append(filter.pkhs, hash160.Sum(nil))
}

func (filter *TxFilter) RemovePubKey(ctx context.Context, contractPubKey *btcec.PublicKey) {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	pubkeyData := contractPubKey.SerializeCompressed()

	for i, pubkey := range filter.pubkeys {
		if bytes.Equal(pubkey, pubkeyData) {
			filter.pubkeys = append(filter.pubkeys[:i], filter.pubkeys[i+1:]...)
			break
		}
	}

	// Hash public key
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(pubkeyData)
	hash160.Write(hash256[:])
	contactPKH := hash160.Sum(nil)

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
		logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
			"Matches Tracer : %s", tx.TxHash().String())
		return true
	}

	// Check if relevant to contract
	for _, output := range tx.TxOut {
		pkh, err := txbuilder.PubKeyHashFromP2PKH(output.PkScript)
		if err != nil {
			continue
		}
		for _, cpkh := range filter.pkhs {
			if bytes.Equal(pkh, cpkh) {
				logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
					"Matches PaymentToContract : %s", tx.TxHash().String())
				return true
			}
		}
	}

	// Check if txin is from contract
	// Reject responses don't go to the contract. They are from contract to request sender.
	for _, input := range tx.TxIn {
		pk, err := txbuilder.PubKeyFromP2PKHSigScript(input.SignatureScript)
		if err != nil {
			continue
		}

		for _, cpk := range filter.pubkeys {
			if bytes.Equal(pk, cpk) {
				logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
					"Matches PaymentFromContract : %s", tx.TxHash().String())
				return true
			}
		}
	}

	return false
}
