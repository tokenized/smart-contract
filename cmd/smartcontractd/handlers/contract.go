package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
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

type Contract struct {
	MasterDB *db.DB
	Config   *node.Config
	Headers  node.BitcoinHeaders
}

// OfferRequest handles an incoming Contract Offer and prepares a Formation response
func (c *Contract) OfferRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Offer")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractOffer)
	if !ok {
		return errors.New("Could not assert as *actions.ContractOffer")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract offer invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	if _, err := contract.Retrieve(ctx, c.MasterDB, rk.Address, c.Config.IsTest); err == nil {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Contract already exists : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExists)
	} else if errors.Cause(err) != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if msg.BodyOfAgreementType == 1 && len(msg.BodyOfAgreement) != 32 {
		node.LogWarn(ctx, "Contract body of agreement hash is incorrect length : %d",
			len(msg.BodyOfAgreement))
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
			fmt.Sprintf("Contract body of agreement hash is incorrect length : %d",
				len(msg.BodyOfAgreement)))
	}

	if msg.ContractExpiration != 0 && msg.ContractExpiration < v.Now.Nano() {
		node.LogWarn(ctx, "Expiration already passed : %d", msg.ContractExpiration)
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
			fmt.Sprintf("Expiration already passed : %d", msg.ContractExpiration))
	}

	// Verify entity contract
	if len(msg.EntityContract) > 0 {
		if _, err := bitcoin.DecodeRawAddress(msg.EntityContract); err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Entity contract address invalid : %s", err))
		}
	}

	// Verify operator entity contract
	if len(msg.OperatorEntityContract) > 0 {
		ra, err := bitcoin.DecodeRawAddress(msg.OperatorEntityContract)
		if err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Operator entity contract address invalid : %s", err))
		}

		entityCF, err := contract.FetchContractFormation(ctx, c.MasterDB, ra, c.Config.IsTest)
		if err != nil {
			if errors.Cause(err) == contract.ErrNotFound {
				return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
					"Operator entity contract not found")
			}
			return errors.Wrap(err, "fetch operator entity contract formation")
		}
		logger.Info(ctx, "Found Operator Entity Contract : %s", entityCF.ContractName)

		// Check service type
		found := false
		for _, service := range entityCF.Services {
			if service.Type == actions.ServiceTypeContractOperator {
				if len(itx.Inputs) < 2 {
					return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
						"Contract operator input missing")
				}

				servicePublicKey, err := bitcoin.PublicKeyFromBytes(service.PublicKey)
				if err != nil {
					return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
						fmt.Sprintf("Contract operator public key invalid : %s", err))
				}

				serviceAddress, err := servicePublicKey.RawAddress()
				if err != nil {
					return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
						fmt.Sprintf("Contract operator public key not addressable : %s", err))
				}

				// Check that second input is from contract operator service key
				if !itx.Inputs[1].Address.Equal(serviceAddress) {
					return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
						"Contract operator input from wrong address")
				}

				found = true
				break
			}
		}

		if !found {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				"Contract operator service type not found for contract")
		}
	}

	if len(msg.MasterAddress) > 0 {
		if _, err := bitcoin.DecodeRawAddress(msg.MasterAddress); err != nil {
			node.LogWarn(ctx, "Invalid master address : %s", err)
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				"invalid master address")
		}
	}

	if len(msg.MasterAddress) > 0 {
		if _, err := bitcoin.DecodeRawAddress(msg.MasterAddress); err != nil {
			node.LogWarn(ctx, "Invalid master address : %s", err)
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				"invalid master address")
		}
	}

	if _, err := permissions.PermissionsFromBytes(msg.ContractPermissions,
		len(msg.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid contract permissions : %s", err)
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
			fmt.Sprintf("Invalid contract permissions : %s", err))
	}

	// Validate voting systems are all valid.
	for _, votingSystem := range msg.VotingSystems {
		if err := vote.ValidateVotingSystem(votingSystem); err != nil {
			node.LogWarn(ctx, "Invalid voting system : %s", err)
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Invalid voting system : %s", err))
		}
	}

	// Check any oracle entity contracts
	for _, oracle := range msg.Oracles {
		ra, err := bitcoin.DecodeRawAddress(oracle.EntityContract)
		if err != nil {
			node.LogWarn(ctx, "Invalid oracle entity address : %s", err)
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Invalid oracle entity address : %s", err))
		}
		oracleCF, err := contract.FetchContractFormation(ctx, c.MasterDB, ra, c.Config.IsTest)
		if err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Oracle entity address : %s", err))
		}
		logger.Info(ctx, "Found Oracle Contract : %s", oracleCF.ContractName)

		// Check oracle type
		for _, ot := range oracle.OracleTypes {
			found := false
			for _, service := range oracleCF.Services {
				if service.Type == ot {
					found = true
					break
				}
			}

			if !found {
				return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
					"Oracle type not found for contract")
			}
		}
	}

	node.Log(ctx, "Accepting contract offer : %s", msg.ContractName)

	// Contract Formation <- Contract Offer
	cf := &actions.ContractFormation{}
	if err := node.Convert(ctx, &msg, cf); err != nil {
		return err
	}

	cf.AdminAddress = itx.Inputs[0].Address.Bytes()
	if msg.ContractOperatorIncluded {
		if len(itx.Inputs) < 2 {
			node.LogWarn(ctx, "Missing operator input")
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				"missing operator input")
		}
		cf.OperatorAddress = itx.Inputs[1].Address.Bytes()
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
		return errors.Wrap(err, "save tx")
	}

	// Respond with a formation
	if err := node.RespondSuccess(ctx, w, itx, rk, cf); err != nil {
		return errors.Wrap(err, "respond success")
	}

	return contract.SaveContractFormation(ctx, c.MasterDB, rk.Address, cf, c.Config.IsTest)
}

// AmendmentRequest handles an incoming Contract Amendment and prepares a Formation response
func (c *Contract) AmendmentRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	node.Log(ctx, "Amendment Tx : %s", itx.Hash)

	ctx, span := trace.StartSpan(ctx, "handlers.Contract.Amendment")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractAmendment)
	if !ok {
		return errors.New("Could not assert as *protocol.ContractAmendment")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract amendment invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address, c.Config.IsTest)
	if err != nil {
		if errors.Cause(err) == contract.ErrNotFound {
			address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
			node.LogWarn(ctx, "Contract doesn't exist : %s", address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractDoesNotExist)
		}
		return errors.Wrap(err, "Failed to retrieve contract")
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

	if ct.Revision != msg.ContractRevision {
		node.LogWarn(ctx, "Incorrect contract revision : specified %d != current %d",
			msg.ContractRevision, ct.Revision)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractRevision)
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
		voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
		if err != nil {
			node.LogWarn(ctx, "Invalid vote txid : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		vt, err := vote.Retrieve(ctx, c.MasterDB, rk.Address, voteTxId)
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

	// Contract Formation <- Contract Amendment
	cf, err := contract.FetchContractFormation(ctx, c.MasterDB, rk.Address, c.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "fetch contract formation")
	}

	// Ensure reduction in qty is OK, keeping in mind that zero (0) means
	// unlimited asset creation is permitted.
	if cf.RestrictedQtyAssets > 0 && cf.RestrictedQtyAssets < uint64(len(ct.AssetCodes)) {
		node.LogWarn(ctx, "Cannot reduce allowable assets below existing number")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractAssetQtyReduction)
	}

	if msg.ChangeAdministrationAddress || msg.ChangeOperatorAddress {
		if !ct.OperatorAddress.IsEmpty() {
			if len(itx.Inputs) < 2 {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk,
					actions.RejectionsContractBothOperatorsRequired)
			}

			if itx.Inputs[0].Address.Equal(itx.Inputs[1].Address) ||
				!contract.IsOperator(ctx, ct, itx.Inputs[0].Address) ||
				!contract.IsOperator(ctx, ct, itx.Inputs[1].Address) {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk,
					actions.RejectionsContractBothOperatorsRequired)
			}
		} else {
			if len(itx.Inputs) < 1 {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk,
					actions.RejectionsContractBothOperatorsRequired)
			}

			if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
				node.Log(ctx, "All operators required for operator change")
				return node.RespondReject(ctx, w, itx, rk,
					actions.RejectionsContractBothOperatorsRequired)
			}
		}
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

		cf.AdminAddress = itx.Inputs[inputIndex].Address.Bytes()
		inputIndex++
	}

	// Operator changes. New operator in second input unless there is also a new administration,
	// then it is in the third input
	if msg.ChangeOperatorAddress {
		if len(itx.Inputs) <= inputIndex {
			return errors.New("New operator specified but not included in inputs")
		}

		cf.OperatorAddress = itx.Inputs[inputIndex].Address.Bytes()
	}

	if err := applyContractAmendments(cf, msg.Amendments, proposed, proposalType,
		votingSystem); err != nil {
		node.LogWarn(ctx, "Failed to apply amendments : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Verify entity contract
	if len(cf.EntityContract) > 0 {
		if _, err := bitcoin.DecodeRawAddress(cf.EntityContract); err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Entity contract address invalid : %s", err))
		}
	}

	// Verify operator entity contract
	if len(cf.OperatorEntityContract) > 0 {
		ra, err := bitcoin.DecodeRawAddress(cf.OperatorEntityContract)
		if err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Operator entity contract address invalid : %s", err))
		}

		entityCF, err := contract.FetchContractFormation(ctx, c.MasterDB, ra, c.Config.IsTest)
		if err != nil {
			if errors.Cause(err) == contract.ErrNotFound {
				return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
					"Operator entity contract not found")
			}
			return errors.Wrap(err, "fetch operator entity contract formation")
		}
		logger.Info(ctx, "Found Operator Entity Contract : %s", entityCF.ContractName)
	}

	// Check admin identity oracle signatures
	for _, adminCert := range cf.AdminIdentityCertificates {
		if err := validateContractAmendOracleSig(ctx, c.MasterDB, cf, adminCert, c.Headers,
			c.Config.IsTest); err != nil {
			node.LogVerbose(ctx, "New admin identity signature invalid : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInvalidSignature)
		}
	}

	// Check any oracle entity contracts
	for _, oracle := range cf.Oracles {
		ra, err := bitcoin.DecodeRawAddress(oracle.EntityContract)
		if err != nil {
			node.LogWarn(ctx, "Invalid oracle entity address : %s", err)
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Invalid oracle entity address : %s", err))
		}
		if _, err := contract.FetchContractFormation(ctx, c.MasterDB, ra, c.Config.IsTest); err != nil {
			return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed,
				fmt.Sprintf("Oracle entity address : %s", err))
		}
	}

	// Apply modifications
	cf.ContractRevision = ct.Revision + 1 // Bump the revision
	cf.Timestamp = v.Now.Nano()

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx.
	if err := transactions.AddTx(ctx, c.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	node.Log(ctx, "Accepting contract amendment")

	// Respond with a formation
	if err := node.RespondSuccess(ctx, w, itx, rk, cf); err == nil {
		return contract.SaveContractFormation(ctx, c.MasterDB, rk.Address, cf, c.Config.IsTest)
	}
	return err
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

	// Locate Contract. Sender is verified to be contract before this response function is called.
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		return fmt.Errorf("Contract formation not from contract : %s", address)
	}

	contractName := msg.ContractName
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address, c.Config.IsTest)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct != nil && !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address)
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
				return errors.Wrap(err, "Failed to convert bitcoin.Hash32 to bitcoin.Hash32")
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
			voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
			if err != nil {
				return errors.Wrap(err, "invalid vote txid")
			}

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
			node.LogWarn(ctx, "Failed to convert formation to new contract (%s) : %s",
				contractName, err.Error())
			return err
		}

		// Get contract offer message to retrieve administration and operator.
		var offerTx *inspector.Transaction
		offerTx, err = transactions.GetTx(ctx, c.MasterDB, &itx.Inputs[0].UTXO.Hash,
			c.Config.IsTest)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Contract Offer tx not found : %s",
				itx.Inputs[0].UTXO.Hash.String()))
		}

		// Get offer from it
		offer, ok := offerTx.MsgProto.(*actions.ContractOffer)
		if !ok {
			return fmt.Errorf("Could not find Contract Offer in offer tx")
		}

		nc.AdminAddress = offerTx.Inputs[0].Address // First input of offer tx
		if offer.ContractOperatorIncluded && len(offerTx.Inputs) > 1 {
			nc.OperatorAddress = offerTx.Inputs[1].Address // Second input of offer tx
		}

		if err := contract.Create(ctx, c.MasterDB, rk.Address, &nc, c.Config.IsTest, v.Now); err != nil {
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

			uc.AdminAddress = &request.Inputs[inputIndex].Address
			inputIndex++
			address := bitcoin.NewAddressFromRawAddress(*uc.AdminAddress, w.Config.Net)
			node.Log(ctx, "Updating contract administration address : %s", address.String())
		}

		// Operator changes. New operator in second input unless there is also a new administration,
		// then it is in the third input
		if amendment != nil && amendment.ChangeOperatorAddress {
			if len(request.Inputs) <= inputIndex {
				return errors.New("New operator specified but not included in inputs")
			}

			uc.OperatorAddress = &request.Inputs[inputIndex].Address
			address := bitcoin.NewAddressFromRawAddress(*uc.OperatorAddress, w.Config.Net)
			node.Log(ctx, "Updating contract operator PKH : %s", address.String())
		}

		if ct.ContractType != msg.ContractType {
			uc.ContractType = &msg.ContractType
			node.Log(ctx, "Updating contract type : %d", *uc.ContractType)
		}

		if ct.ContractFee != msg.ContractFee {
			uc.ContractFee = &msg.ContractFee
			node.Log(ctx, "Updating contract fee : %d", *uc.ContractFee)
		}

		if ct.ContractExpiration.Nano() != msg.ContractExpiration {
			ts := protocol.NewTimestamp(msg.ContractExpiration)
			uc.ContractExpiration = &ts
			newExpiration := time.Unix(int64(msg.ContractExpiration), 0)
			node.Log(ctx, "Updating contract expiration : %s", newExpiration.Format(time.UnixDate))
		}

		if ct.RestrictedQtyAssets != msg.RestrictedQtyAssets {
			uc.RestrictedQtyAssets = &msg.RestrictedQtyAssets
			node.Log(ctx, "Updating contract restricted quantity assets : %d",
				*uc.RestrictedQtyAssets)
		}

		if ct.AdministrationProposal != msg.AdministrationProposal {
			uc.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating contract administration proposal : %t",
				*uc.AdministrationProposal)
		}

		if ct.HolderProposal != msg.HolderProposal {
			uc.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating contract holder proposal : %t", *uc.HolderProposal)
		}

		if ct.BodyOfAgreementType != msg.BodyOfAgreementType {
			uc.BodyOfAgreementType = &msg.BodyOfAgreementType
			node.Log(ctx, "Updating contract body of agreement type : %d", *uc.BodyOfAgreementType)
		}

		// Check if oracles are different
		different := len(ct.Oracles) != len(msg.Oracles)
		if !different {
			for i, oracle := range ct.Oracles {
				if !oracle.Equal(msg.Oracles[i]) {
					different = true
					break
				}
			}
		}

		if different {
			node.Log(ctx, "Updating contract oracles")
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
			node.Log(ctx, "Updating contract voting systems")
			uc.VotingSystems = &msg.VotingSystems
		}

		if err := contract.Update(ctx, c.MasterDB, rk.Address, &uc, c.Config.IsTest,
			v.Now); err != nil {
			return errors.Wrap(err, "Failed to update contract")
		}
		node.Log(ctx, "Updated contract")

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId.String())
			if err := vote.MarkApplied(ctx, c.MasterDB, rk.Address, vt.VoteTxId, request.Hash,
				v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}
	}

	return nil
}

// AddressChange handles an incoming Contract Address Change.
func (c *Contract) AddressChange(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Contract.AddressChange")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.ContractAddressChange)
	if !ok {
		return errors.New("Could not assert as *actions.ContractAddressChange")
	}

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Contract address change invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, c.MasterDB, rk.Address, c.Config.IsTest)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Check that it is from the master PKH
	if !itx.Inputs[0].Address.Equal(ct.MasterAddress) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogWarn(ctx, "Contract address change must be from master address : %s",
			address.String())
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
	err = contract.Move(ctx, c.MasterDB, rk.Address, newContractAddress, c.Config.IsTest,
		protocol.NewTimestamp(msg.Timestamp))
	if err != nil {
		return err
	}

	//TODO Transfer all UTXOs to fee address.

	return nil
}

// applyContractAmendments applies the amendments to the contract formation.
func applyContractAmendments(cf *actions.ContractFormation, amendments []*actions.AmendmentField,
	proposed bool, proposalType, votingSystem uint32) error {

	perms, err := permissions.PermissionsFromBytes(cf.ContractPermissions,
		len(cf.VotingSystems))
	if err != nil {
		return fmt.Errorf("Invalid contract permissions : %s", err)
	}

	for i, amendment := range amendments {
		fip, err := permissions.FieldIndexPathFromBytes(amendment.FieldIndexPath)
		if err != nil {
			return fmt.Errorf("Failed to read amendment %d field index path : %s", i, err)
		}
		if len(fip) == 0 {
			return fmt.Errorf("Amendment %d has no field specified", i)
		}

		switch fip[0] {
		case actions.ContractFieldContractPermissions:
			if _, err := permissions.PermissionsFromBytes(amendment.Data,
				len(cf.VotingSystems)); err != nil {
				return fmt.Errorf("ContractPermissions amendment value is invalid : %s", err)
			}
		}

		fieldPermissions, err := cf.ApplyAmendment(fip, amendment.Operation, amendment.Data, perms)
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
	if err := cf.Validate(); err != nil {
		return fmt.Errorf("Contract data invalid after amendments : %s", err)
	}

	return nil
}

func validateContractAmendOracleSig(ctx context.Context, dbConn *db.DB,
	cf *actions.ContractFormation, adminCert *actions.AdminIdentityCertificateField,
	headers node.BitcoinHeaders, isTest bool) error {

	oracleAddress, err := bitcoin.DecodeRawAddress(adminCert.EntityContract)
	if err != nil {
		return errors.Wrap(err, "entity address")
	}

	oracleContract, err := contract.FetchContractFormation(ctx, dbConn, oracleAddress, isTest)
	if err != nil {
		return errors.Wrap(err, "fetch oracle")
	}

	oracle, err := contract.GetIdentityOracleKey(oracleContract)
	if err != nil {
		return errors.Wrap(err, "get identity oracle")
	}

	// Parse signature
	oracleSig, err := bitcoin.SignatureFromBytes(adminCert.Signature)
	if err != nil {
		return errors.Wrap(err, "Failed to parse oracle signature")
	}

	hash, err := headers.BlockHash(ctx, int(adminCert.BlockHeight))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to retrieve hash for block height %d",
			adminCert.BlockHeight))
	}

	adminAddress, err := bitcoin.DecodeRawAddress(cf.AdminAddress)
	if err != nil {
		return errors.Wrap(err, "admin address")
	}

	var entity interface{}
	if len(cf.EntityContract) > 0 {
		// Use parent entity contract address in signature instead of entity structure.
		entityRA, err := bitcoin.DecodeRawAddress(cf.EntityContract)
		if err != nil {
			return errors.Wrap(err, "entity address")
		}

		entity = entityRA
	} else {
		entity = cf.Issuer
	}

	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, adminAddress, entity, *hash,
		adminCert.Expiration, 1)
	if err != nil {
		return err
	}

	if oracleSig.Verify(*sigHash, oracle) {
		return nil // Valid signature found
	}

	return fmt.Errorf("Contract signature invalid")
}
