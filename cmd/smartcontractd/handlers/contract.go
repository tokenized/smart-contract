package handlers

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"

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
			return errors.Wrap(err, "Failed to retreive contract")
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
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
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
		return errors.Wrap(err, "Failed to retreive contract")
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

	// TODO Validate that changes are allowed. Check votes, ...
	// TODO Verify RefTxID data
	//msg.RefTxID               []byte

	logger.Info(ctx, "%s : Accepting contract amendment (%s) : %s", v.TraceID, ct.ContractName, contractPKH.String())

	// Contract Formation <- Contract Amendment
	cf := protocol.ContractFormation{}

	// Get current state
	err = node.Convert(ctx, &ct, &cf)
	if err != nil {
		return err
	}

	// Apply modifications
	cf.ContractRevision = ct.Revision + 1 // Bump the revision
	cf.Timestamp = v.Now

	// TODO Implement contract amendments
	// type Amendment struct {
	// FieldIndex    uint8
	// Element       uint16
	// SubfieldIndex uint8
	// Operation     uint8
	// Data          []byte
	// }
	// for _, amendment := range msg.Amendments {
	// switch(amendment.FieldIndex) {
	// default:
	// logger.Warn(ctx, "%s : Incorrect contract amendment field offset (%s) : %d", v.TraceID, ct.ContractName, amendment.FieldIndex)
	// return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	// }
	// }

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &c.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
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

	// TODO Fetch and remove request tx
	// hash, err = chainhash.NewHash(msg.VoteTxId.Bytes())
	// c.TxCache.RemoveTx(ctx, hash)

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
		// TODO Pull from ammendment tx.
		// // TODO Issuer change. New issuer in second input
		// if msg.ChangeIssuerAddress {
		// if len(itx.Inputs) < 2 {
		// return errors.New("New issuer specified but not included in inputs")
		// }

		// cf.IssuerPKH = protocol.PublicKeyHashFromBytes(itx.Inputs[1].Address.ScriptAddress)
		// }

		// // TODO Operator changes. New operator in second input unless there is also a new issuer, then it is in the third input
		// if msg.ChangeOperatorAddress {
		// index := 1
		// if msg.ChangeIssuerAddress {
		// index++
		// }
		// if index >= len(itx.Inputs) {
		// return errors.New("New operator specified but not included in inputs")
		// }

		// cf.OperatorPKH = protocol.PublicKeyHashFromBytes(itx.Inputs[index].Address.ScriptAddress)
		// }

		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		uc := contract.UpdateContract{
			Revision:  &msg.ContractRevision,
			Timestamp: &msg.Timestamp,
		}

		if !bytes.Equal(ct.IssuerPKH.Bytes(), itx.Outputs[1].Address.ScriptAddress()) { // Second output of formation tx
			// TODO Should asset balances be moved from previous issuer to new issuer.
			uc.IssuerPKH = protocol.PublicKeyHashFromBytes(itx.Outputs[1].Address.ScriptAddress())
			logger.Info(ctx, "%s : Updating contract issuer address (%s) : %s", v.TraceID, ct.ContractName, itx.Outputs[1].Address.String())
		}

		// TODO Update operator address - OperatorAddress *string

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
		different := len(ct.Registries) != len(msg.Registries)
		if !different {
			for i, registry := range ct.Registries {
				if !registry.Equal(msg.Registries[i]) {
					different = true
					break
				}
			}
		}

		if different {
			logger.Info(ctx, "%s : Updating contract registries (%s)", v.TraceID, ct.ContractName)
			uc.Registries = &msg.Registries
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

		// TODO Mark vote as "applied" if this amendment was a result of a vote.
		// Pull
	}

	return nil
}
