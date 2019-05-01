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

var (
	// Tokenized.com OP_RETURN script signature
	// 0x6a <OP_RETURN>
	// 0x0d <Push next 13 bytes>
	// 0x746f6b656e697a65642e636f6d <"tokenized.com">
	tokenizedSignature     = []byte{0x6a, 0x0d, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6d}
	testTokenizedSignature = []byte{0x6a, 0x12, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x2e, 0x63, 0x6f, 0x6d}
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

	containsTokenized := false
	for _, output := range tx.TxOut {
		if IsTokenizedOpReturn(output.PkScript, filter.isTest) {
			logger.LogDepth(logger.ContextWithOutLogSubSystem(ctx), logger.LevelInfo, 3,
				"Matches TokenizedFilter : %s", tx.TxHash().String())
			containsTokenized = true
			break
		}
	}

	if !containsTokenized {
		return false
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

// Checks if a script carries the tokenized.com protocol signature
func IsTokenizedOpReturn(pkScript []byte, isTest bool) bool {
	if isTest {
		if len(pkScript) < len(testTokenizedSignature) {
			return false
		}
		return bytes.Equal(pkScript[:len(testTokenizedSignature)], testTokenizedSignature)
	}

	if len(pkScript) < len(tokenizedSignature) {
		return false
	}
	return bytes.Equal(pkScript[:len(tokenizedSignature)], tokenizedSignature)
}
