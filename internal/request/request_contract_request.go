package request

import (
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

type contractRequest struct {
	tx        *wire.MsgTx
	hash      chainhash.Hash
	senders   []btcutil.Address
	receivers []txbuilder.TxOutput
	contract  contract.Contract
	m         protocol.OpReturnMessage
}
