package filters

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
)

// Filters for transactions with tokenized.com op return scripts.
type TxFilter struct {
	chainParams  *chaincfg.Params
	contractPKHs [][]byte
	tracer       *listeners.Tracer
	isTest       bool
}

func NewTxFilter(chainParams *chaincfg.Params, contractPKHs [][]byte, tracer *listeners.Tracer, isTest bool) *TxFilter {
	result := TxFilter{
		chainParams:  chainParams,
		contractPKHs: contractPKHs,
		tracer:       tracer,
		isTest:       isTest,
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
		for _, cpkh := range filter.contractPKHs {
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
		pkh, err := txbuilder.PubKeyHashFromP2PKHSigScript(input.SignatureScript)
		if err != nil {
			continue
		}

		for _, cpkh := range filter.contractPKHs {
			if bytes.Equal(pkh, cpkh) {
				logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
					"Matches PaymentFromContract : %s", tx.TxHash().String())
				return true
			}
		}
	}

	return false
}
