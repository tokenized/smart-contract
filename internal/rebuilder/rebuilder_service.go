package rebuilder

import (
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/broadcaster"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
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

func (r RebuilderService) FindState(ctx context.Context,
	addr btcutil.Address) (*contract.Contract, *contract.Contract, error) {

	var (
		soft   *contract.Contract
		hard   *contract.Contract
		headTx int
		err    error
	)

	// Soft state
	//
	soft, err = r.State.Read(ctx, addr.String())
	if err != nil {
		soft, headTx, err = r.FindContract(ctx, addr)
		if err != nil {
			return nil, nil, err
		}

		soft.TxHeadCount = headTx
	}

	// Hard state
	//
	hard, err = r.State.ReadHard(ctx, addr.String())
	if err != nil {
		hard = soft
	}

	return &*hard, &*soft, nil
}

func (r RebuilderService) FindContract(ctx context.Context,
	addr btcutil.Address) (*contract.Contract, int, error) {

	log := logger.NewLoggerFromContext(ctx).Sugar()
	txHeadCount := 0

	// Find transactions
	transactions, err := r.ListTx(ctx, addr, 0)
	if err != nil {
		return nil, 0, err
	}

	// First pass, find responses
	for _, tx := range transactions {
		txHeadCount++

		// Inspector: Does this transaction concern the protocol?
		itx, err := r.Inspector.MakeTransaction(tx.msg)
		if err != nil || itx == nil {
			continue
		}

		// Introduce Inputs and UTXOs in the Transaction
		itx, err = r.Inspector.PromoteTransaction(itx)
		if err != nil {
			log.Error(err)
			continue
		}

		_, contract, err := r.Validator.CheckContract(ctx, itx)
		if contract != nil && contract.ID == addr.String() {
			return contract, txHeadCount, nil
		}
	}

	return nil, 0, errors.New("Could not find a contract offer for PKH anywhere")
}

func (r RebuilderService) ListTx(ctx context.Context,
	addr btcutil.Address, headCount int) ([]*RebuilderItem, error) {

	// Oldest -> Newest
	listResults, err := r.Network.ListTransactions(ctx, addr)
	if err != nil {
		return nil, err
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

	return transactions, nil
}

func (r RebuilderService) Sync(ctx context.Context,
	softContract *contract.Contract,
	hardContract *contract.Contract,
	contractAddr btcutil.Address) error {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	respondedHashes := []string{}
	txHeadCount := hardContract.TxHeadCount

	hourAgo := int64(time.Now().Add(-(time.Hour * 1)).Unix())

	// Find transactions
	transactions, err := r.ListTx(ctx, contractAddr, 0)
	if err != nil {
		return err
	}

	// First pass, find responses
	for _, tx := range transactions {
		txHeadCount++

		// Inspector: Does this transaction concern the protocol?
		itx, err := r.Inspector.MakeTransaction(tx.msg)
		if err != nil || itx == nil {
			continue
		}

		// Filter by Contract PKH and Response-type action
		itx, err = r.Response.PreFilter(ctx, itx)
		if err != nil || itx == nil {
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
		rejectTx, err := r.Validator.Check(ctx, itx, softContract)
		if err != nil {
			log.Error(err)
			continue
		}

		// Validator: Message is a reject
		if rejectTx != nil {
			_, _ = r.Broadcaster.Announce(ctx, rejectTx)
			continue
		}

		// Request: Grab me a response
		resItx, err := r.Request.Process(ctx, itx, softContract)
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

		// Response: Process response
		err = r.Response.Process(ctx, resItx, softContract)
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
