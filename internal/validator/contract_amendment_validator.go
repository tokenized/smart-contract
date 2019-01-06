package validator

import (
	"context"
	"reflect"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type contractAmendmentValidator struct {
	Fee config.Fee
}

// newContractAmendmentValidator returns a new contractAmendmentValidator.
func newContractAmendmentValidator(fee config.Fee) contractAmendmentValidator {
	return contractAmendmentValidator{
		Fee: fee,
	}
}

// can returns a code indicating if the message can be applied to the
// contract.
//
// A return value of 0 (protocol.RejectionCodeOK) indicates that the message
// can be applied to the Contract. Any non-zero value should be interpreted
// as the rejection code.
func (h contractAmendmentValidator) validate(ctx context.Context,
	itx *inspector.Transaction, vd validatorData) uint8 {

	log := logger.NewLoggerFromContext(ctx).Sugar()

	// Contract and Message
	c := vd.contract
	m := vd.m.(*protocol.ContractAmendment)

	// if c.Revision != m.ContractRevision {
	// 	return protocol.RejectionCodeContractRevision
	// }

	// TODO enable this again after authflags have been discussed further.

	// if h.hasGeneralChanges(c, m) && !h.canUpdate(c) {
	// 	// general update denied
	// 	return protocol.RejectionCodeContractUpdate
	// }

	// if h.authFlagsChanged(c, m) && !h.canChangeAuthFlags(c) {
	// 	// auth flag change denied
	// 	return protocol.RejectionCodeContractAuthFlags
	// }

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if c.Qty > 0 && int(m.RestrictedQty) < len(c.Assets) {
		log.Errorf("contract amendment : Cannot reduce allowable assets below existing number")
		return protocol.RejectionCodeContractQtyReduction
	}

	return protocol.RejectionCodeOK
}

// canChangeAuthFlags returns true if the auth flags allow the issuer to
// change auh flags, false otherwise.
func (h contractAmendmentValidator) canChangeAuthFlags(c contract.Contract) bool {
	return protocol.IsAuthorized(c.Flags(), protocol.ContractAuthFlagsIssuer)
}

// canUpdate returns true if the contract auth flags permit the issuer to
// change the contract, false otherwise.
func (h contractAmendmentValidator) canUpdate(c contract.Contract) bool {
	return protocol.IsAuthorized(c.Flags(), protocol.ContractIssuerUpdate)
}

// authFlagsChanged returns true if the message is changing auth flags,
// false otherwise.
func (h contractAmendmentValidator) authFlagsChanged(c contract.Contract,
	m *protocol.ContractAmendment) bool {

	return !reflect.DeepEqual(c.AuthorizationFlags, m.AuthorizationFlags)
}

// hasGeneralChanges returns true if any field, (excluding auth flags,
// expiration and quantity), false otherwise.
func (h contractAmendmentValidator) hasGeneralChanges(c contract.Contract,
	m *protocol.ContractAmendment) bool {

	return c.ContractName != string(m.ContractName) ||
		c.ContractFileHash != string(m.ContractFileHash) ||
		c.GoverningLaw != string(m.GoverningLaw) ||
		c.Jurisdiction != string(m.Jurisdiction) ||
		c.ContractExpiration != c.ContractExpiration ||
		c.URI != string(m.URI) ||
		c.IssuerID != string(m.IssuerID) ||
		c.ContractOperatorID != string(m.ContractOperatorID) ||
		c.VotingSystem != string(m.VotingSystem) ||
		c.InitiativeThreshold != m.InitiativeThreshold ||
		c.Qty != m.RestrictedQty ||
		c.InitiativeThresholdCurrency != string(m.InitiativeThresholdCurrency)
}
