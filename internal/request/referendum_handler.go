package request

import (
	"context"
	"errors"
	"time"

	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type referendumHandler struct{}

func newReferendumHandler() referendumHandler {
	return referendumHandler{}
}

func (h referendumHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	referendum, ok := r.m.(*protocol.Referendum)
	if !ok {
		return nil, errors.New("Not *protocol.Referendum")
	}

	// Contract
	c := r.contract

	// Vote <- Referendum
	vote := protocol.NewVote()
	vote.AssetType = referendum.AssetType
	vote.AssetID = referendum.AssetID
	vote.VoteType = referendum.VoteType
	vote.VoteOptions = referendum.VoteOptions
	vote.VoteMax = referendum.VoteMax
	vote.VoteLogic = referendum.VoteLogic
	vote.ProposalDescription = referendum.ProposalDescription
	vote.ProposalDocumentHash = referendum.ProposalDocumentHash
	vote.VoteCutOffTimestamp = referendum.VoteCutOffTimestamp
	vote.Timestamp = uint64(time.Now().Unix())

	// create the Vote
	v := contract.NewVoteFromProtocolVote(r.senders[0].EncodeAddress(), &vote)

	// record the UTXO for later when we need to send the Result when the
	// Vote cutoff time passes.
	v.UTXO = txbuilder.NewUTXOFromTX(*r.tx, 1)

	// add the Vote to the Contract
	c.Votes[v.RefTxnIDHash] = v

	resp := contractResponse{
		Contract: c,
		Message:  &vote,
	}

	return &resp, nil
}
