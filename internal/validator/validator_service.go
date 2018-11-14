package validator

/**
 * Validator Service
 *
 * What is my purpose?
 * - You take a Request action and tell me if its valid
 * - You return the Contract; or
 * - You prepare a rejection response
 */
import (
	"context"
	"errors"
	"fmt"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcutil"
)

var (
	ErrInsufficientPayment   = errors.New("Insufficient payment")
	ErrContractAlreadyExists = errors.New("Contract already exists at address")
)

const (
	MinimumForResponse = protocol.LimitDefault
)

func newRequestValidators(state state.StateInterface,
	config config.Config) map[string]validatorInterface {

	return map[string]validatorInterface{
		protocol.CodeContractOffer:     newContractOfferValidator(config.Fee),
		protocol.CodeContractAmendment: newContractAmendmentValidator(config.Fee),
		protocol.CodeAssetDefinition:   newAssetDefinitionValidator(config.Fee),
		protocol.CodeAssetModification: newAssetModificationValidator(config.Fee),
		protocol.CodeSend:              newSendValidator(config.Fee),
		protocol.CodeExchange:          newExchangeValidator(config.Fee),
		protocol.CodeOrder:             newOrderValidator(config.Fee),
		// protocol.CodeInitiative:        newInitiativeValidator(),
		// protocol.CodeReferendum:        newReferendumValidator(),
		// protocol.CodeBallotCast:        newBallotCastValidator(),
	}
}

type ValidatorService struct {
	Config     config.Config
	State      state.StateInterface
	Wallet     wallet.WalletInterface
	Fees       map[string]uint64
	validators map[string]validatorInterface
}

func NewValidatorService(config config.Config,
	wallet wallet.WalletInterface,
	state state.StateInterface) ValidatorService {
	return ValidatorService{
		Config:     config,
		State:      state,
		Wallet:     wallet,
		Fees:       protocol.Minimum,
		validators: newRequestValidators(state, config),
	}
}

// Validate Existing Contract
func (s ValidatorService) CheckContract(ctx context.Context,
	itx *inspector.Transaction,
	contract *contract.Contract) (*wire.MsgTx, *contract.Contract, error) {

	err := s.check(ctx, itx)
	if err != nil {
		return nil, nil, err
	}

	return s.checkContract(ctx, itx, contract)
}

// Validate and Find Contract
func (s ValidatorService) CheckAndFetch(ctx context.Context,
	itx *inspector.Transaction) (*wire.MsgTx, *contract.Contract, error) {

	m := itx.MsgProto
	log := logger.NewLoggerFromContext(ctx).Sugar()
	log.Infof("Received message : %s", m.Type())

	err := s.check(ctx, itx)
	if err != nil {
		return nil, nil, err
	}

	contractAddress := itx.Outputs[0].Address

	// This is the message we need to create a contract for. All other
	// messages will be able to get this contract
	contract, err := s.findContract(ctx, itx, contractAddress)
	if err != nil {
		if err == ErrContractAlreadyExists {
			code := protocol.RejectionCodeContractExists
			newTx, err := s.reject(ctx, itx, code)
			if err != nil {
				return nil, nil, err
			}
			log.Infof("Rejecting message : Contract already exists")
			return newTx, nil, nil
		}
		return nil, nil, err
	}

	return s.checkContract(ctx, itx, contract)
}

// Basic Check
func (s ValidatorService) check(ctx context.Context,
	itx *inspector.Transaction) error {

	m := itx.MsgProto

	if _, ok := protocol.TypeMapping[m.Type()]; !ok {
		return fmt.Errorf("Missing type mapping type : %v", m.Type())
	}

	_, ok := s.Fees[m.Type()]
	if !ok {
		return fmt.Errorf("No minimum for type : %v", m.Type())
	}

	return nil
}

// Contract-based Check
func (s ValidatorService) checkContract(ctx context.Context,
	itx *inspector.Transaction,
	contract *contract.Contract) (*wire.MsgTx, *contract.Contract, error) {

	m := itx.MsgProto
	log := logger.NewLoggerFromContext(ctx).Sugar()
	contractAddress := itx.Outputs[0].Address

	// Get spendable UTXO's received for the contract address
	utxos, err := itx.UTXOs.ForAddress(contractAddress)
	if err != nil {
		return nil, nil, err
	}

	// Check that we have received enough to perform the action.
	//
	// The txn fee (if any) will be paid by the responding transaction, so
	// the amount paid to the contract address needs to be the minimum, plus
	// the txn fee value.
	minimum := s.Fees[m.Type()]
	if uint64(utxos.Value()) < minimum {
		// There is insufficient value to fund this transaction.
		code := protocol.RejectionCodeInsufficientValue
		newTx, err := s.reject(ctx, itx, code)
		if err != nil {
			return nil, nil, err
		}
		log.Infof("Rejecting message : Insufficient value provided")
		return newTx, nil, nil
	}

	// State: I have seen this already
	if contract.KnownTX(ctx, itx.MsgTx) {
		return nil, nil, nil
	}

	// General permission check
	if !s.isPermitted(itx, contract) {
		code := protocol.RejectionCodeIssuerAddress
		newTx, err := s.reject(ctx, itx, code)
		if err != nil {
			return nil, nil, err
		}
		return newTx, nil, nil
	}

	// Action based validation
	msg := itx.MsgProto
	h, ok := s.validators[msg.Type()]
	if !ok {
		return nil, nil, fmt.Errorf("No validator found for type %v", msg.Type())
	}

	vdata := validatorData{
		contract: contract,
		m:        msg,
	}

	// Run the custom validator
	vcode := h.validate(ctx, itx, vdata)
	if vcode > 0 {
		newTx, err := s.reject(ctx, itx, vcode)
		if err != nil {
			return nil, nil, err
		}
		log.Infof("Rejecting message : Custom validator failed")
		return newTx, nil, nil
	}

	return nil, contract, nil
}

// findContract returns a new Contract for a ContractOffer, or finds the
// existing Contract for all other message types.
func (s ValidatorService) findContract(ctx context.Context,
	itx *inspector.Transaction,
	contractAddress btcutil.Address) (*contract.Contract, error) {

	addr := contractAddress.EncodeAddress()
	m := itx.MsgProto

	// Operator
	var operator btcutil.Address

	if len(itx.InputAddrs) > 1 {
		operator = itx.InputAddrs[1]
	}

	if m.Type() == protocol.CodeContractOffer {
		issuer := itx.InputAddrs[0]

		// this is a CO, so there must be not existing contract at this
		// address.
		c, err := s.State.Read(ctx, addr)
		if err != nil && err != state.ErrContractNotFound {
			return nil, err
		}

		// TODO guarantee that no contract exists at the address?

		if c != nil {
			return nil, ErrContractAlreadyExists
		}

		// Contract was not found, so create it.
		con := contract.NewContract(itx.MsgTx, contractAddress, issuer, operator)
		return con, nil
	}

	// TODO double check verification of issuer, contract operator and asset
	// holder addresses.
	//
	// Can the issuer or operator send this message, or is the sender an
	// asset holder? can they send this message?  This is at least partially
	// implemented in the handlers. Some of it may benefit from being back
	// at this layer.

	// all other message will have an existing contract. if they don't have
	// an existing contract a NotFound error will be returned.
	c, err := s.State.Read(ctx, addr)
	if err != nil {
		return nil, err
	}

	// ensure the assets map is initialized
	if c.Assets == nil {
		c.Assets = map[string]contract.Asset{}
	}

	// ensure the votes map is initialized
	if c.Votes == nil {
		c.Votes = map[string]contract.Vote{}
	}

	return c, nil
}

// Permission check
//
func (s ValidatorService) isPermitted(itx *inspector.Transaction,
	contract *contract.Contract) bool {

	msg := itx.MsgProto
	sender := itx.InputAddrs[0]

	if msg.Type() == protocol.CodeContractOffer {
		// anyone can send a CO
		return true
	}

	if contract.IsIssuer(sender.String()) ||
		contract.IsOperator(sender.String()) {

		return true
	}

	// TODO what about owners of assets? They can perform certain
	// operations.

	return true
}

// reject handles the situation where a message needs to be rejected.
//
// A Rejection message will be sent to the network, if there are enough
// funds issue the message.
//
func (s ValidatorService) reject(ctx context.Context,
	itx *inspector.Transaction,
	code uint8) (*wire.MsgTx, error) {

	// sender is the address that sent the message that we are rejecting.
	sender := itx.InputAddrs[0]

	// receiver (contract) is the address sending the message (UTXO)
	receiver := itx.Outputs[0]
	if receiver.Value < MinimumForResponse {
		// we did not receive enough to fund the response.
		return nil, ErrInsufficientPayment
	}

	// Find spendable UTXOs
	utxos, err := itx.UTXOs.ForAddress(receiver.Address)
	if err != nil {
		return nil, err
	}

	// Rejection protocol message
	rejection := buildRejectionFromCode(code)

	// sending the message to the sender of the message being rejected
	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: sender,
			Value:   546,
		},
	}

	// we spend the UTXO's to respond to the sender (+ others).
	//
	// The UTXOs to spend are in the TX we received.
	changeAddress := sender

	// Contract address
	key, err := s.Wallet.Get(receiver.Address.String())
	if err != nil {
		return nil, err
	}

	newTx, err := s.Wallet.BuildTX(key, utxos, outs, changeAddress, &rejection)
	if err != nil {
		return nil, err
	}

	return newTx, nil
}

// Build a rejection protocol message
func buildRejectionFromCode(code uint8) protocol.Rejection {
	r := protocol.NewRejection()
	r.RejectionType = code
	r.Message = protocol.RejectionCodes[code]
	return r
}
