package request

/**
 * Request Service
 *
 * What is my purpose?
 * - You accept Request actions
 * - You are given a validator to validate the request
 * - You prepare a Response action
 */

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

const (
	dustLimit = 546
)

func newRequestHandlers(state state.StateInterface,
	config config.Config) map[string]requestHandlerInterface {

	return map[string]requestHandlerInterface{
		protocol.CodeContractOffer:     newContractOfferHandler(config.Fee),
		protocol.CodeContractAmendment: newContractAmendmentHandler(config.Fee),
		protocol.CodeAssetDefinition:   newAssetDefinitionHandler(config.Fee),
		protocol.CodeAssetModification: newAssetModificationHandler(config.Fee),
		protocol.CodeSend:              newSendHandler(config.Fee),
		protocol.CodeExchange:          newExchangeHandler(config.Fee),
		protocol.CodeOrder:             newOrderHandler(config.Fee),
		// protocol.CodeInitiative:        newInitiativeHandler(),
		// protocol.CodeReferendum:        newReferendumHandler(),
		// protocol.CodeBallotCast:        newBallotCastHandler(),
	}
}

type RequestService struct {
	Config    config.Config
	State     state.StateInterface
	Wallet    wallet.WalletInterface
	Inspector inspector.InspectorService
	handlers  map[string]requestHandlerInterface
}

func NewRequestService(config config.Config,
	wallet wallet.WalletInterface,
	state state.StateInterface,
	inspector inspector.InspectorService) RequestService {

	return RequestService{
		Config:    config,
		State:     state,
		Wallet:    wallet,
		Inspector: inspector,
		handlers:  newRequestHandlers(state, config),
	}
}

// Performant filter to run before validation checks
//
func (s RequestService) PreFilter(ctx context.Context,
	itx *inspector.Transaction) (*inspector.Transaction, error) {

	// Filter by: Request-type action
	if !s.Inspector.IsIncomingMessageType(itx.MsgProto) {
		// This isn't an error, it just isn't an incoming message we don't
		// want to process, such as a "response" message, such as a
		// ContractFormation. We send those to the network, but we don't
		// process them as incoming messages.
		return nil, nil
	}

	// Filter by: Contract PKH
	if len(itx.Outputs) == 0 {
		return nil, fmt.Errorf("No outputs in TX %s", itx.MsgTx.TxHash())
	}

	contractAddress := itx.Outputs[0].Address.String()
	_, err := s.Wallet.Get(contractAddress)
	if err != nil {
		return nil, err
	}

	return itx, nil
}

// Process the request through a handler
//
func (s RequestService) Process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) (*inspector.Transaction, error) {

	tx := itx.MsgTx
	msg := itx.MsgProto

	// select the handler for this message type
	h, ok := s.handlers[msg.Type()]
	if !ok {
		return nil, fmt.Errorf("No request handler found for type %v", msg.Type())
	}

	hash := tx.TxHash()

	req := contractRequest{
		tx:        tx,
		hash:      hash,
		senders:   itx.InputAddrs,
		receivers: itx.Outputs,
		contract:  *contract,
		m:         msg,
	}

	// Run the handler, return the response
	res, err := h.handle(ctx, req)
	if err != nil {
		return nil, err
	}

	res.Contract.Hashes = append(res.Contract.Hashes, hash.String())

	// Get spendable UTXO's received for the contract address
	contractAddress := itx.Outputs[0].Address
	utxos, err := itx.UTXOs.ForAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	// Send change back to the sender by default,
	// unless there is a specific address.
	changeAddress := itx.InputAddrs[0]
	if res.changeAddress != nil {
		changeAddress = res.changeAddress
	}

	// Contract private key
	key, err := s.Wallet.Get(contractAddress.String())
	if err != nil {
		return nil, err
	}

	// Create usable transaction to pass back
	newTx, err := s.Wallet.BuildTX(key, utxos, res.outs, changeAddress, res.Message)
	if err != nil {
		return nil, err
	}

	newItx := s.Inspector.CreateTransaction(utxos, res.outs, res.Message)
	newItx.MsgTx = newTx

	return newItx, nil
}

// hashToBytes returns a Hash in little endian format as Hash.CloneByte()
// returns bytes in big endian format.
func hashToBytes(hash chainhash.Hash) []byte {
	b, _ := hex.DecodeString(hash.String())

	return b
}
