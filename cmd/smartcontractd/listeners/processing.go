package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

// ProcessTxs performs "core" processing on transactions.
func (server *Server) ProcessTxs(ctx context.Context) error {
	for ptx := range server.processingTxs.Channel {
		ctx = node.ContextWithLogTrace(ctx, ptx.Itx.Hash.String())

		node.Log(ctx, "Processing tx : %s", ptx.Itx.Hash)

		node.LogVerbose(ctx, "Adding tx to tracer : %s", ptx.Itx.Hash)
		server.lock.Lock()
		server.Tracer.AddTx(ctx, ptx.Itx.MsgTx)
		server.lock.Unlock()
		node.LogVerbose(ctx, "Added tx to tracer : %s", ptx.Itx.Hash)

		server.walletLock.RLock()
		node.LogVerbose(ctx, "Wallet read locked : %s", ptx.Itx.Hash)

		if !ptx.Itx.IsTokenized() {
			node.Log(ctx, "Not tokenized : %s", ptx.Itx.Hash)
			server.utxos.Add(ptx.Itx.MsgTx, server.contractAddresses)
			server.walletLock.RUnlock()
			node.LogVerbose(ctx, "Processed tx : %s", ptx.Itx.Hash)
			continue
		}

		if err := server.removeConflictingPending(ctx, ptx.Itx); err != nil {
			node.LogError(ctx, "Failed to remove conflicting pending : %s", err)
			server.walletLock.RUnlock()
			node.LogVerbose(ctx, "Processed tx : %s", ptx.Itx.Hash)
			continue
		}

		if ptx.Itx.MsgProto != nil && ptx.Itx.MsgProto.Code() == actions.CodeContractFormation {
			cf := ptx.Itx.MsgProto.(*actions.ContractFormation)
			if err := contract.SaveContractFormation(ctx, server.MasterDB,
				ptx.Itx.Inputs[0].Address, cf, server.Config.IsTest); err != nil {
				node.LogError(ctx, "Failed to save contract formation : %s", ptx.Itx.Hash.String())
			}
		}

		node.LogVerbose(ctx, "Past save formation : %s", ptx.Itx.Hash)

		found := false

		// Save tx to cache so it can be used to process the response
		for index, output := range ptx.Itx.Outputs {
			for _, address := range server.contractAddresses {
				if address.Equal(output.Address) {
					found = true
					if err := server.RpcNode.SaveTX(ctx, ptx.Itx.MsgTx); err != nil {
						node.LogError(ctx, "Failed to save tx to RPC : %s", err)
					}
					if !server.IsInSync() && ptx.Itx.IsIncomingMessageType() {
						node.Log(ctx, "Request added to pending : %s", ptx.Itx.Hash)
						// Save pending request to ensure it has a response, and process it if not.
						server.pendingRequests = append(server.pendingRequests, pendingRequest{
							Itx:           ptx.Itx,
							ContractIndex: index,
						})
					}
					break
				}
			}
		}

		node.LogVerbose(ctx, "Past inputs : %s", ptx.Itx.Hash)

		// Save pending responses so they can be processed in proper order, which may not be on
		//   chain order.
		if ptx.Itx.IsOutgoingMessageType() {
			responseAdded := false
			for _, input := range ptx.Itx.Inputs {
				for _, address := range server.contractAddresses {
					if address.Equal(input.Address) {
						found = true
						responseAdded = true
						if !server.IsInSync() {
							node.Log(ctx, "Response added to pending : %s", ptx.Itx.Hash)
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

		node.LogVerbose(ctx, "Past outputs : %s", ptx.Itx.Hash)

		server.walletLock.RUnlock()

		if found { // Tx is associated with one of our contracts.
			node.LogVerbose(ctx, "Handling tx : %s", ptx.Itx.Hash)
			if server.IsInSync() {
				// Process this tx
				if err := server.Handler.Trigger(ctx, ptx.Event, ptx.Itx); err != nil {
					node.LogError(ctx, "Failed to handle tx : %s", err)
				}
			} else {
				// Save tx for response processing after smart contract is in sync with on chain
				//   data.
				if err := transactions.AddTx(ctx, server.MasterDB, ptx.Itx); err != nil {
					node.LogError(ctx, "Failed to save tx : %s", err)
				}
			}
			node.LogVerbose(ctx, "Handled tx : %s", ptx.Itx.Hash)
		} else {
			node.LogVerbose(ctx, "Tx not for contracts : %s", ptx.Itx.Hash)
		}

		node.LogVerbose(ctx, "Processed tx : %s", ptx.Itx.Hash)
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
