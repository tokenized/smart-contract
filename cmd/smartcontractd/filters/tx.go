package filters

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
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
	tracer      *listeners.Tracer
	isTest      bool
}

func NewTxFilter(chainParams *chaincfg.Params, contractPubKeys []*btcec.PublicKey, tracer *listeners.Tracer, isTest bool) *TxFilter {
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

func (filter *TxFilter) IsRelevant(ctx context.Context, tx *wire.MsgTx) bool {
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
