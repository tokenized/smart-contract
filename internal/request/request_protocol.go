package request

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func buildContractFormationFromContractOffer(ctx context.Context,
	o protocol.ContractOffer) *protocol.ContractFormation {

	f := protocol.NewContractFormation()
	f.Version = o.Version
	f.ContractName = o.ContractName
	f.ContractFileHash = o.ContractFileHash
	f.GoverningLaw = o.GoverningLaw
	f.Jurisdiction = o.Jurisdiction
	f.ContractExpiration = o.ContractExpiration
	f.URI = o.URI
	f.ContractRevision = 0
	f.IssuerID = o.IssuerID
	f.IssuerType = o.IssuerType
	f.ContractOperatorID = o.ContractOperatorID
	f.AuthorizationFlags = o.AuthorizationFlags
	f.VotingSystem = o.VotingSystem
	f.InitiativeThreshold = o.InitiativeThreshold
	f.InitiativeThresholdCurrency = o.InitiativeThresholdCurrency
	f.RestrictedQty = o.RestrictedQty

	return &f
}

func buildContractFormationFromContractAmendment(c contract.Contract,
	m *protocol.ContractAmendment) (contract.Contract, protocol.ContractFormation) {

	// bump the revision
	newRevision := c.Revision + 1

	o := protocol.NewContractFormation()
	o.Version = m.Version
	o.ContractName = m.ContractName
	o.ContractFileHash = m.ContractFileHash
	o.GoverningLaw = m.GoverningLaw
	o.Jurisdiction = m.Jurisdiction
	o.ContractExpiration = m.ContractExpiration
	o.URI = m.URI
	o.ContractRevision = newRevision
	o.IssuerID = m.IssuerID
	o.IssuerType = m.IssuerType
	o.ContractOperatorID = m.ContractOperatorID
	o.AuthorizationFlags = m.AuthorizationFlags
	o.VotingSystem = m.VotingSystem
	o.InitiativeThreshold = m.InitiativeThreshold
	o.InitiativeThresholdCurrency = m.InitiativeThresholdCurrency
	o.RestrictedQty = m.RestrictedQty

	// now assign contract values
	r := c

	r.ContractName = string(o.ContractName)
	r.ContractFileHash = string(o.ContractFileHash)
	r.GoverningLaw = string(o.GoverningLaw)
	r.Jurisdiction = string(o.Jurisdiction)
	r.ContractExpiration = o.ContractExpiration
	r.URI = string(o.URI)
	r.Revision = newRevision
	r.IssuerID = string(o.IssuerID)
	r.IssuerType = string(o.IssuerType)
	r.ContractOperatorID = string(o.ContractOperatorID)
	r.AuthorizationFlags = o.AuthorizationFlags
	r.VotingSystem = string(o.VotingSystem)
	r.InitiativeThreshold = o.InitiativeThreshold
	r.InitiativeThresholdCurrency = string(o.InitiativeThresholdCurrency)
	r.Qty = m.RestrictedQty

	return r, o
}

func buildAssetCreationFromAssetDefinition(m *protocol.AssetDefinition) protocol.AssetCreation {
	o := protocol.NewAssetCreation()
	o.AssetType = m.AssetType
	o.AssetID = m.AssetID
	o.AssetRevision = 0
	o.AuthorizationFlags = m.AuthorizationFlags
	o.VotingSystem = m.VotingSystem
	o.VoteMultiplier = m.VoteMultiplier
	o.Qty = m.Qty
	o.ContractFeeCurrency = m.ContractFeeCurrency
	o.ContractFeeVar = m.ContractFeeVar
	o.ContractFeeFixed = m.ContractFeeFixed
	o.Payload = m.Payload

	return o
}

func buildAssetCreationFromAssetModification(m *protocol.AssetModification) protocol.AssetCreation {
	o := protocol.NewAssetCreation()
	o.AssetType = m.AssetType
	o.AssetID = m.AssetID
	o.AssetRevision = m.AssetRevision + 1
	o.AuthorizationFlags = m.AuthorizationFlags
	o.VotingSystem = m.VotingSystem
	o.VoteMultiplier = m.VoteMultiplier
	o.Qty = m.Qty
	o.ContractFeeCurrency = m.ContractFeeCurrency
	o.ContractFeeVar = m.ContractFeeVar
	o.ContractFeeFixed = m.ContractFeeFixed
	o.Payload = m.Payload

	return o
}

func buildSettlementFromIssue(m *protocol.Send,
	hash chainhash.Hash,
	balance1 uint64,
	balance2 uint64) protocol.Settlement {

	s := protocol.NewSettlement()

	s.AssetType = m.AssetType
	s.AssetID = m.AssetID
	s.Party1TokenQty = balance1
	s.Party2TokenQty = balance2
	s.Timestamp = uint64(time.Now().Unix())

	return s
}

func buildSettlementFromExchange(m *protocol.Exchange,
	hash chainhash.Hash,
	party1Balance uint64,
	party2Balance uint64) protocol.Settlement {

	s := protocol.NewSettlement()

	// settlement only has 5 fields
	s.AssetType = m.Party1AssetType
	s.AssetID = m.Party1AssetID
	s.Party1TokenQty = party1Balance
	s.Party2TokenQty = party2Balance
	s.Timestamp = uint64(time.Now().Unix())

	return s
}

func buildVoteFromInitiative(hash chainhash.Hash,
	m *protocol.Initiative) protocol.Vote {

	v := protocol.NewVote()

	v.AssetType = m.AssetType
	v.AssetID = m.AssetID
	v.VoteType = m.VoteType
	v.VoteOptions = m.VoteOptions
	v.VoteMax = m.VoteMax
	v.VoteLogic = m.VoteLogic
	v.ProposalDescription = m.ProposalDescription
	v.ProposalDocumentHash = v.ProposalDocumentHash
	v.VoteCutOffTimestamp = v.VoteCutOffTimestamp
	v.Timestamp = uint64(time.Now().Unix())

	return v
}

func buildVoteFromReferendum(hash chainhash.Hash,
	m *protocol.Referendum) protocol.Vote {

	// NOTE This message is exactly the same structure as Initiative, and
	// could be factored out of the protocol.
	//
	// This only differences exist in the TX.
	v := protocol.NewVote()

	v.AssetType = m.AssetType
	v.AssetID = m.AssetID
	v.VoteType = m.VoteType
	v.VoteOptions = m.VoteOptions
	v.VoteMax = m.VoteMax
	v.VoteLogic = m.VoteLogic
	v.ProposalDescription = m.ProposalDescription
	v.ProposalDocumentHash = v.ProposalDocumentHash
	v.VoteCutOffTimestamp = v.VoteCutOffTimestamp
	v.Timestamp = uint64(time.Now().Unix())

	return v
}

func buildEnforcementFromOrder(m *protocol.Order,
	hash chainhash.Hash) protocol.OpReturnMessage {

	switch m.ComplianceAction {
	case protocol.ComplianceActionFreeze:
		f := buildFreezeFromOrder(m, hash)
		return &f

	case protocol.ComplianceActionThaw:
		t := buildThawFromOrder(m, hash)
		return &t

	case protocol.ComplianceActionConfiscation:
		c := buildConfiscationFromOrder(m, hash)
		return &c

	default:
		return nil
	}
}

func buildFreezeFromOrder(m *protocol.Order,
	hash chainhash.Hash) protocol.Freeze {

	f := protocol.NewFreeze()
	f.AssetID = m.AssetID
	f.AssetType = m.AssetType
	f.Timestamp = uint64(time.Now().Unix())
	f.Qty = m.Qty
	f.Message = m.Message
	f.Expiration = m.Expiration

	return f
}

func buildThawFromOrder(m *protocol.Order, hash chainhash.Hash) protocol.Thaw {
	t := protocol.NewThaw()

	t.AssetID = m.AssetID
	t.AssetType = m.AssetType
	t.Timestamp = uint64(time.Now().Unix())
	t.Qty = m.Qty
	t.Message = m.Message

	return t
}

func buildConfiscationFromOrder(m *protocol.Order,
	hash chainhash.Hash) protocol.Confiscation {

	c := protocol.NewConfiscation()

	// the target and deposit balances will be set elsewhere
	c.AssetID = m.AssetID
	c.AssetType = m.AssetType
	c.Timestamp = uint64(time.Now().Unix())
	c.Message = m.Message

	return c
}

func buildResultFromVoteResult(vote contract.Vote) protocol.Result {
	result := protocol.NewResult()
	result.AssetType = []byte(vote.AssetType)
	result.AssetID = []byte(vote.AssetID)
	result.VoteType = vote.VoteType
	result.VoteTxnID = []byte(vote.RefTxnIDHash)

	// get the result from the vote, which is a map of counts
	voteResult := *vote.Result

	// maximum seen vote count for an option
	max := uint64(0)

	for _, option := range vote.VoteOptions {
		count := voteResult[option]

		if count < max {
			continue
		}

		max = count
	}

	// we know the largest value, find any values with that count. there
	// can be more than one as it is possible for a vote to draw.
	winners := make([]byte, 16, 16)
	winnerIdx := 0

	for _, option := range vote.VoteOptions {
		count := voteResult[option]

		if count != max {
			continue
		}

		winners[winnerIdx] = option
		winnerIdx++
	}

	result.Result = winners

	return result
}

// hashToBytes returns a Hash in little endian format as Hash.CloneByte()
// returns bytes in big endian format.
func hashToBytes(hash chainhash.Hash) []byte {
	b, _ := hex.DecodeString(hash.String())

	return b
}
