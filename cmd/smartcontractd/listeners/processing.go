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
	for ptx := range server.processingTxs.Channel {
		node.Log(ctx, "Processing tx : %s", ptx.Itx.Hash)
		server.lock.Lock()
		server.Tracer.AddTx(ctx, ptx.Itx.MsgTx)
		server.lock.Unlock()

		server.walletLock.RLock()
		defer server.walletLock.RUnlock()

		if !ptx.Itx.IsTokenized() {
			server.utxos.Add(ptx.Itx.MsgTx, server.contractPKHs)
			continue
		}

		if err := server.removeConflictingPending(ctx, ptx.Itx); err != nil {
			node.LogError(ctx, "Failed to remove conflicting pending : %s", err)
			continue
		}

		// Save tx to cache so it can be used to process the response
		for _, output := range ptx.Itx.Outputs {
			for _, pkh := range server.contractPKHs {
				if bytes.Equal(output.Address.ScriptAddress(), pkh) {
					if err := server.RpcNode.SaveTX(ctx, ptx.Itx.MsgTx); err != nil {
						node.LogError(ctx, "Failed to save tx to RPC : %s", err)
					}
					break
				}
			}
		}

		if err := server.Handler.Trigger(ctx, ptx.Event, ptx.Itx); err != nil {
			node.LogError(ctx, "Failed to handle tx : %s", err)
		}
	}
	return nil
}

type ProcessingTx struct {
	Itx   *inspector.Transaction
	Event string
}

type ProcessingTxChannel struct {
	Channel chan ProcessingTx
	lock    sync.Mutex
	open    bool
}

func (c *ProcessingTxChannel) Add(tx ProcessingTx) error {
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

	c.Channel = make(chan ProcessingTx, count)
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
