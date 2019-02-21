package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
)

type Block struct{}

func (b *Block) VerifyTransactions(ctx context.Context, log *log.Logger, m wire.Message) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Block.VerifyTransactions")
	defer span.End()

	// Validate and cast message to type
	b, ok := m.(*wire.MsgBlock)
	if !ok {
		return errors.New("Could not assert as *wire.MsgBlock")
	}

	log.Printf("%s : Received block: %+v\n", v.TraceID, b.BlockHash())

	for _, tx := range b.Transactions {

		itx, err := inspector.NewTransactionFromWire(ctx, tx)
		if err != nil || !itx.IsTokenized() {
			return nil
		}

		// Example of how we can process a transaction found in a new block
		// if err := someservice.ProcessMessage(ctx, itx); err != nil {
		// 	log.Printf("%s : Failed to process transaction: %+v\n", v.TraceID, itx.Hash)
		// 	return err
		// }
	}

	return nil
}
