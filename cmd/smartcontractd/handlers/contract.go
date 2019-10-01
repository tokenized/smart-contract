package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
	Headers  node.BitcoinHeaders
}

// OfferRequest handles an incoming Contract Offer and prepares a Formation response
func (c *Contract) OfferRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Offer")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *actions.ContractOffer")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract offer request invalid : %d", itx.RejectCode)
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	_, err := contract.Retrieve(ctx, c.MasterDB, rk.Address)
	if err != contract.ErrNotFound {
		if err == nil {
			address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
			node.LogWarn(ctx, "Contract already exists : %s", address.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve contract")
		}
	}

	if msg.BodyOfAgreementType == 1 && len(msg.BodyOfAgreement) != 32 {
		node.LogWarn(ctx, "Contract body of agreement hash is incorrect length : %d", len(msg.BodyOfAgreement))
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if msg.ContractExpiration != 0 && msg.ContractExpiration < v.Now.Nano() {
		node.LogWarn(ctx, "Expiration already passed : %d", msg.ContractExpiration)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if _, err = protocol.PermissionsFromBytes(msg.ContractPermissions, len(msg.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid contract permissions : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Validate voting systems are all valid.
	for _, votingSystem := range msg.VotingSystems {
		if err = vote.ValidateVotingSystem(votingSystem); err != nil {
			node.LogWarn(ctx, "Invalid voting system : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
	}

	node.Log(ctx, "Accepting contract offer : %s", msg.ContractName)

	// Contract Formation <- Contract Offer
	cf := actions.ContractFormation{}

	err = node.Convert(ctx, &msg, &cf)
	if err != nil {
		return err
	}

	cf.ContractRevision = 0
	cf.Timestamp = v.Now.Nano()

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, msg.ContractFee)

	// Save Tx for when formation is processed.
	if err := transactions.AddTx(ctx, c.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &cf)
}

// AmendmentRequest handles an incoming Contract Amendment and prepares a Formation response
func (c *Contract) AmendmentRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Amendment")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractAmendment)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractAmendment")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract amendment request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	if ct.Revision != msg.ContractRevision {
		node.LogWarn(ctx, "Incorrect contract revision (%s) : specified %d != current %d",
			ct.ContractName, msg.ContractRevision, ct.Revision)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalInitiator := uint32(0)
	votingSystem := uint32(0)

	if len(msg.RefTxID) != 0 { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := bitcoin.NewHash32(msg.RefTxID)
		if err != nil {
			return errors.Wrap(err, "Failed to convert protocol.TxId to Hash32")
		}

		// Retrieve Vote Result
		voteResultTx, err := transactions.GetTx(ctx, c.MasterDB, refTxId, c.Config.IsTest)
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
		voteTxId := protocol.TxIdFromBytes(voteResult.VoteTxId)
		vt, err := vote.Retrieve(ctx, c.MasterDB, rk.Address, voteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %s", voteTxId.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteTxId.String(), err)
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

		if !vt.AssetCode.IsZero() {
			node.LogWarn(ctx, "Vote was not for contract amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if ct.RestrictedQtyAssets > 0 && ct.RestrictedQtyAssets < uint64(len(ct.AssetCodes)) {
		node.LogWarn(ctx, "Cannot reduce allowable assets below existing number")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractAssetQtyReduction)
	}

	if msg.ChangeAdministrationAddress || msg.ChangeOperatorAddress {
		if !ct.OperatorAddress.IsEmpty() {
			if len(itx.Inputs) < 2 {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractBothOperatorsRequired)
			}

			if itx.Inputs[0].Address.Equal(itx.Inputs[1].Address) ||
				!contract.IsOperator(ctx, ct, itx.Inputs[0].Address) ||
				!contract.IsOperator(ctx, ct, itx.Inputs[1].Address) {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractBothOperatorsRequired)
			}
		} else {
			if len(itx.Inputs) < 1 {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractBothOperatorsRequired)
			}

			if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractBothOperatorsRequired)
			}
		}
	}

	// Check oracle signature
	if ct.AdminOracle != nil {
		// Check that new signature is provided
		adminOracleSigIncluded := false
		for i, amendment := range msg.Amendments {
			fip, err := protocol.FieldIndexPathFromBytes(amendment.FieldIndexPath)
			if err != nil {
				return fmt.Errorf("Failed to read amendment %d field index path : %s", i, err)
			}
			if len(fip) > 0 && fip[0] == actions.ContractFieldAdminOracleSignature {
				adminOracleSigIncluded = true
				break
			}
		}

		if !adminOracleSigIncluded {
			node.Log(ctx, "New oracle signature required to change administration or operator")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInvalidSignature)
		}

		// Build updated contract to check against signature
		cf := actions.ContractFormation{}

		// Get current state
		err = node.Convert(ctx, ct, &cf)
		if err != nil {
			return errors.Wrap(err, "Failed to convert state contract to contract formation")
		}

		if err := applyContractAmendments(&cf, msg.Amendments, proposed, proposalInitiator,
			votingSystem); err != nil {
			node.LogWarn(ctx, "Failed to apply amendments to check admin oracle sig : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Apply updates
		updatedContract := *ct
		err = node.Convert(ctx, &cf, &updatedContract)
		if err != nil {
			return errors.Wrap(err, "Failed to convert amended contract formation to contract state")
		}

		// Pull from amendment tx.
		// Administration change. New administration in second input
		inputIndex := 1
		if !ct.OperatorAddress.IsEmpty() {
			inputIndex++
		}

		if msg.ChangeAdministrationAddress {
			if len(itx.Inputs) <= inputIndex {
				return errors.New("New administration specified but not included in inputs")
			}

			updatedContract.AdministrationAddress = itx.Inputs[inputIndex].Address
			inputIndex++
		}

		// Operator changes. New operator in second input unless there is also a new administration,
		//   then it is in the third input
		if msg.ChangeOperatorAddress {
			if len(itx.Inputs) <= inputIndex {
				return errors.New("New operator specified but not included in inputs")
			}

			updatedContract.OperatorAddress = itx.Inputs[inputIndex].Address
		}

		if err := validateContractAmendOracleSig(ctx, &updatedContract, c.Headers); err != nil {
			node.LogVerbose(ctx, "New oracle signature invalid : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInvalidSignature)
		}
	}

	// Contract Formation <- Contract Amendment
	cf := actions.ContractFormation{}

	// Get current state
	err = node.Convert(ctx, ct, &cf)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state contract to contract formation")
	}

	// Apply modifications
	cf.ContractRevision = ct.Revision + 1 // Bump the revision
	cf.Timestamp = v.Now.Nano()

	if err := applyContractAmendments(&cf, msg.Amendments, proposed, proposalInitiator,
		votingSystem); err != nil {
		node.LogWarn(ctx, "Contract amendments failed : %s", err)
		code, ok := node.ErrorCode(err)
		if ok {
			return node.RespondReject(ctx, w, itx, rk, code)
		}
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Administration change. New administration in next input
	inputIndex := 1
	if !ct.OperatorAddress.IsEmpty() {
		inputIndex++
	}
	if msg.ChangeAdministrationAddress {
		if len(itx.Inputs) <= inputIndex {
			node.LogWarn(ctx, "New administration specified but not included in inputs (%s)", ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsTxMalformed)
		}
		inputIndex++
	}

	// Operator changes. New operator in second input unless there is also a new administration, then it is in the third input
	if msg.ChangeOperatorAddress {
		if len(itx.Inputs) <= inputIndex {
			node.LogWarn(ctx, "New operator specified but not included in inputs (%s)", ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsTxMalformed)
		}
	}

	// Save Tx.
	if err := transactions.AddTx(ctx, c.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	node.Log(ctx, "Accepting contract amendment (%s)", ct.ContractName)

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &cf)
}

// FormationResponse handles an outgoing Contract Formation and writes it to the state
func (c *Contract) FormationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Formation")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractFormation)
	if !ok {
		return errors.New("Could not assert as *actions.ContractFormation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	if itx.RejectCode != 0 {
		return errors.New("Contract formation response invalid")
	}

	// Locate Contract. Sender is verified to be contract before this response function is called.
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		return fmt.Errorf("Contract formation not from contract : %x",
			itx.Inputs[0].Address.Bytes())
	}

	contractName := msg.ContractName
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct != nil && !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, c.MasterDB, &itx.Inputs[0].UTXO.Hash, c.Config.IsTest)
	var vt *state.Vote
	var amendment *actions.ContractAmendment
	if err == nil && request != nil {
		var ok bool
		amendment, ok = request.MsgProto.(*actions.ContractAmendment)

		if ok && len(amendment.RefTxID) != 0 {
			refTxId, err := bitcoin.NewHash32(amendment.RefTxID)
			if err != nil {
				return errors.Wrap(err, "Failed to convert protocol.TxId to bitcoin.Hash32")
			}

			// Retrieve Vote Result
			voteResultTx, err := transactions.GetTx(ctx, c.MasterDB, refTxId, c.Config.IsTest)
			if err != nil {
				return errors.New("Vote Result tx not found for amendment")
			}

			voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
			if !ok {
				return errors.New("Vote Result invalid for amendment")
			}

			// Retrieve the vote
			voteTxId := protocol.TxIdFromBytes(voteResult.VoteTxId)
			vt, err = vote.Retrieve(ctx, c.MasterDB, rk.Address, voteTxId)
			if err == vote.ErrNotFound {
				return errors.New("Vote not found for amendment")
			} else if err != nil {
				return errors.New("Failed to retrieve vote for amendment")
			}
		}
	}

	// Create or update Contract
	if ct == nil {
		// Prepare creation object
		var nc contract.NewContract
		err := node.Convert(ctx, &msg, &nc)
		if err != nil {
			node.LogWarn(ctx, "Failed to convert formation to new contract (%s) : %s", contractName, err.Error())
			return err
		}

		// Get contract offer message to retrieve administration and operator.
		var offerTx *inspector.Transaction
		offerTx, err = transactions.GetTx(ctx, c.MasterDB, &itx.Inputs[0].UTXO.Hash, c.Config.IsTest)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Contract Offer tx not found : %s", itx.Inputs[0].UTXO.Hash.String()))
		}

		// Get offer from it
		offer, ok := offerTx.MsgProto.(*actions.ContractOffer)
		if !ok {
			return fmt.Errorf("Could not find Contract Offer in offer tx")
		}

		nc.AdministrationAddress = offerTx.Inputs[0].Address // First input of offer tx
		if offer.ContractOperatorIncluded && len(offerTx.Inputs) > 1 {
			nc.OperatorAddress = offerTx.Inputs[1].Address // Second input of offer tx
		}

		if err := contract.Create(ctx, c.MasterDB, rk.Address, &nc, v.Now); err != nil {
			node.LogWarn(ctx, "Failed to create contract (%s) : %s", contractName, err)
			return err
		}
		node.Log(ctx, "Created contract (%s)", contractName)
	} else {
		// Prepare update object
		ts := protocol.NewTimestamp(msg.Timestamp)
		uc := contract.UpdateContract{
			Revision:  &msg.ContractRevision,
			Timestamp: &ts,
		}

		// Pull from amendment tx.
		// Administration change. New administration in next input
		inputIndex := 1
		if !ct.OperatorAddress.IsEmpty() {
			inputIndex++
		}
		if amendment != nil && amendment.ChangeAdministrationAddress {
			if len(request.Inputs) <= inputIndex {
				return errors.New("New administration specified but not included in inputs")
			}

			uc.AdministrationAddress = &request.Inputs[inputIndex].Address
			inputIndex++
			address := bitcoin.NewAddressFromRawAddress(*uc.AdministrationAddress, w.Config.Net)
			node.Log(ctx, "Updating contract administration address : %s", address.String())
		}

		// Operator changes. New operator in second input unless there is also a new administration, then it is in the third input
		if amendment != nil && amendment.ChangeOperatorAddress {
			if len(request.Inputs) <= inputIndex {
				return errors.New("New operator specified but not included in inputs")
			}

			uc.OperatorAddress = &request.Inputs[inputIndex].Address
			address := bitcoin.NewAddressFromRawAddress(*uc.OperatorAddress, w.Config.Net)
			node.Log(ctx, "Updating contract operator PKH : %s", address.String())
		}

		// Required pointers
		stringPointer := func(s string) *string { return &s }

		if ct.ContractName != msg.ContractName {
			uc.ContractName = stringPointer(msg.ContractName)
			node.Log(ctx, "Updating contract name (%s) : %s", ct.ContractName, *uc.ContractName)
		}

		if ct.ContractType != msg.ContractType {
			uc.ContractType = stringPointer(msg.ContractType)
			node.Log(ctx, "Updating contract type (%s) : %s", ct.ContractName, *uc.ContractType)
		}

		if ct.BodyOfAgreementType != msg.BodyOfAgreementType {
			uc.BodyOfAgreementType = &msg.BodyOfAgreementType
			node.Log(ctx, "Updating agreement file type (%s) : %02x", ct.ContractName, msg.BodyOfAgreementType)
		}

		if !bytes.Equal(ct.BodyOfAgreement, msg.BodyOfAgreement) {
			uc.BodyOfAgreement = &msg.BodyOfAgreement
			node.Log(ctx, "Updating agreement (%s)", ct.ContractName)
		}

		// Check if SupportingDocs are different
		different := len(ct.SupportingDocs) != len(msg.SupportingDocs)
		if !different {
			for i, doc := range ct.SupportingDocs {
				if !doc.Equal(msg.SupportingDocs[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating contract supporting docs (%s)", ct.ContractName)
			uc.SupportingDocs = &msg.SupportingDocs
		}

		if ct.GoverningLaw != string(msg.GoverningLaw) {
			uc.GoverningLaw = stringPointer(string(msg.GoverningLaw))
			node.Log(ctx, "Updating contract governing law (%s) : %s", ct.ContractName, *uc.GoverningLaw)
		}

		if ct.Jurisdiction != string(msg.Jurisdiction) {
			uc.Jurisdiction = stringPointer(string(msg.Jurisdiction))
			node.Log(ctx, "Updating contract jurisdiction (%s) : %s", ct.ContractName, *uc.Jurisdiction)
		}

		if ct.ContractExpiration.Nano() != msg.ContractExpiration {
			ts := protocol.NewTimestamp(msg.ContractExpiration)
			uc.ContractExpiration = &ts
			newExpiration := time.Unix(int64(msg.ContractExpiration), 0)
			node.Log(ctx, "Updating contract expiration (%s) : %s", ct.ContractName, newExpiration.Format(time.UnixDate))
		}

		if ct.ContractURI != msg.ContractURI {
			uc.ContractURI = stringPointer(msg.ContractURI)
			node.Log(ctx, "Updating contract URI (%s) : %s", ct.ContractName, *uc.ContractURI)
		}

		if !ct.Issuer.Equal(msg.Issuer) {
			uc.Issuer = msg.Issuer
			node.Log(ctx, "Updating contract issuer data (%s)", ct.ContractName)
		}

		if ct.IssuerLogoURL != msg.IssuerLogoURL {
			uc.IssuerLogoURL = stringPointer(msg.IssuerLogoURL)
			node.Log(ctx, "Updating contract issuer logo URL (%s) : %s", ct.ContractName, *uc.IssuerLogoURL)
		}

		if !ct.ContractOperator.Equal(msg.ContractOperator) {
			uc.ContractOperator = msg.ContractOperator
			node.Log(ctx, "Updating contract operator data (%s)", ct.ContractName)
		}

		if !ct.AdminOracle.Equal(msg.AdminOracle) {
			uc.AdminOracle = msg.AdminOracle
			node.Log(ctx, "Updating admin oracle (%s)", ct.ContractName)
		}

		if !bytes.Equal(ct.AdminOracleSignature, msg.AdminOracleSignature) {
			uc.AdminOracleSignature = &msg.AdminOracleSignature
			node.Log(ctx, "Updating admin signature (%s)", ct.ContractName)
		}

		if ct.AdminOracleSigBlockHeight != msg.AdminOracleSigBlockHeight {
			uc.AdminOracleSigBlockHeight = &msg.AdminOracleSigBlockHeight
			node.Log(ctx, "Updating admin oracle sig block height (%s)", ct.ContractName)
		}

		if !bytes.Equal(ct.ContractPermissions[:], msg.ContractPermissions[:]) {
			uc.ContractPermissions = &msg.ContractPermissions
			node.Log(ctx, "Updating contract permissions (%s) : %v", ct.ContractName, *uc.ContractPermissions)
		}

		if ct.ContractFee != msg.ContractFee {
			uc.ContractFee = &msg.ContractFee
			node.Log(ctx, "Updating contract fee (%s) : %d", ct.ContractName, *uc.ContractFee)
		}

		if ct.RestrictedQtyAssets != msg.RestrictedQtyAssets {
			uc.RestrictedQtyAssets = &msg.RestrictedQtyAssets
			node.Log(ctx, "Updating contract restricted quantity assets (%s) : %d", ct.ContractName, *uc.RestrictedQtyAssets)
		}

		if ct.AdministrationProposal != msg.AdministrationProposal {
			uc.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating contract administration proposal (%s) : %t", ct.ContractName, *uc.AdministrationProposal)
		}

		if ct.HolderProposal != msg.HolderProposal {
			uc.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating contract holder proposal (%s) : %t", ct.ContractName, *uc.HolderProposal)
		}

		// Check if oracles are different
		different = len(ct.Oracles) != len(msg.Oracles)
		if !different {
			for i, oracle := range ct.Oracles {
				if !oracle.Equal(msg.Oracles[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating contract oracles (%s)", ct.ContractName)
			uc.Oracles = &msg.Oracles
		}

		// Check if voting systems are different
		different = len(ct.VotingSystems) != len(msg.VotingSystems)
		if !different {
			for i, votingSystem := range ct.VotingSystems {
				if !votingSystem.Equal(msg.VotingSystems[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating contract voting systems (%s)", ct.ContractName)
			uc.VotingSystems = &msg.VotingSystems
		}

		if err := contract.Update(ctx, c.MasterDB, rk.Address, &uc, v.Now); err != nil {
			return errors.Wrap(err, "Failed to update contract")
		}
		node.Log(ctx, "Updated contract (%s)", msg.ContractName)

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId.String())
			if err := vote.MarkApplied(ctx, c.MasterDB, rk.Address, vt.VoteTxId,
				protocol.TxIdFromBytes(request.Hash[:]), v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}
	}

	return nil
}

// AddressChange handles an incoming Contract Address Change.
func (c *Contract) AddressChange(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.AddressChange")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractAddressChange)
	if !ok {
		return errors.New("Could not assert as *actions.ContractAddressChange")
	}

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract address change request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Check that it is from the master PKH
	if !itx.Inputs[0].Address.Equal(ct.MasterAddress) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogWarn(ctx, "Contract address change must be from master address : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsTxMalformed)
	}

	newContractAddress, err := bitcoin.DecodeRawAddress(msg.NewContractAddress)
	if err != nil {
		node.LogWarn(ctx, "Invalid new contract address : %x", msg.NewContractAddress)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsTxMalformed)
	}

	// Check that it is to the current contract address and the new contract address
	toCurrent := false
	toNew := false
	for _, output := range itx.Outputs {
		if output.Address.Equal(rk.Address) {
			toCurrent = true
		}
		if output.Address.Equal(newContractAddress) {
			toNew = true
		}
	}

	if !toCurrent || !toNew {
		node.LogWarn(ctx, "Contract address change must be to current and new PKH")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsTxMalformed)
	}

	// Perform move
	err = contract.Move(ctx, c.MasterDB, rk.Address, newContractAddress,
		protocol.NewTimestamp(msg.Timestamp))
	if err != nil {
		return err
	}

	//TODO Transfer all UTXOs to fee address.

	return nil
}

// applyOracleAmendments applies the amendments to the oracle field.
func applyOracleAmendments(oracle *actions.OracleField, amendment *actions.AmendmentField,
	parentFIP, fip protocol.FieldIndexPath) error {

	if len(fip) == 0 {
		return errors.New("Amendments on complex fields (Oracle) not allowed")
	}

	switch fip[0] {
	case 0: // Name
		oracle.Name = string(amendment.Data)

	case 1: // URL
		oracle.URL = string(amendment.Data)

	case 2: // PublicKey
		if _, err := bitcoin.DecodePublicKeyBytes(amendment.Data); err != nil {
			return errors.Wrap(err, "AdminOracle public key invalid")
		}
		oracle.PublicKey = amendment.Data

	default:
		return fmt.Errorf("Contract amendment subfield offset for Oracle out of range : %s", fip.String())
	}

	return nil
}

// applyEntityAmendments applies the amendments to the entity field.
func applyEntityAmendments(entity *actions.EntityField, amendment *actions.AmendmentField,
	parentFIP, fip protocol.FieldIndexPath) error {

	if len(fip) == 0 {
		return errors.New("Amendments on complex fields (Entity) not allowed")
	}

	switch fip[0] {
	case actions.EntityFieldName:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.Name : %s",
				fip.String())
		}
		entity.Name = string(amendment.Data)

	case actions.EntityFieldType:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.Type : %s",
				fip.String())
		}
		if len(amendment.Data) != 1 {
			return fmt.Errorf("Entity.Type amendment value is wrong size : %d", len(amendment.Data))
		}
		entity.Type = string([]byte{amendment.Data[0]})

	case actions.EntityFieldLEI:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.LEI : %s",
				fip.String())
		}
		entity.LEI = string(amendment.Data)

	case actions.EntityFieldUnitNumber:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.UnitNumber : %s",
				fip.String())
		}
		entity.UnitNumber = string(amendment.Data)

	case actions.EntityFieldBuildingNumber:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.BuildingNumber : %s",
				fip.String())
		}
		entity.BuildingNumber = string(amendment.Data)

	case actions.EntityFieldStreet:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.Street : %s",
				fip.String())
		}
		entity.Street = string(amendment.Data)

	case actions.EntityFieldSuburbCity:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.SuburbCity : %s",
				fip.String())
		}
		entity.SuburbCity = string(amendment.Data)

	case actions.EntityFieldTerritoryStateProvinceCode:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.TerritoryStateProvinceCode : %s",
				fip.String())
		}
		entity.TerritoryStateProvinceCode = string(amendment.Data)

	case actions.EntityFieldCountryCode:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.CountryCode : %s",
				fip.String())
		}
		entity.CountryCode = string(amendment.Data)

	case actions.EntityFieldPostalZIPCode:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.ZIPCode : %s",
				fip.String())
		}
		entity.PostalZIPCode = string(amendment.Data)

	case actions.EntityFieldEmailAddress:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.EmailAddress : %s",
				fip.String())
		}
		entity.EmailAddress = string(amendment.Data)

	case actions.EntityFieldPhoneNumber:
		if len(fip) > 1 {
			return fmt.Errorf("Contract amendment field index path too deep for Entity.PhoneNumber : %s",
				fip.String())
		}
		entity.PhoneNumber = string(amendment.Data)

	case actions.EntityFieldAdministration: // []*actions.Administrator
		switch amendment.Operation {
		case 0: // Modify
			if len(fip) != 2 { // includes list index
				return fmt.Errorf("Contract amendment field index path incorrect depth for modify Entity.Administration : %s",
					fip.String())
			}
			if int(fip[1]) >= len(entity.Administration) {
				return fmt.Errorf("Contract amendment subfield element out of range for Entity.Administration : %d",
					fip[1])
			}

			entity.Administration[fip[1]].Reset()
			if len(amendment.Data) != 0 {
				if err := proto.Unmarshal(amendment.Data, entity.Administration[fip[1]]); err != nil {
					return fmt.Errorf("Contract amendment Entity.Administration[%d] failed to deserialize : %s",
						fip[1], err)
				}
			}

		case 1: // Add element
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for add Entity.Administration : %s",
					fip.String())
			}
			newAdministrator := actions.AdministratorField{}
			if len(amendment.Data) != 0 {
				if err := proto.Unmarshal(amendment.Data, &newAdministrator); err != nil {
					return fmt.Errorf("Contract amendment addition to Entity.Administration failed to deserialize : %s",
						err)
				}
			}
			entity.Administration = append(entity.Administration, &newAdministrator)

		case 2: // Delete element
			if len(fip) != 2 { // includes list index
				return fmt.Errorf("Contract amendment field index path incorrect depth for delete Entity.Administration : %s",
					fip.String())
			}
			if int(fip[1]) >= len(entity.Administration) {
				return fmt.Errorf("Contract amendment subfield element out of range for Entity.Administration : %d",
					fip[1])
			}
			entity.Administration = append(entity.Administration[:fip[1]],
				entity.Administration[fip[1]+1:]...)

		default:
			return fmt.Errorf("Invalid contract amendment operation for Entity.Administration : %d",
				amendment.Operation)
		}

	case actions.EntityFieldManagement: // []*actions.Manager
		switch amendment.Operation {
		case 0: // Modify
			if len(fip) != 2 { // includes list index
				return fmt.Errorf("Contract amendment field index path incorrect depth for modify Entity.Management : %s",
					fip.String())
			}
			if int(fip[1]) >= len(entity.Management) {
				return fmt.Errorf("Contract amendment subfield element out of range for Entity.Management : %d",
					fip[1])
			}

			entity.Management[fip[1]].Reset()
			if len(amendment.Data) != 0 {
				if err := proto.Unmarshal(amendment.Data, entity.Management[fip[1]]); err != nil {
					return fmt.Errorf("Contract amendment Entity.Management[%d] failed to deserialize : %s",
						fip[1], err)
				}
			}

		case 1: // Add element
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for add Entity.Management : %s",
					fip.String())
			}
			newManager := actions.ManagerField{}
			if len(amendment.Data) != 0 {
				if err := proto.Unmarshal(amendment.Data, &newManager); err != nil {
					return fmt.Errorf("Contract amendment addition to Entity.Management failed to deserialize : %s",
						err)
				}
			}
			entity.Management = append(entity.Management, &newManager)

		case 2: // Delete element
			if len(fip) != 2 { // includes list index
				return fmt.Errorf("Contract amendment field index path incorrect depth for delete Entity.Management : %s",
					fip.String())
			}
			if int(fip[1]) >= len(entity.Management) {
				return fmt.Errorf("Contract amendment subfield element out of range for Entity.Management : %d",
					fip[1])
			}
			entity.Management = append(entity.Management[:fip[1]],
				entity.Management[fip[1]+1:]...)

		default:
			return fmt.Errorf("Invalid contract amendment operation for Entity.Management : %d",
				amendment.Operation)
		}

	default:
		return fmt.Errorf("Contract amendment subfield offset for Entity out of range : %s",
			fip.String())
	}

	return nil
}

// applyContractAmendments applies the amendments to the contract formation.
func applyContractAmendments(cf *actions.ContractFormation, amendments []*actions.AmendmentField,
	proposed bool, proposalInitiator, votingSystem uint32) error {

	permissions, err := protocol.PermissionsFromBytes(cf.ContractPermissions, len(cf.VotingSystems))
	if err != nil {
		return fmt.Errorf("Invalid contract permissions : %s", err)
	}

	for i, amendment := range amendments {
		fip, err := protocol.FieldIndexPathFromBytes(amendment.FieldIndexPath)
		if err != nil {
			return fmt.Errorf("Failed to read amendment %d field index path : %s", i, err)
		}
		if len(fip) == 0 {
			return fmt.Errorf("Amendment %d has no field specified", i)
		}

		permission := permissions.PermissionOf(fip)

		if proposed {
			switch proposalInitiator {
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
			default:
				return fmt.Errorf("Invalid proposal initiator type : %d", proposalInitiator)
			}

			if int(votingSystem) >= len(permission.VotingSystemsAllowed) {
				return fmt.Errorf("Field %s amendment voting system out of range : %d", fip,
					votingSystem)
			}
			if !permission.VotingSystemsAllowed[votingSystem] {
				return node.NewError(actions.RejectionsContractPermissions,
					fmt.Sprintf("Field %s amendment not allowed using voting system %d", fip,
						votingSystem))
			}
		} else if !permission.Permitted {
			return node.NewError(actions.RejectionsContractPermissions,
				fmt.Sprintf("Field %s amendment not permitted without proposal", fip))
		}

		switch fip[0] {
		case actions.ContractFieldContractName:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractName : %s",
					fip.String())
			}
			cf.ContractName = string(amendment.Data)

		case actions.ContractFieldBodyOfAgreementType:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for BodyOfAgreementType : %s",
					fip.String())
			}
			if len(amendment.Data) != 1 {
				return fmt.Errorf("BodyOfAgreementType amendment value is wrong size : %d", len(amendment.Data))
			}
			cf.BodyOfAgreementType = uint32(amendment.Data[0])

		case actions.ContractFieldBodyOfAgreement:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for BodyOfAgreement : %s",
					fip.String())
			}
			cf.BodyOfAgreement = amendment.Data

		case actions.ContractFieldContractType:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractType : %s",
					fip.String())
			}
			cf.ContractType = string(amendment.Data)

		case actions.ContractFieldSupportingDocs:
			switch amendment.Operation {
			case 0: // Modify
				if len(fip) != 2 { // includes list index
					return fmt.Errorf("Contract amendment field index path incorrect depth for modify SupportingDocs : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.SupportingDocs) {
					return fmt.Errorf("Contract amendment element out of range for SupportingDocs : %d",
						fip[1])
				}

				cf.SupportingDocs[fip[1]].Reset()
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, cf.SupportingDocs[fip[1]]); err != nil {
						return fmt.Errorf("Contract amendment SupportingDocs[%d] failed to deserialize : %s",
							fip[1], err)
					}
				}

			case 1: // Add element
				if len(fip) > 1 {
					return fmt.Errorf("Contract amendment field index path too deep for add SupportingDocs : %s",
						fip.String())
				}
				newDocument := actions.DocumentField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newDocument); err != nil {
						return fmt.Errorf("Contract amendment addition to SupportingDocs failed to deserialize : %s",
							err)
					}
				}
				cf.SupportingDocs = append(cf.SupportingDocs, &newDocument)

			case 2: // Delete element
				if len(fip) != 2 { // includes list index
					return fmt.Errorf("Contract amendment field index path incorrect depth for delete SupportingDocs : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.SupportingDocs) {
					return fmt.Errorf("Contract amendment element out of range for SupportingDocs : %d",
						fip[1])
				}
				cf.SupportingDocs = append(cf.SupportingDocs[:fip[1]],
					cf.SupportingDocs[fip[1]+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for SupportingDocs : %d",
					amendment.Operation)
			}

		case actions.ContractFieldGoverningLaw:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for GoverningLaw : %s",
					fip.String())
			}
			cf.GoverningLaw = string(amendment.Data)

		case actions.ContractFieldJurisdiction:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for Jurisdiction : %s",
					fip.String())
			}
			cf.Jurisdiction = string(amendment.Data)

		case actions.ContractFieldContractExpiration:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractExpiration : %s",
					fip.String())
			}
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractExpiration amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewReader(amendment.Data)
			contractExpiration, err := protocol.DeserializeTimestamp(buf)
			if err != nil {
				return fmt.Errorf("ContractExpiration amendment value failed to deserialize : %s", err)
			}
			cf.ContractExpiration = contractExpiration.Nano()

		case actions.ContractFieldContractURI:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractURI : %s",
					fip.String())
			}
			cf.ContractURI = string(amendment.Data)

		case actions.ContractFieldIssuer:
			err := applyEntityAmendments(cf.Issuer, amendment, fip[:1], fip[1:])
			if err != nil {
				return errors.Wrap(err, "apply entity amendment")
			}

		case actions.ContractFieldIssuerLogoURL:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for IssuerLogoURL : %s",
					fip.String())
			}
			cf.IssuerLogoURL = string(amendment.Data)

		case actions.ContractFieldContractOperator:
			err := applyEntityAmendments(cf.ContractOperator, amendment, fip[:1], fip[1:])
			if err != nil {
				return errors.Wrap(err, "apply entity amendment")
			}

		case actions.ContractFieldAdminOracle:
			err := applyOracleAmendments(cf.AdminOracle, amendment, fip[:1], fip[1:])
			if err != nil {
				return errors.Wrap(err, "apply oracle amendment")
			}

		case actions.ContractFieldAdminOracleSignature:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for OracleSignature : %s",
					fip.String())
			}
			if _, err := bitcoin.DecodeSignatureBytes(amendment.Data); err != nil {
				return errors.Wrap(err, "AdminOracleSignature invalid")
			}
			cf.AdminOracleSignature = amendment.Data

		case actions.ContractFieldAdminOracleSigBlockHeight:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for SigBlockHeight : %s",
					fip.String())
			}
			if len(amendment.Data) != 4 {
				return fmt.Errorf("AdminOracleSigBlockHeight amendment value is wrong size : %d",
					len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.AdminOracleSigBlockHeight); err != nil {
				return fmt.Errorf("AdminOracleSigBlockHeight amendment value failed to deserialize : %s", err)
			}

		case actions.ContractFieldContractPermissions:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractPermissions : %s",
					fip.String())
			}
			_, err := protocol.PermissionsFromBytes(amendment.Data, len(cf.VotingSystems))
			if err != nil {
				return fmt.Errorf("Invalid amendment data for ContractPermissions : %s", err)
			}
			cf.ContractPermissions = amendment.Data

		case actions.ContractFieldContractFee:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for ContractFee : %s",
					fip.String())
			}
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractFee amendment value is wrong size : %d",
					len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.ContractFee); err != nil {
				return fmt.Errorf("ContractFee amendment value failed to deserialize : %s", err)
			}

		case actions.ContractFieldVotingSystems:
			switch amendment.Operation {
			case 0: // Modify
				if len(fip) != 2 { // includes list index
					return fmt.Errorf("Contract amendment field index path incorrect depth for modify VotingSystems : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.VotingSystems) {
					return fmt.Errorf("Contract amendment element out of range for VotingSystems : %d",
						fip[1])
				}

				cf.VotingSystems[fip[1]].Reset()
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, cf.VotingSystems[fip[1]]); err != nil {
						return fmt.Errorf("Contract amendment VotingSystems[%d] failed to deserialize : %s",
							fip[1], err)
					}
				}

			case 1: // Add element
				if len(fip) > 1 {
					return fmt.Errorf("Contract amendment field index path too deep for add VotingSystems : %s",
						fip.String())
				}
				newVotingSystem := actions.VotingSystemField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newVotingSystem); err != nil {
						return fmt.Errorf("Contract amendment addition to VotingSystems failed to deserialize : %s",
							err)
					}
				}

				cf.VotingSystems = append(cf.VotingSystems, &newVotingSystem)

			case 2: // Delete element
				if len(fip) != 2 { // includes list index
					return fmt.Errorf("Contract amendment field index path incorrect depth for delete VotingSystems : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.VotingSystems) {
					return fmt.Errorf("Contract amendment element out of range for VotingSystems : %d",
						fip[1])
				}
				cf.VotingSystems = append(cf.VotingSystems[:fip[1]], cf.VotingSystems[fip[1]+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for VotingSystems : %d",
					amendment.Operation)
			}

		case actions.ContractFieldRestrictedQtyAssets:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for RestrictedQtyAssets : %s",
					fip.String())
			}
			if len(amendment.Data) != 8 {
				return fmt.Errorf("RestrictedQtyAssets amendment value is wrong size : %d",
					len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.RestrictedQtyAssets); err != nil {
				return fmt.Errorf("RestrictedQtyAssets amendment value failed to deserialize : %s", err)
			}

		case actions.ContractFieldAdministrationProposal:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for AdministrationProposal : %s",
					fip.String())
			}
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AdministrationProposal amendment value is wrong size : %d",
					len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.AdministrationProposal); err != nil {
				return fmt.Errorf("AdministrationProposal amendment value failed to deserialize : %s", err)
			}

		case actions.ContractFieldHolderProposal:
			if len(fip) > 1 {
				return fmt.Errorf("Contract amendment field index path too deep for HolderProposal : %s",
					fip.String())
			}
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d",
					len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case actions.ContractFieldOracles:
			switch amendment.Operation {
			case 0: // Modify
				if len(fip) < 3 { // includes list index
					return fmt.Errorf("Contract amendment field index path too shallow for modify Oracles : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.Oracles) {
					return fmt.Errorf("Contract amendment element out of range for Oracles : %d",
						fip[1])
				}

				err := applyOracleAmendments(cf.Oracles[fip[1]], amendment, fip[:2], fip[2:])
				if err != nil {
					return errors.Wrap(err, "apply oracle amendment")
				}

			case 1: // Add element
				if len(fip) > 1 {
					return fmt.Errorf("Contract amendment field index path too deep for add Oracles : %s",
						fip.String())
				}
				newOracle := actions.OracleField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newOracle); err != nil {
						return fmt.Errorf("Contract amendment addition to Oracles failed to deserialize : %s",
							err)
					}
				}

				cf.Oracles = append(cf.Oracles, &newOracle)

			case 2: // Delete element
				if len(fip) != 2 { // includes list index
					return fmt.Errorf("Contract amendment field index path incorrect depth for delete Oracles : %s",
						fip.String())
				}
				if int(fip[1]) >= len(cf.Oracles) {
					return fmt.Errorf("Contract amendment element out of range for Oracles : %d",
						fip[1])
				}
				cf.Oracles = append(cf.Oracles[:fip[1]], cf.Oracles[fip[1]+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for Oracles : %d",
					amendment.Operation)
			}

		default:
			return fmt.Errorf("Contract amendment field offset out of range : %s", fip.String())
		}
	}

	// Check validity of updated contract data
	if err := cf.Validate(); err != nil {
		return fmt.Errorf("Contract data invalid after amendments : %s", err)
	}

	return nil
}

func validateContractAmendOracleSig(ctx context.Context, updatedContract *state.Contract,
	headers node.BitcoinHeaders) error {

	oracle, err := bitcoin.DecodePublicKeyBytes(updatedContract.AdminOracle.PublicKey)
	if err != nil {
		return err
	}

	// Parse signature
	oracleSig, err := bitcoin.DecodeSignatureBytes(updatedContract.AdminOracleSignature)
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	hash, err := headers.Hash(ctx, int(updatedContract.AdminOracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
			updatedContract.AdminOracleSigBlockHeight))
	}

	addresses := make([]bitcoin.RawAddress, 0, 2)
	entities := make([]*actions.EntityField, 0, 2)

	addresses = append(addresses, updatedContract.AdministrationAddress)
	entities = append(entities, updatedContract.Issuer)

	if !updatedContract.OperatorAddress.IsEmpty() {
		addresses = append(addresses, updatedContract.OperatorAddress)
		entities = append(entities, updatedContract.ContractOperator)
	}

	sigHash, err := protocol.ContractOracleSigHash(ctx, addresses, entities, hash, 1)
	if err != nil {
		return err
	}

	if oracleSig.Verify(sigHash, oracle) {
		return nil // Valid signature found
	}

	return fmt.Errorf("Contract signature invalid")
}
