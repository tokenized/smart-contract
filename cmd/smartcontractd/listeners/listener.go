package listeners

import (
	"context"
	"log"

	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/spynode/handlers"
	"github.com/tokenized/smart-contract/pkg/spynode/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Listener exists to handle the spynode messages.
type Listener struct {
	Node        inspector.NodeInterface
	Handler     protomux.Handler
	blockHeight int // track current block height for confirm messages
}

// Handle implments the Handler interface.
//
// This function handles type conversion and delegates the the concrete
// handler.
func (listener *Listener) Handle(ctx context.Context, msgType int, msgValue interface{}) error {
	ctx = logger.ContextWithOutLogSubSystem(ctx)

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
			log.Printf("Failed to create inspector tx : %s\n", err.Error())
			return err
		}

		// Prefilter out non-protocol messages
		if !itx.IsTokenized() {
			log.Printf("Not tokenized tx : %s\n", tx.TxHash().String())
			return nil
		}

		// Promote TX
		if err := itx.Promote(ctx, listener.Node); err != nil {
			log.Printf("Failed to promote inspector tx : %s\n", err.Error())
			return err
		}

		return listener.Handler.Trigger(ctx, protomux.SEE, itx)

	case handlers.ListenerMsgTxConfirm:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxConfirm : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx confirm : %s\n", txHash.String())
		// TODO Mark as confirmed

	case handlers.ListenerMsgBlock:
		blockData, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			log.Fatalf("Block : Could not assert as handlers.BlockMessage\n")
			return nil
		}
		log.Printf("New Block (%d) : %s\n", blockData.Height, blockData.Hash.String())
		// Remember current block height for confirm messages
		listener.blockHeight = blockData.Height

		// TODO Mark in contract so confirm counts are known

	case handlers.ListenerMsgBlockRevert:
		blockData, ok := msgValue.(handlers.BlockMessage)
		if !ok {
			log.Fatalf("Block Revert : Could not assert as handlers.BlockMessage\n")
			return nil
		}
		log.Printf("Revert Block (%d) : %s\n", blockData.Height, blockData.Hash.String())
		// Remember current block height for confirm messages
		listener.blockHeight = blockData.Height - 1

		// TODO Mark in contract so confirm counts are known

	case handlers.ListenerMsgTxRevert:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxRevert : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx revert : %s\n", txHash.String())

	case handlers.ListenerMsgTxCancel:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxCancel : Could not assert as chainhash.Hash\n")
			return nil
		}
		log.Printf("Tx cancel : %s\n", txHash.String())
		// TODO Undo anything related to this transaction as a conflicting tx has been mined, so it
		//   will never confirm (without a reorg).

	case handlers.ListenerMsgTxUnsafe:
		txHash, ok := msgValue.(chainhash.Hash)
		if !ok {
			log.Fatalf("TxUnsafe : Could not assert as chainhash.Hash\n")
			return nil
		}
		// TODO Pause anything related to this transaction as a conflicting tx has been seen, so it
		//   might never confirm.
		log.Printf("Tx unsafe : %s\n", txHash.String())
	}

	return nil
}
