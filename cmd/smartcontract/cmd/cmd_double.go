package cmd

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/tokenized/smart-contract/cmd/smartcontract/client"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdDoubleSpend = &cobra.Command{
	Use:   "double <N> <receiver addr>",
	Short: "Sends N consecutive transfer requests with double spend tests.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("Incorrect argument count")
		}

		ctx := client.Context()
		if ctx == nil {
			return nil
		}

		count, err := strconv.Atoi(args[0])
		if err != nil {
			logger.Warn(ctx, "Invalid count : %s", err)
			return nil
		}

		theClient, err := client.NewClient(ctx, network(c))
		if err != nil {
			logger.Warn(ctx, "Failed to create client : %s", err)
			return nil
		}

		receiver, err := btcutil.DecodeAddress(args[1], &theClient.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "Invalid address : %s", err)
			return nil
		}
		receiverPKH = receiver.ScriptAddress()

		addressPKH, ok := theClient.ContractAddress.(*bitcoin.AddressPKH)
		if !ok {
			logger.Warn(ctx, "Contract address not PKH : %s", err)
			return nil
		}
		assetCode = protocol.AssetCodeFromContract(addressPKH.PKH(), 0)
		fundingAmount := uint64(2000)
		utxoAmount := uint64(fundingAmount + 500)
		requiredBalance := utxoAmount * 2             // Create contract and asset
		requiredBalance += utxoAmount * uint64(count) // Transfers

		// Start SpyNode ===========================================================================
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := theClient.RunSpyNode(ctx, false); err != nil {
				logger.Warn(ctx, "Spynode failed : %s", err)
			}
		}()

		// Wait for sync
		for i := 0; ; i++ {
			if theClient.IsInSync() && theClient.OutgoingCount() > 4 {
				break
			}
			if i > 60 {
				logger.Warn(ctx, "Timed out waiting for sync")
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
			time.Sleep(time.Second)
		}

		// Create UTXOs ============================================================================
		tx := txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit,
			theClient.Config.FeeRate)

		UTXOs := theClient.Wallet.UnspentOutputs()
		balance := uint64(0)
		for _, utxo := range UTXOs {
			if err := tx.AddInput(utxo.OutPoint, utxo.PkScript, utxo.Value); err != nil {
				logger.Warn(ctx, "Failed to add input to contract tx : %s", err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
			balance += utxo.Value
			if balance > requiredBalance {
				break
			}
		}

		if balance < requiredBalance {
			logger.Warn(ctx, "Not enough funds in wallet : %d < %d", balance, requiredBalance)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		utxoCount := count + 2
		for i := 0; i < utxoCount; i++ {
			if err := tx.AddPaymentOutput(theClient.Wallet.Address, utxoAmount, false); err != nil {
				logger.Warn(ctx, "Failed to add utxo output : %s", err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
		}

		// Sign tx
		if err := tx.Sign([]bitcoin.Key{theClient.Wallet.Key}); err != nil {
			logger.Warn(ctx, "Failed to sign utxo tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		utxoTx := tx
		utxoIndex := uint32(0)

		// Create contract =========================================================================
		tx = txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit,
			theClient.Config.FeeRate)

		if err := tx.AddInput(wire.OutPoint{Hash: utxoTx.MsgTx.TxHash(), Index: utxoIndex},
			utxoTx.MsgTx.TxOut[utxoIndex].PkScript,
			uint64(utxoTx.MsgTx.TxOut[utxoIndex].Value)); err != nil {
			logger.Warn(ctx, "Failed to add input to asset tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}
		utxoIndex++

		if err := tx.AddPaymentOutput(theClient.ContractAddress, fundingAmount, false); err != nil {
			logger.Warn(ctx, "Failed to add contract output : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		contract, err := contractOpReturn()
		if err != nil {
			logger.Warn(ctx, "Failed to create contract op return : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}
		if err := tx.AddOutput(contract, 0, false, false); err != nil {
			logger.Warn(ctx, "Failed to add op return output : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		// Sign tx
		if err := tx.Sign([]bitcoin.Key{theClient.Wallet.Key}); err != nil {
			logger.Warn(ctx, "Failed to sign contract offer tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		contractTx := tx

		// Create asset ============================================================================
		tx = txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit,
			theClient.Config.FeeRate)

		if err := tx.AddInput(wire.OutPoint{Hash: utxoTx.MsgTx.TxHash(), Index: utxoIndex},
			utxoTx.MsgTx.TxOut[utxoIndex].PkScript,
			uint64(utxoTx.MsgTx.TxOut[utxoIndex].Value)); err != nil {
			logger.Warn(ctx, "Failed to add input to asset tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}
		utxoIndex++

		if err := tx.AddPaymentOutput(theClient.ContractAddress, fundingAmount, false); err != nil {
			logger.Warn(ctx, "Failed to add contract output : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		asset, err := assetOpReturn()
		if err != nil {
			logger.Warn(ctx, "Failed to create asset op return : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}
		if err := tx.AddOutput(asset, 0, false, false); err != nil {
			logger.Warn(ctx, "Failed to add op return output : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		// Sign tx
		if err := tx.Sign([]bitcoin.Key{theClient.Wallet.Key}); err != nil {
			logger.Warn(ctx, "Failed to sign asset offer tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		assetTx := tx

		// Create transfer txs =====================================================================
		transferTxs := make([]*txbuilder.TxBuilder, 0, count)
		doubleTxs := make([]*txbuilder.TxBuilder, 0, count)

		for i := 0; i < count; i++ {
			tx = txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit,
				theClient.Config.FeeRate)

			if err := tx.AddInput(wire.OutPoint{Hash: utxoTx.MsgTx.TxHash(), Index: utxoIndex},
				utxoTx.MsgTx.TxOut[utxoIndex].PkScript,
				uint64(utxoTx.MsgTx.TxOut[utxoIndex].Value)); err != nil {
				logger.Warn(ctx, "Failed to add input to transfer %d tx : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
			utxoIndex++

			if err := tx.AddPaymentOutput(theClient.ContractAddress, fundingAmount, false); err != nil {
				logger.Warn(ctx, "Failed to add contract output to transfer %d tx : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}

			transfer, err := transferOpReturn()
			if err != nil {
				logger.Warn(ctx, "Failed to create transfer op return to transfer %d tx : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
			if err := tx.AddOutput(transfer, 0, false, false); err != nil {
				logger.Warn(ctx, "Failed to add op return output to transfer %d tx : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}

			// Sign tx
			if err := tx.Sign([]bitcoin.Key{theClient.Wallet.Key}); err != nil {
				logger.Warn(ctx, "Failed to sign transfer %d tx : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}

			transferTxs = append(transferTxs, tx)

			// TODO Build double spend tx
			tx = txbuilder.NewTxBuilder(theClient.Wallet.Address, theClient.Config.DustLimit,
				theClient.Config.FeeRate)

			doubleTxs = append(doubleTxs, tx)
		}

		var incomingTx *wire.MsgTx

		// Clear any previous incoming txs
		for len(theClient.IncomingTx.Channel) > 0 {
			_ = <-theClient.IncomingTx.Channel
		}

		// Send UTXO tx ============================================================================
		logger.Info(ctx, "Sending utxo tx")
		if err := theClient.BroadcastTxUntrustedOnly(ctx, utxoTx.MsgTx); err != nil {
			logger.Warn(ctx, "Failed to broadcast UTXO tx : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		// Send contract tx ========================================================================
		incomingTx = sendRequest(ctx, theClient, contractTx.MsgTx, "contract")
		if incomingTx == nil {
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		response, err := getResponse(incomingTx)
		if err != nil {
			logger.Warn(ctx, "Failed to parse contract response : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		if response.Type() == protocol.CodeContractFormation {
			logger.Info(ctx, "Contract formed")
		} else if response.Type() == protocol.CodeRejection {
			reject, _ := response.(*protocol.Rejection)
			logger.Warn(ctx, "Contract rejected : %s", reject.Message)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		} else {
			logger.Warn(ctx, "Unknown contract response type : %s", response.Type())
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		// Send asset tx ===========================================================================
		incomingTx = sendRequest(ctx, theClient, assetTx.MsgTx, "asset")
		if incomingTx == nil {
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		response, err = getResponse(incomingTx)
		if err != nil {
			logger.Warn(ctx, "Failed to parse asset response : %s", err)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		if response.Type() == protocol.CodeAssetCreation {
			assetCreation, _ := response.(*protocol.AssetCreation)
			logger.Info(ctx, "Asset created : %s", assetCreation.AssetCode.String())
		} else if response.Type() == protocol.CodeRejection {
			reject, _ := response.(*protocol.Rejection)
			logger.Warn(ctx, "Asset rejected : %s", reject.Message)
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		} else {
			logger.Warn(ctx, "Unknown asset response type : %s", response.Type())
			theClient.StopSpyNode(ctx)
			wg.Wait()
			return nil
		}

		// Send transfer Txs =======================================================================
		times := make([]uint64, 0, count)
		for i, transferTx := range transferTxs {
			start := time.Now()
			incomingTx = sendDoubleRequests(ctx, theClient, transferTx.MsgTx, doubleTxs[i].MsgTx, fmt.Sprintf("transfer %d", i))
			if incomingTx == nil {
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
			end := time.Now()

			// TODO Determine if the tx was accepted

			response, err := getResponse(incomingTx)
			if err != nil {
				logger.Warn(ctx, "Failed to parse transfer %d response : %s", i, err)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}

			times = append(times, uint64(end.UnixNano()-start.UnixNano()))

			if response.Type() == protocol.CodeSettlement {
				logger.Info(ctx, "Transfer %d accepted in %d ns", i, end.UnixNano()-start.UnixNano())
			} else if response.Type() == protocol.CodeRejection {
				reject, _ := response.(*protocol.Rejection)
				logger.Warn(ctx, "Transfer %d rejected : %s", i, reject.Message)
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			} else {
				logger.Warn(ctx, "Unknown transfer %d response type : %s", i, response.Type())
				theClient.StopSpyNode(ctx)
				wg.Wait()
				return nil
			}
		}

		// TODO Wait for confirms and determine which double spends were successful and if any were
		//   accepted by the smart-contract.

		total := uint64(0)
		for _, round := range times {
			total += round
		}
		logger.Info(ctx, "Average round trip (for %d) : %d ns", count, total/uint64(count))

		theClient.StopSpyNode(ctx)
		wg.Wait()
		return nil
	},
}

func sendDoubleRequests(ctx context.Context, client *client.Client, tx *wire.MsgTx, tx2 *wire.MsgTx, name string) *wire.MsgTx {
	logger.Info(ctx, "Sending %s tx", name)
	if err := client.BroadcastTxUntrustedOnly(ctx, tx); err != nil {
		logger.Warn(ctx, "Failed to broadcast %s tx : %s", name, err)
		return nil
	}

	// Wait for response on tx channel
	// TODO Add delay in case request was not accepted
	hash := tx.TxHash()
	for incomingTx := range client.IncomingTx.Channel {
		for _, input := range incomingTx.TxIn {
			if input.PreviousOutPoint.Hash == hash {
				return incomingTx
			}
		}
	}

	logger.Warn(ctx, "Channel closed")
	return nil
}

func init() {
}
