package listeners

import (
	"context"
	"log"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Listener exists to handle the spynode messages.
type Listener struct {
	Node    inspector.NodeInterface
	Handler protomux.Handler
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (listener *Listener) Handle(ctx context.Context, msgType int, msgValue interface{}) error {
	switch msgType {
	case handlers.ListenerMsgTx:
		tx, ok := msgValue.(wire.MsgTx)
		if !ok {
			log.Fatalf("Could not assert as wire.MsgTx\n")
			return nil
		}
		log.Printf("Tx : %s\n", tx.TxHash().String())

		// Check if transaction relates to protocol
		itx, err := inspector.NewTransactionFromWire(ctx, &tx)
		if err != nil {
			return err
		}

		// Prefilter out non-protocol messages
		if !itx.IsTokenized() {
			return nil
		}

		// Promote TX
		if err := itx.Promote(ctx, listener.Node); err != nil {
			return err
		}

		return listener.Handler.Trigger(ctx, protomux.SEE, itx)

	case handlers.ListenerMsgTxConfirm:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxConfirm : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx confirm : %s\n", value.String())

	case handlers.ListenerMsgBlock:
		value, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			log.Fatalf("Block : Could not assert as handlers.BlockMessage\n")
			return nil
		}
		log.Printf("New Block (%d) : %s\n", value.Height, value.Hash.String())

	case handlers.ListenerMsgBlockRevert:
		value, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			log.Fatalf("Block Revert : Could not assert as handlers.BlockMessage\n")
			return nil
		}
		log.Printf("Revert Block (%d) : %s\n", value.Height, value.Hash.String())

	case handlers.ListenerMsgTxRevert:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxRevert : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx revert : %s\n", value.String())

	case handlers.ListenerMsgTxCancel:
		value, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxCancel : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx cancel : %s\n", value.String())
	}

	return nil
}
