package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
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
		node.LogWarn(ctx, "Contract offer request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	_, err := contract.Retrieve(ctx, c.MasterDB, rk.Address)
	if err != contract.ErrNotFound {
		if err == nil {
			address := bitcoin.NewAddressFromRawAddress(rk.Address, wire.BitcoinNet(w.Config.ChainParams.Net))
			node.LogWarn(ctx, "Contract already exists : %s", address.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectContractExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve contract")
		}
	}

	if msg.BodyOfAgreementType == 1 && len(msg.BodyOfAgreement) != 32 {
		node.LogWarn(ctx, "Contract body of agreement hash is incorrect length : %d", len(msg.BodyOfAgreement))
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if msg.ContractExpiration != 0 && msg.ContractExpiration < v.Now.Nano() {
		node.LogWarn(ctx, "Expiration already passed : %d", msg.ContractExpiration)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if _, err = protocol.ReadAuthFlags(msg.ContractAuthFlags, contract.FieldCount, len(msg.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid contract auth flags : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	// Validate voting systems are all valid.
	for _, votingSystem := range msg.VotingSystems {
		if err = vote.ValidateVotingSystem(votingSystem); err != nil {
			node.LogWarn(ctx, "Invalid voting system : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
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

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractMoved)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogVerbose(ctx, "Requestor is not operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectNotOperator)
	}

	if ct.Revision != msg.ContractRevision {
		node.LogWarn(ctx, "Incorrect contract revision (%s) : specified %d != current %d",
			ct.ContractName, msg.ContractRevision, ct.Revision)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalInitiator := uint32(0)
	votingSystem := uint32(0)

	if len(msg.RefTxID) != 0 { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := chainhash.NewHash(msg.RefTxID)
		if err != nil {
			return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
		}

		// Retrieve Vote Result
		voteResultTx, err := transactions.GetTx(ctx, c.MasterDB, refTxId, c.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Vote Result tx not found for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
		if !ok {
			node.LogWarn(ctx, "Vote Result invalid for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		// Retrieve the vote
		voteTxId := protocol.TxIdFromBytes(voteResult.VoteTxId)
		vt, err := vote.Retrieve(ctx, c.MasterDB, rk.Address, voteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %s", voteTxId.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteTxId.String(), err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			node.LogWarn(ctx, "Vote not complete yet")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		if vt.Result != "A" {
			node.LogWarn(ctx, "Vote result not A(Accept) : %s", vt.Result)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		if !vt.Specific {
			node.LogWarn(ctx, "Vote was not for specific amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		if vt.AssetSpecificVote {
			node.LogWarn(ctx, "Vote was not for contract amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if ct.RestrictedQtyAssets > 0 && ct.RestrictedQtyAssets < uint64(len(ct.AssetCodes)) {
		node.LogWarn(ctx, "Cannot reduce allowable assets below existing number")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractAssetQtyReduction)
	}

	if msg.ChangeAdministrationAddress || msg.ChangeOperatorAddress {
		if len(itx.Inputs) < 2 {
			node.LogVerbose(ctx, "Both operators required for operator change")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectContractBothOperatorsRequired)
		}

		if itx.Inputs[0].Address.Equal(itx.Inputs[1].Address) ||
			!contract.IsOperator(ctx, ct, itx.Inputs[0].Address) ||
			!contract.IsOperator(ctx, ct, itx.Inputs[1].Address) {
			node.LogVerbose(ctx, "Both operators required for operator change")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectContractBothOperatorsRequired)
		}
	}

	if err := checkContractAmendmentsPermissions(ct, msg.Amendments, proposed, proposalInitiator, votingSystem); err != nil {
		node.LogWarn(ctx, "Contract amendments not permitted : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractAuthFlags)
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

	if err := applyContractAmendments(&cf, msg.Amendments); err != nil {
		node.LogWarn(ctx, "Contract amendments failed : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Administration change. New administration in second input
	if msg.ChangeAdministrationAddress {
		if len(itx.Inputs) < 2 {
			node.LogWarn(ctx, "New administration specified but not included in inputs (%s)", ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectTxMalformed)
		}
	}

	// Operator changes. New operator in second input unless there is also a new administration, then it is in the third input
	if msg.ChangeOperatorAddress {
		index := 1
		if msg.ChangeAdministrationAddress {
			index++
		}
		if index >= len(itx.Inputs) {
			node.LogWarn(ctx, "New operator specified but not included in inputs (%s)", ct.ContractName)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectTxMalformed)
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

	if ct != nil && ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, wire.BitcoinNet(w.Config.ChainParams.Net))
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
			refTxId, err := chainhash.NewHash(amendment.RefTxID)
			if err != nil {
				return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
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

		nc.AdministrationAddress = bitcoin.NewJSONRawAddress(offerTx.Inputs[0].Address) // First input of offer tx
		if offer.ContractOperatorIncluded && len(offerTx.Inputs) > 1 {
			nc.OperatorAddress = bitcoin.NewJSONRawAddress(offerTx.Inputs[1].Address) // Second input of offer tx
		}
		fmt.Printf("New Contract Admin : %x\n", nc.AdministrationAddress.Bytes())

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
		// Administration change. New administration in second input
		if amendment != nil && amendment.ChangeAdministrationAddress {
			if len(request.Inputs) < 2 {
				return errors.New("New administration specified but not included in inputs")
			}

			uc.AdministrationAddress = bitcoin.NewJSONRawAddress(request.Inputs[1].Address)
			address := bitcoin.NewAddressFromRawAddress(uc.AdministrationAddress,
				wire.BitcoinNet(w.Config.ChainParams.Net))
			node.Log(ctx, "Updating contract administration address : %s", address.String())
		}

		// Operator changes. New operator in second input unless there is also a new administration, then it is in the third input
		if amendment != nil && amendment.ChangeOperatorAddress {
			index := 1
			if amendment.ChangeAdministrationAddress {
				index++
			}
			if index >= len(request.Inputs) {
				return errors.New("New operator specified but not included in inputs")
			}

			uc.OperatorAddress = bitcoin.NewJSONRawAddress(request.Inputs[index].Address)
			address := bitcoin.NewAddressFromRawAddress(uc.OperatorAddress,
				wire.BitcoinNet(w.Config.ChainParams.Net))
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

		if !bytes.Equal(ct.ContractAuthFlags[:], msg.ContractAuthFlags[:]) {
			uc.ContractAuthFlags = &msg.ContractAuthFlags
			node.Log(ctx, "Updating contract auth flags (%s) : %v", ct.ContractName, *uc.ContractAuthFlags)
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
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogWarn(ctx, "Contract address change must be from master address : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectTxMalformed)
	}

	newContractAddress, err := bitcoin.DecodeRawAddress(msg.NewContractAddress)
	if err != nil {
		node.LogWarn(ctx, "Invalid new contract address : %x", msg.NewContractAddress)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectTxMalformed)
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
		return node.RespondReject(ctx, w, itx, rk, actions.RejectTxMalformed)
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

// checkContractAmendmentsPermissions verifies that the amendments are permitted bases on the auth flags.
func checkContractAmendmentsPermissions(ct *state.Contract, amendments []*actions.AmendmentField, proposed bool,
	proposalInitiator, votingSystem uint32) error {

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
			case 0: // Administration
				if !permissions[amendment.FieldIndex].AdministrationProposal {
					return fmt.Errorf("Field %d amendment not permitted by administration proposal", amendment.FieldIndex)
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
func applyContractAmendments(cf *actions.ContractFormation, amendments []*actions.AmendmentField) error {
	authFieldsUpdated := false
	for _, amendment := range amendments {
		switch amendment.FieldIndex {
		case 0: // ContractName
			cf.ContractName = string(amendment.Data)

		case 1: // BodyOfAgreementType
			if len(amendment.Data) != 1 {
				return fmt.Errorf("BodyOfAgreementType amendment value is wrong size : %d", len(amendment.Data))
			}
			cf.BodyOfAgreementType = uint32(amendment.Data[0])

		case 2: // BodyOfAgreement
			cf.BodyOfAgreement = amendment.Data

		case 3: // ContractType
			cf.ContractType = string(amendment.Data)

		case 4: // SupportingDocs
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(cf.SupportingDocs) {
					return fmt.Errorf("Contract amendment element out of range for SupportingDocs : %d",
						amendment.Element)
				}

				cf.SupportingDocs[amendment.Element].Reset()
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, cf.SupportingDocs[amendment.Element]); err != nil {
						return fmt.Errorf("Contract amendment SupportingDocs[%d] failed to deserialize : %s",
							amendment.Element, err)
					}
				}

			case 1: // Add element
				newDocument := actions.DocumentField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newDocument); err != nil {
						return fmt.Errorf("Contract amendment addition to SupportingDocs failed to deserialize : %s",
							err)
					}
				}
				cf.SupportingDocs = append(cf.SupportingDocs, &newDocument)

			case 2: // Delete element
				if int(amendment.Element) >= len(cf.SupportingDocs) {
					return fmt.Errorf("Contract amendment element out of range for SupportingDocs : %d",
						amendment.Element)
				}
				cf.SupportingDocs = append(cf.SupportingDocs[:amendment.Element],
					cf.SupportingDocs[amendment.Element+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for SupportingDocs : %d", amendment.Operation)
			}

		case 5: // GoverningLaw
			cf.GoverningLaw = string(amendment.Data)

		case 6: // Jurisdiction
			cf.Jurisdiction = string(amendment.Data)

		case 7: // ContractExpiration
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractExpiration amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewReader(amendment.Data)
			contractExpiration, err := protocol.DeserializeTimestamp(buf)
			if err != nil {
				return fmt.Errorf("ContractExpiration amendment value failed to deserialize : %s", err)
			}
			cf.ContractExpiration = contractExpiration.Nano()

		case 8: // ContractURI
			cf.ContractURI = string(amendment.Data)

		case 9: // Issuer
			switch amendment.SubfieldIndex {
			case 0: // Name
				cf.Issuer.Name = string(amendment.Data)

			case 1: // Type
				if len(amendment.Data) != 1 {
					return fmt.Errorf("Issuer.Type amendment value is wrong size : %d", len(amendment.Data))
				}
				cf.Issuer.Type = string([]byte{amendment.Data[0]})

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

					cf.Issuer.Administration[amendment.SubfieldElement].Reset()
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, cf.Issuer.Administration[amendment.SubfieldElement]); err != nil {
							return fmt.Errorf("Contract amendment Issuer.Administration[%d] failed to deserialize : %s",
								amendment.SubfieldElement, err)
						}
					}

				case 1: // Add element
					newAdministrator := actions.AdministratorField{}
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, &newAdministrator); err != nil {
							return fmt.Errorf("Contract amendment addition to Issuer.Administration failed to deserialize : %s",
								err)
						}
					}
					cf.Issuer.Administration = append(cf.Issuer.Administration, &newAdministrator)

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

					cf.Issuer.Management[amendment.SubfieldElement].Reset()
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, cf.Issuer.Management[amendment.SubfieldElement]); err != nil {
							return fmt.Errorf("Contract amendment Issuer.Management[%d] failed to deserialize : %s",
								amendment.SubfieldElement, err)
						}
					}

				case 1: // Add element
					newManager := actions.ManagerField{}
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, &newManager); err != nil {
							return fmt.Errorf("Contract amendment addition to Issuer.Management failed to deserialize : %s",
								err)
						}
					}
					cf.Issuer.Management = append(cf.Issuer.Management, &newManager)

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

		case 10: // IssuerLogoURL
			cf.IssuerLogoURL = string(amendment.Data)

		case 11: // ContractOperatorIncluded
			return fmt.Errorf("Amendment attempting to change ContractOperatorIncluded")

		case 12: // ContractOperator
			switch amendment.SubfieldIndex {
			case 0: // Name
				cf.ContractOperator.Name = string(amendment.Data)

			case 1: // Type
				if len(amendment.Data) != 1 {
					return fmt.Errorf("ContractOperator.Type amendment value is wrong size : %d", len(amendment.Data))
				}
				cf.ContractOperator.Type = string([]byte{amendment.Data[0]})

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

					cf.ContractOperator.Administration[amendment.SubfieldElement].Reset()
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, cf.ContractOperator.Administration[amendment.SubfieldElement]); err != nil {
							return fmt.Errorf("Contract amendment ContractOperator.Administration[%d] failed to deserialize : %s",
								amendment.SubfieldElement, err)
						}
					}

				case 1: // Add element
					newAdministrator := actions.AdministratorField{}
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, &newAdministrator); err != nil {
							return fmt.Errorf("Contract amendment addition to ContractOperator.Administration failed to deserialize : %s",
								err)
						}
					}

					cf.ContractOperator.Administration = append(cf.ContractOperator.Administration, &newAdministrator)

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

					cf.ContractOperator.Management[amendment.SubfieldElement].Reset()
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, cf.ContractOperator.Management[amendment.SubfieldElement]); err != nil {
							return fmt.Errorf("Contract amendment ContractOperator.Management[%d] failed to deserialize : %s",
								amendment.SubfieldElement, err)
						}
					}

				case 1: // Add element
					newManager := actions.ManagerField{}
					if len(amendment.Data) != 0 {
						if err := proto.Unmarshal(amendment.Data, &newManager); err != nil {
							return fmt.Errorf("Contract amendment addition to ContractOperator.Management failed to deserialize : %s",
								err)
						}
					}

					cf.ContractOperator.Management = append(cf.ContractOperator.Management, &newManager)

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

		case 13: // ContractAuthFlags
			cf.ContractAuthFlags = amendment.Data
			authFieldsUpdated = true

		case 14: // ContractFee
			if len(amendment.Data) != 8 {
				return fmt.Errorf("ContractFee amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.ContractFee); err != nil {
				return fmt.Errorf("ContractFee amendment value failed to deserialize : %s", err)
			}

		case 15: // VotingSystems
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(cf.VotingSystems) {
					return fmt.Errorf("Contract amendment element out of range for VotingSystems : %d",
						amendment.Element)
				}

				cf.VotingSystems[amendment.Element].Reset()
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, cf.VotingSystems[amendment.Element]); err != nil {
						return fmt.Errorf("Contract amendment VotingSystems[%d] failed to deserialize : %s",
							amendment.Element, err)
					}
				}

			case 1: // Add element
				newVotingSystem := actions.VotingSystemField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newVotingSystem); err != nil {
						return fmt.Errorf("Contract amendment addition to VotingSystems failed to deserialize : %s",
							err)
					}
				}

				cf.VotingSystems = append(cf.VotingSystems, &newVotingSystem)

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

		case 16: // RestrictedQtyAssets
			if len(amendment.Data) != 8 {
				return fmt.Errorf("RestrictedQtyAssets amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.RestrictedQtyAssets); err != nil {
				return fmt.Errorf("RestrictedQtyAssets amendment value failed to deserialize : %s", err)
			}

		case 17: // AdministrationProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AdministrationProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.AdministrationProposal); err != nil {
				return fmt.Errorf("AdministrationProposal amendment value failed to deserialize : %s", err)
			}

		case 18: // HolderProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &cf.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case 19: // Oracles
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(cf.Oracles) {
					return fmt.Errorf("Contract amendment element out of range for Oracles : %d",
						amendment.Element)
				}

				cf.Oracles[amendment.Element].Reset()
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, cf.Oracles[amendment.Element]); err != nil {
						return fmt.Errorf("Contract amendment Oracles[%d] failed to deserialize : %s",
							amendment.Element, err)
					}
				}

			case 1: // Add element
				newOracle := actions.OracleField{}
				if len(amendment.Data) != 0 {
					if err := proto.Unmarshal(amendment.Data, &newOracle); err != nil {
						return fmt.Errorf("Contract amendment addition to Oracles failed to deserialize : %s",
							err)
					}
				}

				cf.Oracles = append(cf.Oracles, &newOracle)

			case 2: // Delete element
				if int(amendment.Element) >= len(cf.Oracles) {
					return fmt.Errorf("Contract amendment element out of range for Oracles : %d",
						amendment.Element)
				}
				cf.Oracles = append(cf.Oracles[:amendment.Element],
					cf.Oracles[amendment.Element+1:]...)

			default:
				return fmt.Errorf("Invalid contract amendment operation for Oracles : %d", amendment.Operation)
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
