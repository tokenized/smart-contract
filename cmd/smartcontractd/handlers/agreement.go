package handlers

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/agreement"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Agreement struct {
	MasterDB *db.DB
	Config   *node.Config
}

// OfferRequest handles an incoming Body of Agreement Offer and prepares a Formation response
func (a *Agreement) OfferRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Agreement.Offer")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.BodyOfAgreementOffer)
	if !ok {
		return errors.New("Could not assert as *actions.BodyOfAgreementOffer")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Agreement offer invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Contract doesn't exist : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractDoesNotExist)
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen : %s", ct.FreezePeriod.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFrozen)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExpired)
	}

	if ct.BodyOfAgreementType != actions.ContractBodyOfAgreementTypeFull {
		node.LogWarn(ctx, "Contract body of agreement not allowed")
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsContractNotPermitted,
			"not body of agreement type full")
	}

	// Check for existing agreement
	if _, err := agreement.Retrieve(ctx, a.MasterDB, rk.Address); err == nil {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Agreement already exists : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAgreementExists)
	}

	// Create new agreement
	node.Log(ctx, "Accepting body of agreement offer")

	// BodyOfAgreementFormation <- BodyOfAgreementOffer
	formation := &actions.BodyOfAgreementFormation{}
	if err := node.Convert(ctx, &msg, formation); err != nil {
		return err
	}

	formation.Revision = 0
	formation.Timestamp = v.Now.Nano()

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx.
	if err := transactions.AddTx(ctx, a.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a formation
	if err := node.RespondSuccess(ctx, w, itx, rk, formation); err != nil {
		return err
	}

	return nil
}

// AmendmentRequest handles an incoming agreement Amendment and prepares a Formation response
func (a *Agreement) AmendmentRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Agreement.Amendment")
	defer span.End()

	node.Log(ctx, "Agreement amendment request")

	msg, ok := itx.MsgProto.(*actions.BodyOfAgreementAmendment)
	if !ok {
		return errors.New("Could not assert as *actions.BodyOfAgreementAmendment")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Agreement amendment invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		if errors.Cause(err) == contract.ErrNotFound {
			address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
			node.LogWarn(ctx, "Contract doesn't exist : %s", address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractDoesNotExist)
		}
		return errors.Wrap(err, "retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	agree, err := agreement.Retrieve(ctx, a.MasterDB, rk.Address)
	if err != nil {
		if errors.Cause(err) == agreement.ErrNotFound {
			address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
			node.LogWarn(ctx, "Agreement doesn't exist : %s", address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAgreementDoesNotExist)
		}
		return errors.Wrap(err, "retrieve agreement")
	}

	if agree.Revision != msg.Revision {
		node.LogWarn(ctx, "Incorrect agreement revision : specified %d != current %d",
			msg.Revision, agree.Revision)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAgreementRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalType := uint32(0)
	votingSystem := uint32(0)

	if len(msg.RefTxID) != 0 { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := bitcoin.NewHash32(msg.RefTxID)
		if err != nil {
			return errors.Wrap(err, "Failed to convert bitcoin.Hash32 to Hash32")
		}

		// Retrieve Vote Result
		voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Vote Result tx not found for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
		if !ok {
			node.LogWarn(ctx, "Vote Result invalid for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Retrieve the vote
		voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
		if err != nil {
			node.LogWarn(ctx, "Invalid vote txid : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		vt, err := vote.Retrieve(ctx, a.MasterDB, rk.Address, voteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %s", voteTxId)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteTxId, err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			node.LogWarn(ctx, "Vote not complete yet")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if vt.Result != "A" {
			node.LogWarn(ctx, "Vote result not A(Accept) : %s", vt.Result)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if len(vt.ProposedAmendments) == 0 {
			node.LogWarn(ctx, "Vote was not for specific amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if vt.AssetCode != nil && !vt.AssetCode.IsZero() {
			node.LogWarn(ctx, "Vote was not for contract amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "Proposal has different count of amendments : %d != %d",
				len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		proposalType = vt.Type
		votingSystem = vt.VoteSystem
	}

	contractFormation, err := contract.FetchContractFormation(ctx, a.MasterDB, rk.Address,
		a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "fetch contract formation")
	}

	// Build response
	formation := &actions.BodyOfAgreementFormation{}
	if err := node.Convert(ctx, agree, formation); err != nil {
		return err
	}

	formation.Revision = agree.Revision + 1
	formation.Timestamp = v.Now.Nano()

	if err := applyAgreementAmendments(formation, contractFormation.ContractPermissions,
		len(contractFormation.VotingSystems), msg.Amendments, proposed, proposalType,
		votingSystem); err != nil {
		node.LogWarn(ctx, "Failed to apply amendments : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx.
	if err := transactions.AddTx(ctx, a.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a formation
	if err := node.RespondSuccess(ctx, w, itx, rk, formation); err != nil {
		return errors.Wrap(err, "Failed to respond")
	}

	return nil
}

// FormationResponse handles an outgoing Agreement Formation and writes it to the state
func (a *Agreement) FormationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Agreement.Formation")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.BodyOfAgreementFormation)
	if !ok {
		return errors.New("Could not assert as *actions.BodyOfAgreementFormation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		return fmt.Errorf("Agreement formation not from contract : %s", address)
	}

	// Locate contract
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct != nil && !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address)
	}

	agree, err := agreement.Retrieve(ctx, a.MasterDB, rk.Address)
	if err != nil && errors.Cause(err) != agreement.ErrNotFound {
		return errors.Wrap(err, "retrieve agreement")
	}

	// Get request tx to find relevant vote if there was one.
	request, err := transactions.GetTx(ctx, a.MasterDB, &itx.Inputs[0].UTXO.Hash, a.Config.IsTest)
	var vt *state.Vote
	if err == nil && request != nil {
		amendment, ok := request.MsgProto.(*actions.BodyOfAgreementAmendment)

		if ok && len(amendment.RefTxID) != 0 {
			refTxId, err := bitcoin.NewHash32(amendment.RefTxID)
			if err != nil {
				return errors.Wrap(err, "Failed to convert bitcoin.Hash32 to bitcoin.Hash32")
			}

			// Retrieve Vote Result
			voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
			if err != nil {
				return errors.New("Vote Result tx not found for amendment")
			}

			voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
			if !ok {
				return errors.New("Vote Result invalid for amendment")
			}

			// Retrieve the vote
			voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
			if err != nil {
				return errors.Wrap(err, "invalid vote txid")
			}

			vt, err = vote.Retrieve(ctx, a.MasterDB, rk.Address, voteTxId)
			if err == vote.ErrNotFound {
				return errors.New("Vote not found for amendment")
			} else if err != nil {
				return errors.New("Failed to retrieve vote for amendment")
			}
		}
	}

	// Create or update agreement
	if agree == nil {
		// Prepare creation object
		newAgreement := &agreement.NewAgreement{}
		if err := node.Convert(ctx, &msg, newAgreement); err != nil {
			node.LogWarn(ctx, "Failed to convert formation to new agreement : %s", err)
			return err
		}

		if err := agreement.Create(ctx, a.MasterDB, rk.Address, newAgreement, v.Now); err != nil {
			node.LogWarn(ctx, "Failed to create agreement : %s", err)
			return err
		}
		node.Log(ctx, "Created agreement")

	} else {
		// Prepare update object
		ts := protocol.NewTimestamp(msg.Timestamp)
		uc := agreement.UpdateAgreement{
			Revision:  &msg.Revision,
			Timestamp: &ts,
		}

		// Check if chapters are different
		different := len(agree.Chapters) != len(msg.Chapters)
		if !different {
			for i, chapter := range agree.Chapters {
				if !chapter.Equal(msg.Chapters[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating chapters")
			uc.Chapters = &msg.Chapters
		}

		// Check if definitions are different
		different = len(agree.Definitions) != len(msg.Definitions)
		if !different {
			for i, definition := range agree.Definitions {
				if !definition.Equal(msg.Definitions[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating definitions")
			uc.Definitions = &msg.Definitions
		}

		if err := agreement.Update(ctx, a.MasterDB, rk.Address, &uc, v.Now); err != nil {
			return errors.Wrap(err, "Failed to update agreement")
		}
		node.Log(ctx, "Updated agreement")

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId)
			if err := vote.MarkApplied(ctx, a.MasterDB, rk.Address, vt.VoteTxId, request.Hash,
				v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}
	}

	return nil
}

// applyAgreementAmendments applies the amendments to the agreement formation.
func applyAgreementAmendments(formation *actions.BodyOfAgreementFormation, permissionBytes []byte,
	votingSystemsCount int, amendments []*actions.AmendmentField, proposed bool, proposalType,
	votingSystem uint32) error {

	perms, err := permissions.PermissionsFromBytes(permissionBytes, votingSystemsCount)
	if err != nil {
		return errors.Wrap(err, "parse permissions")
	}

	perms, err = perms.SubPermissions(
		permissions.FieldIndexPath{actions.ContractFieldBodyOfAgreement}, 0, false)
	if err != nil {
		return errors.Wrap(err, "sub permissions")
	}

	for i, amendment := range amendments {
		fip, err := permissions.FieldIndexPathFromBytes(amendment.FieldIndexPath)
		if err != nil {
			return fmt.Errorf("Failed to read amendment %d field index path : %s", i, err)
		}
		if len(fip) == 0 {
			return fmt.Errorf("Amendment %d has no field specified", i)
		}

		fieldPermissions, err := formation.ApplyAmendment(fip, amendment.Operation, amendment.Data,
			perms)
		if err != nil {
			return errors.Wrapf(err, "apply amendment %d", i)
		}
		if len(fieldPermissions) == 0 {
			return errors.New("Invalid field permissions")
		}

		// fieldPermissions are the permissions that apply to the field that was changed in the
		// amendment.
		permission := fieldPermissions[0]
		if proposed {
			switch proposalType {
			case 0: // Administration
				if !permission.AdministrationProposal {
					return node.NewError(actions.RejectionsContractPermissions,
						fmt.Sprintf("Field %s amendment not permitted by administration proposal",
							fip))
				}
			case 1: // Holder
				if !permission.HolderProposal {
					return node.NewError(actions.RejectionsContractPermissions,
						fmt.Sprintf("Field %s amendment not permitted by holder proposal", fip))
				}
			case 2: // Administrative Matter
				if !permission.AdministrativeMatter {
					return node.NewError(actions.RejectionsContractPermissions,
						fmt.Sprintf("Field %s amendment not permitted by administrative vote",
							fip))
				}
			default:
				return fmt.Errorf("Invalid proposal initiator type : %d", proposalType)
			}

			if int(votingSystem) >= len(permission.VotingSystemsAllowed) {
				return fmt.Errorf("Field %s amendment voting system out of range : %d", fip,
					votingSystem)
			}
			if !permission.VotingSystemsAllowed[votingSystem] {
				return node.NewError(actions.RejectionsContractPermissions,
					fmt.Sprintf("Field %s amendment not allowed using voting system %d",
						fip, votingSystem))
			}
		} else if !permission.Permitted {
			return node.NewError(actions.RejectionsContractPermissions,
				fmt.Sprintf("Field %s amendment not permitted without proposal", fip))
		}
	}

	// Check validity of updated contract data
	if err := formation.Validate(); err != nil {
		return fmt.Errorf("Agreement data invalid after amendments : %s", err)
	}

	return nil
}
