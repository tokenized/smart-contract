package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/tokenized/smart-contract/pkg/wire"
	"go.opencensus.io/trace"
)

type Transaction struct{}

func (t *Transaction) Process(ctx context.Context, log *log.Logger, m wire.Message) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transaction.Process")
	defer span.End()

	// Validate and cast message to type
	tx, ok := m.(*wire.MsgTx)
	if !ok {
		return errors.New("Could not assert as *wire.MsgTx")
	}

	log.Printf("%s : Received transaction: %+v\n", v.TraceID, tx.TxHash())
}
