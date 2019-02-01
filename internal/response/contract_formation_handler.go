package response

import (
	"context"

	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type contractFormationHandler struct{}

func newContractFormationHandler() contractFormationHandler {
	return contractFormationHandler{}
}

func (h contractFormationHandler) process(ctx context.Context,
	itx *inspector.Transaction, c *contract.Contract) error {

	msg := itx.MsgProto.(*protocol.ContractFormation)

	contract.EditContract(c, msg)

	return nil
}
