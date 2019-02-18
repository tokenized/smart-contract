package request

import (
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type contractRequest struct {
	tx        *wire.MsgTx
	hash      chainhash.Hash
	senders   []inspector.Input
	receivers []inspector.Output
	contract  contract.Contract
	m         protocol.OpReturnMessage
}
