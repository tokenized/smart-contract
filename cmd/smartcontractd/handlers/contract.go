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
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
	TxCache  InspectorTxCache
}

// OfferRequest handles an incoming Contract Offer and prepares a Formation response
func (c *Contract) OfferRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Offer")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractOffer")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Contract offer invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	_, err := contract.Retrieve(ctx, c.MasterDB, contractPKH)
	if err != contract.ErrNotFound {
		if err == nil {
			logger.Warn(ctx, "%s : Contract already exists : %s", v.TraceID, contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve contract")
		}
	}

	if msg.BodyOfAgreementType == 1 {
		if len(msg.BodyOfAgreement) != 32 {
			logger.Warn(ctx, "%s : Contract body of agreement hash is incorrect length : %s : %d", v.TraceID, contractPKH.String(), len(msg.BodyOfAgreement))
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	} else if msg.BodyOfAgreementType != 0 {
		logger.Warn(ctx, "%s : Invalid contract body of agreement type : %s : %d", v.TraceID, contractPKH.String(), msg.BodyOfAgreementType)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if msg.SupportingDocsFileType == 1 {
		if len(msg.SupportingDocs) != 32 {
			logger.Warn(ctx, "%s : Contract supporting docs hash is incorrect length : %s : %d", v.TraceID, contractPKH.String(), len(msg.SupportingDocs))
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	} else if msg.SupportingDocsFileType != 0 {
		logger.Warn(ctx, "%s : Invalid contract body of agreement type : %s : %d", v.TraceID, contractPKH.String(), msg.SupportingDocsFileType)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if msg.ContractExpiration.Nano() != 0 && msg.ContractExpiration.Nano() < v.Now.Nano() {
		logger.Warn(ctx, "%s : Expiration already passed : %s : %d", v.TraceID, contractPKH.String(), msg.ContractExpiration.Nano())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if _, err = protocol.ReadAuthFlags(msg.ContractAuthFlags, contract.FieldCount, len(msg.VotingSystems)); err != nil {
		logger.Warn(ctx, "%s : Invalid contract auth flags : %s : %s", v.TraceID, contractPKH.String(), err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Validate voting systems are all valid.
	for _, votingSystem := range msg.VotingSystems {
		if err = vote.ValidateVotingSystem(&votingSystem); err != nil {
			logger.Warn(ctx, "%s : Invalid voting system : %s : %s", v.TraceID, contractPKH.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	}

	logger.Info(ctx, "%s : Accepting contract offer (%s) : %s", v.TraceID, msg.ContractName, contractPKH.String())

	// Contract Formation <- Contract Offer
	cf := protocol.ContractFormation{}

	err = node.Convert(ctx, &msg, &cf)
	if err != nil {
		return err
	}

	cf.ContractRevision = 0
	cf.Timestamp = v.Now

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &c.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, msg.ContractFee)

	// Save Tx for when formation is processed.
	c.TxCache.SaveTx(ctx, itx)

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &cf)
}

// AmendmentRequest handles an incoming Contract Amendment and prepares a Formation response
func (c *Contract) AmendmentRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Amendment")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractAmendment)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractAmendment")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Contract amendment request invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, c.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		logger.Verbose(ctx, "%s : Requestor is not operator : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if ct.RestrictedQtyAssets > 0 && ct.RestrictedQtyAssets < uint64(len(ct.AssetCodes)) {
		logger.Warn(ctx, "%s : Cannot reduce allowable assets below existing number: %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractAssetQtyReduction)
	}

	if ct.Revision != msg.ContractRevision {
		logger.Warn(ctx, "%s : Incorrect contract revision (%s) : specified %d != current %d", v.TraceID, ct.ContractName, msg.ContractRevision, ct.Revision)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalInitiator := uint8(0)
	votingSystem := uint8(0)

	if !msg.RefTxID.IsZero() { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := chainhash.NewHash(msg.RefTxID.Bytes())
		if err != nil {
			return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
		}

		// Retrieve Vote Result
		voteResultTx := c.TxCache.GetTx(ctx, refTxId)
		if voteResultTx == nil {
			logger.Warn(ctx, "%s : Vote Result tx not found for amendment", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*protocol.Result)
		if !ok {
			logger.Warn(ctx, "%s : Vote Result invalid for amendment", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Retrieve the vote
		vt, err := vote.Retrieve(ctx, c.MasterDB, contractPKH, &voteResult.VoteTxId)
		if err == vote.ErrNotFound {
			logger.Warn(ctx, "%s : Vote not found : %s", v.TraceID, voteResult.VoteTxId.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteNotFound)
		} else if err != nil {
			logger.Warn(ctx, "%s : Failed to retrieve vote : %s : %s", v.TraceID, voteResult.VoteTxId.String(), err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			logger.Warn(ctx, "%s : Vote not complete yet", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if vt.Result != "A" {
			logger.Warn(ctx, "%s : Vote result not A(Accept)", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if !vt.Specific {
			logger.Warn(ctx, "%s : Vote was not for specific amendments", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if vt.AssetSpecificVote {
			logger.Warn(ctx, "%s : Vote was not for contract amendments", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			logger.Warn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				logger.Warn(ctx, "%s : Proposal amendment %d doesn't match", v.TraceID, i)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	if err := checkContractAmendmentsPermissions(ct, msg.Amendments, proposed, proposalInitiator, votingSystem); err != nil {
		logger.Warn(ctx, "%s : Contract amendments not permitted : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractAuthFlags)
	}

	logger.Info(ctx, "%s : Accepting contract amendment (%s) : %s", v.TraceID, ct.ContractName, contractPKH.String())

	// Contract Formation <- Contract Amendment
	cf := protocol.ContractFormation{}

	// Get current state
	err = node.Convert(ctx, &ct, &cf)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state contract to contract formation")
	}

	// Apply modifications
	cf.ContractRevision = ct.Revision + 1 // Bump the revision
	cf.Timestamp = v.Now

	if err := applyContractAmendments(&cf, msg.Amendments); err != nil {
		logger.Warn(ctx, "%s : Contract amendments failed : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &c.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to convert contract PKH to address")
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Issuer change. New issuer in second input
	if msg.ChangeIssuerAddress {
		if len(itx.Inputs) < 2 {
			logger.Warn(ctx, "%s : New issuer specified but not included in inputs (%s)", v.TraceID, ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectTxMalformed)
		}
	}

	// Operator changes. New operator in second input unless there is also a new issuer, then it is in the third input
	if msg.ChangeOperatorAddress {
		index := 1
		if msg.ChangeIssuerAddress {
			index++
		}
		if index >= len(itx.Inputs) {
			logger.Warn(ctx, "%s : New operator specified but not included in inputs (%s)", v.TraceID, ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectTxMalformed)
		}
	}

	// Save Tx.
	c.TxCache.SaveTx(ctx, itx)

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &cf)
}

// FormationResponse handles an outgoing Contract Formation and writes it to the state
func (c *Contract) FormationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Formation")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.ContractFormation)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractFormation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract. Sender is verified to be contract before this response function is called.
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Contract formation not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	contractName := msg.ContractName
	ct, err := contract.Retrieve(ctx, c.MasterDB, contractPKH)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Get request tx
	request := c.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
	var vt *state.Vote
	var amendment *protocol.ContractAmendment
	if request != nil {
		c.TxCache.RemoveTx(ctx, &itx.Inputs[0].UTXO.Hash)
		var ok bool
		amendment, ok = request.MsgProto.(*protocol.ContractAmendment)

		if ok && !amendment.RefTxID.IsZero() {
			refTxId, err := chainhash.NewHash(amendment.RefTxID.Bytes())
			if err != nil {
				return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
			}

			// Retrieve Vote Result
			voteResultTx := c.TxCache.GetTx(ctx, refTxId)
			if voteResultTx != nil {
				return errors.New("Vote Result tx not found for amendment")
			}

			voteResult, ok := voteResultTx.MsgProto.(*protocol.Result)
			if !ok {
				return errors.New("Vote Result invalid for amendment")
			}

			// Retrieve the vote
			vt, err = vote.Retrieve(ctx, c.MasterDB, contractPKH, &voteResult.VoteTxId)
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
			logger.Warn(ctx, "%s : Failed to convert formation to new contract (%s) : %s", v.TraceID, contractName, err.Error())
			return err
		}

		// Get contract offer message to retrieve issuer and operator.
		var offerTx *inspector.Transaction
		offerTx = c.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
		if offerTx == nil {
			return errors.New("ContractOffer tx not found")
		}

		// Get offer from it
		offer, ok := offerTx.MsgProto.(*protocol.ContractOffer)
		if ok {
			return errors.New("Could not find ContractOffer in offer tx")
		}

		nc.IssuerPKH = *protocol.PublicKeyHashFromBytes(offerTx.Inputs[0].Address.ScriptAddress()) // First input of offer tx
		if offer.ContractOperatorIncluded && len(offerTx.Inputs) > 1 {
			nc.OperatorPKH = *protocol.PublicKeyHashFromBytes(offerTx.Inputs[1].Address.ScriptAddress()) // Second input of offer tx
		}

		if err := contract.Create(ctx, c.MasterDB, contractPKH, &nc, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to create contract (%s) : %s", v.TraceID, contractName, err.Error())
			return err
		}
		logger.Info(ctx, "%s : Created contract (%s) : %s", v.TraceID, contractName, contractPKH.String())
	} else {
		// Prepare update object
		uc := contract.UpdateContract{
			Revision:  &msg.ContractRevision,
			Timestamp: &msg.Timestamp,
		}

		// Pull from amendment tx.
		// Issuer change. New issuer in second input
		if amendment != nil && amendment.ChangeIssuerAddress {
			if len(request.Inputs) < 2 {
				return errors.New("New issuer specified but not included in inputs")
			}

			uc.IssuerPKH = protocol.PublicKeyHashFromBytes(request.Inputs[1].Address.ScriptAddress())
		}

		// Operator changes. New operator in second input unless there is also a new issuer, then it is in the third input
		if amendment != nil && amendment.ChangeOperatorAddress {
			index := 1
			if amendment.ChangeIssuerAddress {
				index++
			}
			if index >= len(request.Inputs) {
				return errors.New("New operator specified but not included in inputs")
			}

			uc.OperatorPKH = protocol.PublicKeyHashFromBytes(request.Inputs[index].Address.ScriptAddress())
		}

		// Required pointers
		stringPointer := func(s string) *string { return &s }

		if !bytes.Equal(ct.IssuerPKH.Bytes(), uc.IssuerPKH.Bytes()) {
			// TODO Move asset balances from previous issuer to new issuer.
			uc.IssuerPKH = protocol.PublicKeyHashFromBytes(itx.Outputs[1].Address.ScriptAddress())
			logger.Info(ctx, "%s : Updating contract issuer address (%s) : %s", v.TraceID, ct.ContractName, itx.Outputs[1].Address.String())
		}

		if ct.ContractName != msg.ContractName {
			uc.ContractName = stringPointer(msg.ContractName)
			logger.Info(ctx, "%s : Updating contract name (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractName)
		}

		if ct.ContractType != msg.ContractType {
			uc.ContractType = stringPointer(msg.ContractType)
			logger.Info(ctx, "%s : Updating contract type (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractType)
		}

		if ct.BodyOfAgreementType != msg.BodyOfAgreementType {
			uc.BodyOfAgreementType = &msg.BodyOfAgreementType
			logger.Info(ctx, "%s : Updating agreement file type (%s) : %02x", v.TraceID, ct.ContractName, msg.BodyOfAgreementType)
		}

		if !bytes.Equal(ct.BodyOfAgreement, msg.BodyOfAgreement) {
			uc.BodyOfAgreement = &msg.BodyOfAgreement
			logger.Info(ctx, "%s : Updating agreement (%s)", v.TraceID, ct.ContractName)
		}

		if ct.SupportingDocsFileType != msg.SupportingDocsFileType {
			uc.SupportingDocsFileType = &msg.SupportingDocsFileType
			logger.Info(ctx, "%s : Updating supporting docs file type (%s) : %02x", v.TraceID, ct.ContractName, msg.SupportingDocsFileType)
		}

		if !bytes.Equal(ct.SupportingDocs, msg.SupportingDocs) {
			uc.SupportingDocs = &msg.SupportingDocs
			logger.Info(ctx, "%s : Updating supporting docs (%s)", v.TraceID, ct.ContractName)
		}

		if ct.GoverningLaw != string(msg.GoverningLaw) {
			uc.GoverningLaw = stringPointer(string(msg.GoverningLaw))
			logger.Info(ctx, "%s : Updating contract governing law (%s) : %s", v.TraceID, ct.ContractName, *uc.GoverningLaw)
		}

		if ct.Jurisdiction != string(msg.Jurisdiction) {
			uc.Jurisdiction = stringPointer(string(msg.Jurisdiction))
			logger.Info(ctx, "%s : Updating contract jurisdiction (%s) : %s", v.TraceID, ct.ContractName, *uc.Jurisdiction)
		}

		if ct.ContractExpiration.Nano() != msg.ContractExpiration.Nano() {
			uc.ContractExpiration = &msg.ContractExpiration
			newExpiration := time.Unix(int64(msg.ContractExpiration.Nano()), 0)
			logger.Info(ctx, "%s : Updating contract expiration (%s) : %s", v.TraceID, ct.ContractName, newExpiration.Format(time.UnixDate))
		}

		if ct.ContractURI != msg.ContractURI {
			uc.ContractURI = stringPointer(msg.ContractURI)
			logger.Info(ctx, "%s : Updating contract URI (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractURI)
		}

		if !ct.Issuer.Equal(msg.Issuer) {
			uc.Issuer = &msg.Issuer
			logger.Info(ctx, "%s : Updating contract issuer data (%s) : %02x", v.TraceID, ct.ContractName, *uc.Issuer)
		}

		if ct.IssuerLogoURL != msg.IssuerLogoURL {
			uc.IssuerLogoURL = stringPointer(msg.IssuerLogoURL)
			logger.Info(ctx, "%s : Updating contract issuer logo URL (%s) : %s", v.TraceID, ct.ContractName, *uc.IssuerLogoURL)
		}

		if !ct.ContractOperator.Equal(msg.ContractOperator) {
			uc.ContractOperator = &msg.ContractOperator
			logger.Info(ctx, "%s : Updating contract operator data (%s) : %s", v.TraceID, ct.ContractName, *uc.ContractOperator)
		}

		if !bytes.Equal(ct.ContractAuthFlags[:], msg.ContractAuthFlags[:]) {
			uc.ContractAuthFlags = &msg.ContractAuthFlags
			logger.Info(ctx, "%s : Updating contract auth flags (%s) : %v", v.TraceID, ct.ContractName, *uc.ContractAuthFlags)
		}

		if ct.ContractFee != msg.ContractFee {
			uc.ContractFee = &msg.ContractFee
			logger.Info(ctx, "%s : Updating contract fee (%s) : %d", v.TraceID, ct.ContractName, *uc.ContractFee)
		}

		if ct.RestrictedQtyAssets != msg.RestrictedQtyAssets {
			uc.RestrictedQtyAssets = &msg.RestrictedQtyAssets
			logger.Info(ctx, "%s : Updating contract restricted quantity assets (%s) : %d", v.TraceID, ct.ContractName, *uc.RestrictedQtyAssets)
		}

		if ct.IssuerProposal != msg.IssuerProposal {
			uc.IssuerProposal = &msg.IssuerProposal
			logger.Info(ctx, "%s : Updating contract issuer proposal (%s) : %t", v.TraceID, ct.ContractName, *uc.IssuerProposal)
		}

		if ct.HolderProposal != msg.HolderProposal {
			uc.HolderProposal = &msg.HolderProposal
			logger.Info(ctx, "%s : Updating contract holder proposal (%s) : %t", v.TraceID, ct.ContractName, *uc.HolderProposal)
		}

		// Check if registries are different
		different := len(ct.Registers) != len(msg.Registers)
		if !different {
			for i, register := range ct.Registers {
				if !register.Equal(msg.Registers[i]) {
					different = true
					break
				}
			}
		}

		if different {
			logger.Info(ctx, "%s : Updating contract registers (%s)", v.TraceID, ct.ContractName)
			uc.Registers = &msg.Registers
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
			logger.Info(ctx, "%s : Updating contract voting systems (%s)", v.TraceID, ct.ContractName)
			uc.VotingSystems = &msg.VotingSystems
		}

		if err := contract.Update(ctx, c.MasterDB, contractPKH, &uc, v.Now); err != nil {
			return errors.Wrap(err, "Failed to update contract")
		}
		logger.Info(ctx, "%s : Updated contract (%s) : %s", v.TraceID, msg.ContractName, contractPKH.String())

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			uv := vote.UpdateVote{AppliedTxId: protocol.TxIdFromBytes(request.Hash[:])}
			if err := vote.Update(ctx, c.MasterDB, contractPKH, &vt.VoteTxId, &uv, v.Now); err != nil {
				return errors.Wrap(err, "Failed to update vote")
			}
		}

		if request != nil {
			c.TxCache.RemoveTx(ctx, &request.Hash)
		}
	}

	return nil
}

// checkContractAmendmentsPermissions verifies that the amendments are permitted bases on the auth flags.
func checkContractAmendmentsPermissions(ct *state.Contract, amendments []protocol.Amendment, proposed bool,
	proposalInitiator, votingSystem uint8) error {

	permissions, err := protocol.ReadAuthFlags(ct.ContractAuthFlags, contract.FieldCount, len(ct.VotingSystems))
	if err != nil {
		return fmt.Errorf("Invalid contract auth flags : %s", err)
	}

	for _, amendment := range amendments {
		if int(amendment.FieldIndex) >= len(permissions) {
			return fmt.Errorf("Amendment field index out of range : %d", amendment.FieldIndex)
		}
		if proposed {
			switch proposalInitiator {
			case 0: // Issuer
				if !permissions[amendment.FieldIndex].IssuerProposal {
					return fmt.Errorf("Field %d amendment not permitted by issuer proposal", amendment.FieldIndex)
				}
			case 1: // Holder
				if !permissions[amendment.FieldIndex].HolderProposal {
					return fmt.Errorf("Field %d amendment not permitted by holder proposal", amendment.FieldIndex)
				}
			default:
				return fmt.Errorf("Invalid proposal initiator type : %d", proposalInitiator)
			}

			if int(votingSystem) >= len(permissions[amendment.FieldIndex].VotingSystemsAllowed) {
				return fmt.Errorf("Field %d amendment voting system out of range : %d", amendment.FieldIndex, votingSystem)
			}
			if !permissions[amendment.FieldIndex].VotingSystemsAllowed[votingSystem] {
				return fmt.Errorf("Field %d amendment not allowed using voting system %d", amendment.FieldIndex, votingSystem)
			}
		} else if !permissions[amendment.FieldIndex].Permitted {
			return fmt.Errorf("Field %d amendment not permitted without proposal", amendment.FieldIndex)
		}
	}

	return nil
}

// applyContractAmendments applies the amendments to the contract formation.
func applyContractAmendments(cf *protocol.ContractFormation, amendments []protocol.Amendment) error {
	authFieldsUpdated := false
	for _, amendment := range amendments {
		switch amendment.FieldIndex {
		case 0: // ContractName
			cf.ContractName = string(amendment.Data)

		case 1: // BodyOfAgreementType
			if len(amendment.Data) != 1 {
				return fmt.Errorf("BodyOfAgreementType amendment value is wrong size : %d", len(amendment.Data))
			}
			cf.BodyOfAgreementType = uint8(amendment.Data[0])

		case 2: // BodyOfAgreement
			cf.BodyOfAgreement = amendment.Data

		case 3: // ContractType
			cf.ContractType = string(amendment.Data)

		case 4: // SupportingDocsFileType
			if len(amendment.Data) != 1 {
				return fmt.Errorf("SupportingDocsFileType amendment value is wrong size : %d", len(amendment.Data))
			}
			cf.SupportingDocsFileType = uint8(amendment.Data[0])

		case 5: // SupportingDocs
			cf.SupportingDocs = amendment.Data

		case 6: // GoverningLaw
			cf.GoverningLaw = string(amendment.Data)

		case 7: // Jurisdiction
			cf.Jurisdiction = string(amendment.Data)

		case 8: // ContractExpiration
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractExpiration amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := cf.ContractExpiration.Write(buf); err != nil {
				return fmt.Errorf("ContractExpiration amendment value failed to deserialize : %s", err)
			}

		case 9: // ContractURI
			cf.ContractURI = string(amendment.Data)

		case 10: // Issuer
			switch amendment.SubfieldIndex {
			case 0: // Name
				cf.Issuer.Name = string(amendment.Data)

			case 1: // Type
				if len(amendment.Data) != 1 {
					return fmt.Errorf("Issuer.Type amendment value is wrong size : %d", len(amendment.Data))
				}
				cf.Issuer.Type = amendment.Data[0]

			case 2: // LEI
				cf.Issuer.LEI = string(amendment.Data)

			case 3: // AddressIncluded
				return fmt.Errorf("Amendment attempting to change Issuer.AddressIncluded")

			case 4: // UnitNumber
				cf.Issuer.UnitNumber = string(amendment.Data)

			case 5: // BuildingNumber
				cf.Issuer.BuildingNumber = string(amendment.Data)

			case 6: // Street
				cf.Issuer.Street = string(amendment.Data)

			case 7: // SuburbCity
				cf.Issuer.SuburbCity = string(amendment.Data)

			case 8: // TerritoryStateProvinceCode
				cf.Issuer.TerritoryStateProvinceCode = string(amendment.Data)

			case 9: // CountryCode
				cf.Issuer.CountryCode = string(amendment.Data)

			case 10: // PostalZIPCode
				cf.Issuer.PostalZIPCode = string(amendment.Data)

			case 11: // EmailAddress
				cf.Issuer.EmailAddress = string(amendment.Data)

			case 12: // PhoneNumber
				cf.Issuer.PhoneNumber = string(amendment.Data)

			case 13: // Administration []Administrator
				switch amendment.Operation {
				case 0: // Modify
					if int(amendment.SubfieldElement) >= len(cf.Issuer.Administration) {
						return fmt.Errorf("Contract amendment subfield element out of range for Issuer.Administration : %d",
							amendment.SubfieldElement)
					}

					buf := bytes.NewBuffer(amendment.Data)
					if err := cf.Issuer.Administration[amendment.SubfieldElement].Write(buf); err != nil {
						return fmt.Errorf("Contract amendment Issuer.Administration[%d] failed to deserialize : %s",
							amendment.SubfieldElement, err)
					}

				case 1: // Add element
					buf := bytes.NewBuffer(amendment.Data)
					newAdministrator := protocol.Administrator{}
					if err := newAdministrator.Write(buf); err != nil {
						return fmt.Errorf("Contract amendment addition to Issuer.Administration failed to deserialize : %s",
							err)
					}
					cf.Issuer.Administration = append(cf.Issuer.Administration, newAdministrator)

				case 2: // Delete element
					if int(amendment.SubfieldElement) >= len(cf.Issuer.Administration) {
						return fmt.Errorf("Contract amendment subfield element out of range for Issuer.Administration : %d",
							amendment.SubfieldElement)
					}
					cf.Issuer.Administration = append(cf.Issuer.Administration[:amendment.SubfieldElement],
						cf.Issuer.Administration[amendment.SubfieldElement+1:]...)

				default:
					return fmt.Errorf("Invalid contract amendment operation for Issuer.Administration : %d", amendment.Operation)
				}

			case 14: // Management []Manager
				switch amendment.Operation {
				case 0: // Modify
					if int(amendment.SubfieldElement) >= len(cf.Issuer.Management) {
						return fmt.Errorf("Contract amendment subfield element out of range for Issuer.Management : %d",
							amendment.SubfieldElement)
					}

					buf := bytes.NewBuffer(amendment.Data)
					if err := cf.Issuer.Management[amendment.SubfieldElement].Write(buf); err != nil {
						return fmt.Errorf("Contract amendment Issuer.Management[%d] failed to deserialize : %s",
							amendment.SubfieldElement, err)
					}

				case 1: // Add element
					buf := bytes.NewBuffer(amendment.Data)
					newManager := protocol.Manager{}
					if err := newManager.Write(buf); err != nil {
						return fmt.Errorf("Contract amendment addition to Issuer.Management failed to deserialize : %s",
							err)
					}
					cf.Issuer.Management = append(cf.Issuer.Management, newManager)

				case 2: // Delete element
					if int(amendment.SubfieldElement) >= len(cf.Issuer.Management) {
						return fmt.Errorf("Contract amendment subfield element out of range for Issuer.Management : %d",
							amendment.SubfieldElement)
					}
					cf.Issuer.Management = append(cf.Issuer.Management[:amendment.SubfieldElement],
						cf.Issuer.Management[amendment.SubfieldElement+1:]...)

				default:
					return fmt.Errorf("Invalid contract amendment operation for Issuer.Management : %d", amendment.Operation)
				}

			default:
				return fmt.Errorf("Contract amendment subfield offset for Issuer out of range : %d", amendment.SubfieldIndex)
			}

		case 11: // IssuerLogoURL
			cf.IssuerLogoURL = string(amendment.Data)

		case 12: // ContractOperatorIncluded
			return fmt.Errorf("Amendment attempting to change ContractOperatorIncluded")

		case 13: // ContractOperator
			switch amendment.SubfieldIndex {
			case 0: // Name
				cf.ContractOperator.Name = string(amendment.Data)

			case 1: // Type
				if len(amendment.Data) != 1 {
					return fmt.Errorf("ContractOperator.Type amendment value is wrong size : %d", len(amendment.Data))
				}
				cf.ContractOperator.Type = amendment.Data[0]

			case 2: // LEI
				cf.ContractOperator.LEI = string(amendment.Data)

			case 3: // AddressIncluded
				return fmt.Errorf("Amendment attempting to change ContractOperator.AddressIncluded")

			case 4: // UnitNumber
				cf.ContractOperator.UnitNumber = string(amendment.Data)

			case 5: // BuildingNumber
				cf.ContractOperator.BuildingNumber = string(amendment.Data)

			case 6: // Street
				cf.ContractOperator.Street = string(amendment.Data)

			case 7: // SuburbCity
				cf.ContractOperator.SuburbCity = string(amendment.Data)

			case 8: // TerritoryStateProvinceCode
				cf.ContractOperator.TerritoryStateProvinceCode = string(amendment.Data)

			case 9: // CountryCode
				cf.ContractOperator.CountryCode = string(amendment.Data)

			case 10: // PostalZIPCode
				cf.ContractOperator.PostalZIPCode = string(amendment.Data)

			case 11: // EmailAddress
				cf.ContractOperator.EmailAddress = string(amendment.Data)

			case 12: // PhoneNumber
				cf.ContractOperator.PhoneNumber = string(amendment.Data)

			case 13: // Administration []Administrator
				switch amendment.Operation {
				case 0: // Modify
					if int(amendment.SubfieldElement) >= len(cf.ContractOperator.Administration) {
						return fmt.Errorf("Contract amendment subfield element out of range for ContractOperator.Administration : %d",
							amendment.SubfieldElement)
					}

					buf := bytes.NewBuffer(amendment.Data)
					if err := cf.ContractOperator.Administration[amendment.SubfieldElement].Write(buf); err != nil {
						return fmt.Errorf("Contract amendment ContractOperator.Administration[%d] failed to deserialize : %s",
							amendment.SubfieldElement, err)
					}

				case 1: // Add element
					buf := bytes.NewBuffer(amendment.Data)
					newAdministrator := protocol.Administrator{}
					if err := newAdministrator.Write(buf); err != nil {
						return fmt.Errorf("Contract amendment addition to ContractOperator.Administration failed to deserialize : %s",
							err)
					}
					cf.ContractOperator.Administration = append(cf.ContractOperator.Administration, newAdministrator)

				case 2: // Delete element
					if int(amendment.SubfieldElement) >= len(cf.ContractOperator.Administration) {
						return fmt.Errorf("Contract amendment subfield element out of range for ContractOperator.Administration : %d",
							amendment.SubfieldElement)
					}
					cf.ContractOperator.Administration = append(cf.ContractOperator.Administration[:amendment.SubfieldElement],
						cf.ContractOperator.Administration[amendment.SubfieldElement+1:]...)

				default:
					return fmt.Errorf("Invalid contract amendment operation for ContractOperator.Administration : %d", amendment.Operation)
				}

			case 14: // Management []Manager
				switch amendment.Operation {
				case 0: // Modify
					if int(amendment.SubfieldElement) >= len(cf.ContractOperator.Management) {
						return fmt.Errorf("Contract amendment subfield element out of range for ContractOperator.Management : %d",
							amendment.SubfieldElement)
					}

					buf := bytes.NewBuffer(amendment.Data)
					if err := cf.ContractOperator.Management[amendment.SubfieldElement].Write(buf); err != nil {
						return fmt.Errorf("Contract amendment ContractOperator.Management[%d] failed to deserialize : %s",
							amendment.SubfieldElement, err)
					}

				case 1: // Add element
					buf := bytes.NewBuffer(amendment.Data)
					newManager := protocol.Manager{}
					if err := newManager.Write(buf); err != nil {
						return fmt.Errorf("Contract amendment addition to ContractOperator.Management failed to deserialize : %s",
							err)
					}
					cf.ContractOperator.Management = append(cf.ContractOperator.Management, newManager)

				case 2: // Delete element
					if int(amendment.SubfieldElement) >= len(cf.ContractOperator.Management) {
						return fmt.Errorf("Contract amendment subfield element out of range for ContractOperator.Management : %d",
							amendment.SubfieldElement)
					}
					cf.ContractOperator.Management = append(cf.ContractOperator.Management[:amendment.SubfieldElement],
						cf.ContractOperator.Management[amendment.SubfieldElement+1:]...)

				default:
					return fmt.Errorf("Invalid contract amendment operation for ContractOperator.Management : %d", amendment.Operation)
				}

			default:
				return fmt.Errorf("Contract amendment subfield offset for ContractOperator out of range : %d", amendment.SubfieldIndex)
			}

		case 14: // ContractAuthFlags
			cf.ContractAuthFlags = amendment.Data
			authFieldsUpdated = true

		case 15: // ContractFee
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractFee amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.ContractFee); err != nil {
				return fmt.Errorf("ContractFee amendment value failed to deserialize : %s", err)
			}

		case 16: // VotingSystems
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(cf.VotingSystems) {
					return fmt.Errorf("Contract amendment element out of range for VotingSystems : %d",
						amendment.Element)
				}

				buf := bytes.NewBuffer(amendment.Data)
				if err := cf.VotingSystems[amendment.Element].Write(buf); err != nil {
					return fmt.Errorf("Contract amendment VotingSystems[%d] failed to deserialize : %s",
						amendment.Element, err)
				}

			case 1: // Add element
				buf := bytes.NewBuffer(amendment.Data)
				newVotingSystem := protocol.VotingSystem{}
				if err := newVotingSystem.Write(buf); err != nil {
					return fmt.Errorf("Contract amendment addition to VotingSystems failed to deserialize : %s",
						err)
				}
				cf.VotingSystems = append(cf.VotingSystems, newVotingSystem)

			case 2: // Delete element
				if int(amendment.Element) >= len(cf.VotingSystems) {
					return fmt.Errorf("Contract amendment element out of range for VotingSystems : %d",
						amendment.Element)
				}
				cf.VotingSystems = append(cf.VotingSystems[:amendment.Element],
					cf.VotingSystems[amendment.Element+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for VotingSystems : %d", amendment.Operation)
			}

		case 17: // RestrictedQtyAssets
			if len(amendment.Data) != 8 {
				return fmt.Errorf("RestrictedQtyAssets amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.RestrictedQtyAssets); err != nil {
				return fmt.Errorf("RestrictedQtyAssets amendment value failed to deserialize : %s", err)
			}

		case 18: // IssuerProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("IssuerProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.IssuerProposal); err != nil {
				return fmt.Errorf("IssuerProposal amendment value failed to deserialize : %s", err)
			}

		case 19: // HolderProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case 20: // Registers
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(cf.Registers) {
					return fmt.Errorf("Contract amendment element out of range for Registers : %d",
						amendment.Element)
				}

				buf := bytes.NewBuffer(amendment.Data)
				if err := cf.Registers[amendment.Element].Write(buf); err != nil {
					return fmt.Errorf("Contract amendment Registers[%d] failed to deserialize : %s",
						amendment.Element, err)
				}

			case 1: // Add element
				buf := bytes.NewBuffer(amendment.Data)
				newRegister := protocol.Register{}
				if err := newRegister.Write(buf); err != nil {
					return fmt.Errorf("Contract amendment addition to Registries failed to deserialize : %s",
						err)
				}
				cf.Registers = append(cf.Registers, newRegister)

			case 2: // Delete element
				if int(amendment.Element) >= len(cf.Registers) {
					return fmt.Errorf("Contract amendment element out of range for Registers : %d",
						amendment.Element)
				}
				cf.Registers = append(cf.Registers[:amendment.Element],
					cf.Registers[amendment.Element+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for Registries : %d", amendment.Operation)
			}

		default:
			return fmt.Errorf("Contract amendment field offset out of range : %d", amendment.FieldIndex)
		}
	}

	if authFieldsUpdated {
		if _, err := protocol.ReadAuthFlags(cf.ContractAuthFlags, contract.FieldCount, len(cf.VotingSystems)); err != nil {
			return fmt.Errorf("Invalid contract auth flags : %s", err)
		}
	}

	// Check validity of updated contract data
	if err := cf.Validate(); err != nil {
		return fmt.Errorf("Contract data invalid after amendments : %s", err)
	}

	return nil
}
