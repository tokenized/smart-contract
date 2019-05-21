package listeners

import (
	"bytes"
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

// ProcessTxs performs "core" processing on transactions.
func (server *Server) ProcessTxs(ctx context.Context) error {
	for itx := range server.processingTxs.Channel {
		node.Log(ctx, "Processing tx : %s", itx.Hash)
		server.lock.Lock()
		server.Tracer.AddTx(ctx, itx.MsgTx)
		server.lock.Unlock()

		if !itx.IsTokenized() {
			server.utxos.Add(itx.MsgTx, server.contractPKHs)
			continue
		}

		if err := server.removeConflictingPending(ctx, itx); err != nil {
			node.LogError(ctx, "Failed to remove conflicting pending : %s", err)
			continue
		}

		// Save tx to cache so it can be used to process the response
		for _, output := range itx.Outputs {
			for _, pkh := range server.contractPKHs {
				if bytes.Equal(output.Address.ScriptAddress(), pkh) {
					if err := server.RpcNode.SaveTX(ctx, itx.MsgTx); err != nil {
						node.LogError(ctx, "Failed to save tx to RPC : %s", err)
					}
					break
				}
			}
		}

		if err := server.Handler.Trigger(ctx, "SEE", itx); err != nil {
			node.LogError(ctx, "Failed to remove conflicting pending : %s", err)
		}
	}
	return nil
}

type ProcessingTxChannel struct {
	Channel chan *inspector.Transaction
	lock    sync.Mutex
	open    bool
}

func (c *ProcessingTxChannel) Add(tx *inspector.Transaction) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- tx
	return nil
}

func (c *ProcessingTxChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan *inspector.Transaction, count)
	c.open = true
	return nil
}

func (c *ProcessingTxChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
