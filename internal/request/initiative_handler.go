package request

import (
	"context"
	"errors"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type initiativeHandler struct{}

func newInitiativeHandler() initiativeHandler {
	return initiativeHandler{}
}

func (h initiativeHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	initiative, ok := r.m.(*protocol.Initiative)
	if !ok {
		return nil, errors.New("Not *protocol.Initiative")
	}

	// Contract
	c := r.contract

	// Vote <- Initiative
	vote := protocol.NewVote()
	vote.AssetType = initiative.AssetType
	vote.AssetID = initiative.AssetID
	vote.VoteType = initiative.VoteType
	vote.VoteOptions = initiative.VoteOptions
	vote.VoteMax = initiative.VoteMax
	vote.VoteLogic = initiative.VoteLogic
	vote.ProposalDescription = initiative.ProposalDescription
	vote.ProposalDocumentHash = initiative.ProposalDocumentHash
	vote.VoteCutOffTimestamp = initiative.VoteCutOffTimestamp
	vote.Timestamp = uint64(time.Now().Unix())

	// create the Vote
	v := contract.NewVoteFromProtocolVote(r.senders[0].Address.EncodeAddress(), &vote)

	// record the UTXO for later when we need to send the Result when the
	// Vote cutoff time passes.
	v.UTXO = txbuilder.NewUTXOFromTX(*r.tx, 1)

	// add the Vote to the Contract
	c.Votes[v.RefTxnIDHash] = v

	// calculate the fee to pay the issuer.
	issuerFee := uint64(546)

	// how much to return to the sender?
	// 0 : Contract's Public Address
	// 1 : Issuer's Public Address

	issuerAddress, err := btcutil.DecodeAddress(c.IssuerAddress,
		&chaincfg.MainNetParams)

	if err != nil {
		return nil, err
	}

	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: issuerAddress,
			Value:   issuerFee,
		},
	}

	resp := contractResponse{
		Contract: c,
		Message:  &vote,
		outs:     outs,
	}

	return &resp, nil
}
