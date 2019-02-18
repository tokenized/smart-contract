package wallet

import (
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
)

type WalletInterface interface {
	Get(string) (*btcec.PrivateKey, error)
	BuildTX(*btcec.PrivateKey, inspector.UTXOs, []txbuilder.TxOutput, btcutil.Address, protocol.OpReturnMessage) (*wire.MsgTx, error)
}
