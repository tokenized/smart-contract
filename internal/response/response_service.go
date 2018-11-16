package response

/**
 * Response Service
 *
 * What is my purpose?
 * - You accept a Response action
 * - You save it to the Contract state
 */

import (
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/app/config"
	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/internal/app/wallet"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

var (
	outgoingMessageOutputs = map[string]int{
		protocol.CodeAssetCreation:     0,
		protocol.CodeContractFormation: 0,
		protocol.CodeSettlement:        2,
		protocol.CodeVote:              0,
		protocol.CodeBallotCounted:     1,
		protocol.CodeResult:            0,
		protocol.CodeFreeze:            1,
		protocol.CodeThaw:              1,
		protocol.CodeConfiscation:      2,
		protocol.CodeReconciliation:    1,
		protocol.CodeRejection:         1,
	}
)

func newResponseHandlers(state state.StateInterface,
	config config.Config) map[string]responseHandlerInterface {

	return map[string]responseHandlerInterface{
		protocol.CodeAssetCreation:     newAssetCreationHandler(),
		protocol.CodeContractFormation: newContractFormationHandler(),
		protocol.CodeSettlement:        newSettlementHandler(),
		protocol.CodeFreeze:            newFreezeHandler(),
		protocol.CodeThaw:              newThawHandler(),
		protocol.CodeConfiscation:      newConfiscationHandler(),
		protocol.CodeReconciliation:    newReconciliationHandler(),
		protocol.CodeRejection:         newRejectionHandler(),
		// protocol.CodeVote:              newVoteHandler(),
		// protocol.CodeBallotCounted:     newBallotCountedHandler(),
		// protocol.CodeResult:            newResultHandler(),
	}
}

type ResponseService struct {
	Config    config.Config
	State     state.StateInterface
	Wallet    wallet.WalletInterface
	Inspector inspector.InspectorService
	handlers  map[string]responseHandlerInterface
}

func NewResponseService(config config.Config,
	wallet wallet.WalletInterface,
	state state.StateInterface,
	inspector inspector.InspectorService) ResponseService {
	return ResponseService{
		State:     state,
		Wallet:    wallet,
		Config:    config,
		Inspector: inspector,
		handlers:  newResponseHandlers(state, config),
	}
}

// Performant filter to run before validation checks
//
func (s ResponseService) PreFilter(ctx context.Context,
	itx *inspector.Transaction) (*inspector.Transaction, error) {

	// Filter by: Response-type action
	//
	if !s.Inspector.IsOutgoingMessageType(itx.MsgProto) {
		return nil, nil
	}

	// Select the output index for this message type
	//
	msg := itx.MsgProto
	outputIndex, ok := outgoingMessageOutputs[msg.Type()]
	if !ok {
		return nil, fmt.Errorf("No output index found for type %v", msg.Type())
	}

	// Filter by: Contract PKH
	//
	if len(itx.Outputs) < (outputIndex + 1) {
		return nil, fmt.Errorf("Not enough outputs in TX %s", itx.MsgTx.TxHash())
	}

	contractAddress := itx.Outputs[outputIndex].Address.String()
	_, err := s.Wallet.Get(contractAddress)
	if err != nil {
		return nil, err
	}

	return itx, nil
}

func (s ResponseService) Process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	msg := itx.MsgProto

	// select the handler for this message type
	h, ok := s.handlers[msg.Type()]
	if !ok {
		return fmt.Errorf("No response handler found for type %v", msg.Type())
	}

	// Run the handler, return the response
	err := h.process(ctx, itx, contract)
	if err != nil {
		return err
	}

	if err := s.State.Write(ctx, *contract); err != nil {
		return err
	}

	return nil
}
