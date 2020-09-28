package listeners

import (
	"context"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
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
		ctx := node.ContextWithLogTrace(ctx, ptx.Itx.Hash.String())

		node.Log(ctx, "Processing tx")

		server.lock.Lock()
		server.Tracer.AddTx(ctx, ptx.Itx.MsgTx)
		server.lock.Unlock()

		server.walletLock.RLock()

		if !ptx.Itx.IsTokenized() {
			node.Log(ctx, "Not tokenized")
			server.utxos.Add(ptx.Itx.MsgTx, server.contractAddresses)
			server.walletLock.RUnlock()
			continue
		}

		if err := server.removeConflictingPending(ctx, ptx.Itx); err != nil {
			node.LogError(ctx, "Failed to remove conflicting pending : %s", err)
			server.walletLock.RUnlock()
			continue
		}

		if ptx.Itx.MsgProto != nil && ptx.Itx.MsgProto.Code() == actions.CodeContractFormation {
			cf := ptx.Itx.MsgProto.(*actions.ContractFormation)
			node.Log(ctx, "Saving contract formation for %s : %s",
				bitcoin.NewAddressFromRawAddress(ptx.Itx.Inputs[0].Address, server.Config.Net),
				ptx.Itx.Hash)
			if err := contract.SaveContractFormation(ctx, server.MasterDB,
				ptx.Itx.Inputs[0].Address, cf, server.Config.IsTest); err != nil {
				node.LogError(ctx, "Failed to save contract formation : %s", err)
			}
		}

		found := false

		// Save tx to cache so it can be used to process the response
		for index, output := range ptx.Itx.Outputs {
			for _, address := range server.contractAddresses {
				if address.Equal(output.Address) {
					found = true
					node.Log(ctx, "Request for contract %s",
						bitcoin.NewAddressFromRawAddress(address, server.Config.Net))
					if err := server.RpcNode.SaveTX(ctx, ptx.Itx.MsgTx); err != nil {
						node.LogError(ctx, "Failed to save tx to RPC : %s", err)
					}
					if !server.IsInSync() && ptx.Itx.IsIncomingMessageType() {
						node.Log(ctx, "Adding request to pending")
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

		// Save pending responses so they can be processed in proper order, which may not be on
		//   chain order.
		if ptx.Itx.IsOutgoingMessageType() {
			responseAdded := false
			for _, input := range ptx.Itx.Inputs {
				for _, address := range server.contractAddresses {
					if address.Equal(input.Address) {
						node.Log(ctx, "Response for contract %s",
							bitcoin.NewAddressFromRawAddress(address, server.Config.Net))
						found = true
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

		if found { // Tx is associated with one of our contracts.
			if server.IsInSync() {
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
				//   data.
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
