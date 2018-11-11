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

	"github.com/tokenized/smart-contract/internal/app/inspector"
	"github.com/tokenized/smart-contract/internal/app/state"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type ResponseService struct {
	State state.StateInterface
}

func NewResponseService(contractState state.StateInterface) ResponseService {
	return ResponseService{
		State: contractState,
	}
}

func (s ResponseService) Process(ctx context.Context,
	tx *inspector.Transaction, contract *contract.Contract) error {

	// TODO: Move the logic request handlers are using to write to the contract state

	if err := s.State.Write(ctx, *contract); err != nil {
		return err
	}

	return nil
}
