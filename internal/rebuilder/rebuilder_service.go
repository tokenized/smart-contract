package rebuilder

import (
	"context"
	"time"

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/internal/broadcaster"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

/**
 * Rebuilder Service
 *
 * What is my purpose?
 * - You process missed messages whilst contract was offline
 * - You rebuild the contract state from nothing
 * - You harden state by some set timeframe
 */

type RebuilderItem struct {
	result *btcjson.ListTransactionsResult
	msg    *wire.MsgTx
}

type RebuilderService struct {
	Network     network.NetworkInterface
	Inspector   inspector.InspectorService
	Broadcaster broadcaster.BroadcastService
	Request     request.RequestService
	Response    response.ResponseService
	Validator   validator.ValidatorService
	State       state.StateInterface
}

func NewRebuilderService(network network.NetworkInterface,
	inspector inspector.InspectorService,
	broadcaster broadcaster.BroadcastService,
	request request.RequestService,
	response response.ResponseService,
	validator validator.ValidatorService,
	state state.StateInterface) RebuilderService {
	return RebuilderService{
		Network:     network,
		Inspector:   inspector,
		Broadcaster: broadcaster,
		Request:     request,
		Response:    response,
		Validator:   validator,
		State:       state,
	}
}

func (r RebuilderService) Sync(ctx context.Context,
	softContract *contract.Contract,
	hardContract *contract.Contract,
	addr string) error {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	contractAddr, err := btcutil.DecodeAddress(string(addr), &chaincfg.MainNetParams)
	if err != nil {
		return err
	}

	respondedHashes := []string{}
	txHeadCount := hardContract.TxHeadCount
	hourAgo := int64(time.Now().Add(-(time.Hour * 1)).Unix())

	// Oldest -> Newest
	listResults, err := r.Network.ListTransactions(ctx, contractAddr)
	if err != nil {
		return err
	}

	// Make usable TX
	transactions := []*RebuilderItem{}
	for _, rtx := range listResults {
		hash, err := chainhash.NewHashFromStr(rtx.TxID)
		if err != nil {
			continue
		}
		ntx, err := r.Network.GetTX(ctx, hash)
		if err != nil {
			continue
		}
		item := &RebuilderItem{
			result: &rtx,
			msg:    ntx,
		}
		transactions = append(transactions, item)
	}

	// First pass, find responses
	for _, tx := range transactions {
		txHeadCount++

		// Inspector: Does this transaction concern the protocol?
		itx, err := r.Inspector.MakeTransaction(tx.msg)
		if err != nil || itx == nil {
			continue
		}

		// Inspector: Is transaction a response?
		if !r.Inspector.IsOutgoingMessageType(itx.MsgProto) {
			continue
		}

		// Store request related to this response
		respondedHashes = append(respondedHashes, tx.msg.TxIn[0].PreviousOutPoint.Hash.String())

		// Apply to Soft State
		err = r.Response.Process(ctx, itx, softContract)
		if err != nil {
			log.Error(err)
			continue
		}
		softContract.UpdatedAt = tx.result.Time
		softContract.TxHeadCount = txHeadCount

		// Apply to Hard State
		if tx.result.Time > hourAgo {
			err = r.Response.Process(ctx, itx, hardContract)
			if err != nil {
				log.Error(err)
				continue
			}
			hardContract.UpdatedAt = tx.result.Time
			hardContract.TxHeadCount = txHeadCount
		}
	}

	// Second pass, find unresponded requests
	for _, tx := range transactions {

		// Inspector: Does this transaction concern the protocol?
		itx, err := r.Inspector.MakeTransaction(tx.msg)
		if err != nil || itx == nil {
			continue
		}

		// Filter by Contract PKH and Request-type action
		itx, err = r.Request.PreFilter(ctx, itx)
		if err != nil || itx == nil {
			return nil
		}

		// Has request been responded to?
		if stringInSlice(tx.result.TxID, respondedHashes) {
			continue
		}

		// Introduce Inputs and UTXOs in the Transaction
		itx, err = r.Inspector.PromoteTransaction(itx)
		if err != nil {
			log.Error(err)
			continue
		}

		// Validator: Check this request, return the related Contract
		rejectTx, contract, err := r.Validator.CheckContract(ctx, itx, softContract)
		if err != nil {
			log.Error(err)
			continue
		}

		// Validator: Message is a reject
		if rejectTx != nil {
			_, _ = r.Broadcaster.Announce(ctx, rejectTx)
			continue
		}

		// Missing a contract (do nothing)
		if contract == nil {
			continue
		}

		// Request: Grab me a response
		resItx, err := r.Request.Process(ctx, itx, softContract)
		if err != nil {
			log.Error(err)
			continue
		}

		// Response: Process response
		err = r.Response.Process(ctx, resItx, softContract)
		if err != nil {
			log.Error(err)
			continue
		}

		// Broadcaster: Broadcast response
		_, err = r.Broadcaster.Announce(ctx, resItx.MsgTx)
		if err != nil {
			log.Error(err)
			continue
		}
	}

	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
