package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/inspector"
	"github.com/tokenized/logger"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/transactions"

	"github.com/pkg/errors"
)

// ProcessTxs performs "core" processing on transactions.
func (server *Server) ProcessTxs(ctx context.Context) error {
	for ptx := range server.processingTxs.Channel {
		ctx := node.ContextWithLogTrace(ctx, ptx.Itx.Hash.String())

		node.Log(ctx, "Processing tx")

		server.lock.Lock()
		server.Tracer.AddTx(ctx, ptx.Itx.MsgTx)
		server.lock.Unlock()

		server.walletLock.RLock()

		if !ptx.Itx.IsTokenized() {
			node.Log(ctx, "Not tokenized")
			server.utxos.Add(ptx.Itx.MsgTx, server.contractLockingScripts)
			server.walletLock.RUnlock()
			continue
		}

		if err := server.removePendingRequests(ctx, ptx.Itx); err != nil {
			node.LogError(ctx, "Failed to remove pending requests : %s", err)
			server.walletLock.RUnlock()
			continue
		}

		isRelevant := false

		// Save tx to cache so it can be used to process the response
		for index, txout := range ptx.Itx.MsgTx.TxOut {
			for _, ls := range server.contractLockingScripts {
				if !ls.Equal(txout.LockingScript) {
					continue
				}

				isRelevant = true
				if ptx.Itx.IsRequest() {
					logger.InfoWithFields(ctx, []logger.Field{
						logger.Stringer("contract_address", ls),
					}, "Request for contract")
					if !server.IsInSync() {
						node.Log(ctx, "Adding request to pending")
						// Save pending request to ensure it has a response, and process it if not.
						server.pendingRequests = append(server.pendingRequests, pendingRequest{
							Itx:           ptx.Itx,
							ContractIndex: index,
						})
					}
				}
				break
			}
		}

		// Save pending responses so they can be processed in proper order, which may not be on
		//   chain order.
		if ptx.Itx.IsResponse() {
			responseAdded := false
			for _, input := range ptx.Itx.Inputs {
				for _, ls := range server.contractLockingScripts {
					if ls.Equal(input.LockingScript) {
						logger.InfoWithFields(ctx, []logger.Field{
							logger.Stringer("contract_address", ls),
						}, "Response for contract")
						isRelevant = true
						responseAdded = true
						if !server.IsInSync() {
							node.Log(ctx, "Adding response to pending")
							server.pendingResponses = append(server.pendingResponses, ptx.Itx)
						}
						break
					}
				}
				if responseAdded {
					break
				}
			}
		}

		server.walletLock.RUnlock()

		if isRelevant { // Tx is associated with one of our contracts.
			if server.IsInSync() {
				node.Log(ctx, "Triggering response")
				// Process this tx
				if err := server.Handler.Trigger(ctx, ptx.Event, ptx.Itx); err != nil {
					switch errors.Cause(err) {
					case node.ErrNoResponse, node.ErrRejected, node.ErrInsufficientFunds:
						node.Log(ctx, "Failed to handle tx : %s", err)
					default:
						node.LogError(ctx, "Failed to handle tx : %s", err)
					}
				}
			} else {
				// Save tx for response processing after smart contract is in sync with on chain
				// data.
				if err := transactions.AddTx(ctx, server.MasterDB, ptx.Itx); err != nil {
					node.LogError(ctx, "Failed to save tx : %s", err)
				}
			}
		} else {
			node.LogVerbose(ctx, "Tx not for any contract addresses")
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
