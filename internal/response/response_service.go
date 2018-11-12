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
	"github.com/tokenized/smart-contract/pkg/protocol"
)

var (
	incomingMessageTypes = map[string]bool{
		protocol.CodeAssetCreation:     true,
		protocol.CodeContractFormation: true,
		protocol.CodeSettlement:        true,
		protocol.CodeVote:              true,
		protocol.CodeBallotCounted:     true,
		protocol.CodeResult:            true,
		protocol.CodeFreeze:            true,
		protocol.CodeThaw:              true,
		protocol.CodeConfiscation:      true,
		protocol.CodeReconciliation:    true,
		protocol.CodeRejection:         true,
	}
)

func newContractResponders(state state.StateInterface,
	config config.Config) map[string]responseInterface {

	return map[string]responseInterface{
		protocol.CodeAssetCreation:     newAssetCreationResponse(),
		protocol.CodeContractFormation: newContractFormationResponse(),
		protocol.CodeSettlement:        newSettlementResponse(),
		protocol.CodeVote:              newVoteResponse(),
		protocol.CodeBallotCounted:     newBallotCountedResponse(),
		protocol.CodeResult:            newResultResponse(),
		protocol.CodeFreeze:            newFreezeResponse(),
		protocol.CodeThaw:              newThawResponse(),
		protocol.CodeConfiscation:      newConfiscationResponse(),
		protocol.CodeReconciliation:    newReconciliationResponse(),
		protocol.CodeRejection:         newRejectionResponse(),
	}
}

type ResponseService struct {
	Config     config.Config
	State      state.StateInterface
	responders map[string]responseInterface
}

func NewResponseService(config config.Config,
	state state.StateInterface) ResponseService {
	return ResponseService{
		State:      state,
		Config:     config,
		responders: newContractResponders(state, config),
	}
}

func (s ResponseService) Process(ctx context.Context,
	itx *inspector.Transaction, contract *contract.Contract) error {

	msg := itx.MsgProto

	// select the handler for this message type
	h, ok := s.responders[msg.Type()]
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
