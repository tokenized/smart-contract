package vote

import (
	"context"
	"time"

	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type VoteService struct{}

func NewVoteService() VoteService {
	return VoteService{}
}

func (v VoteService) handle(ctx context.Context, c contract.Contract) ([]contract.Vote, error) {

	votes := []contract.Vote{}

	for _, vote := range c.Votes {
		if vote.Result == nil && !vote.IsOpen(time.Now()) {
			// we can result this vote
			result := v.generateResult(c, vote)

			vote.Result = &result
			votes = append(votes, vote)
		}
	}

	return votes, nil
}

func (v VoteService) generateResult(c contract.Contract, vo contract.Vote) contract.BallotResult {
	// before this method can be called, Vote.VoteLogic must be verified as
	// a valid value (0, or 1).
	result := contract.NewBallotResult()

	// get all token owners
	// holdings := contract.getHoldings()

	for _, ballot := range vo.Ballots {
		// if the contract is a contract level vote, then any holder can vote.
		// but if the vote is on a specific asset id, the user must
		// have that asset id.
		if vo.AssetID != "" && ballot.AssetID != vo.AssetID {
			// this ballot cannot be accepted for this vote, wrong asset id
			continue
		}

		tokens := uint64(0)

		asset, ok := c.Assets[ballot.AssetID]
		if !ok {
			// skipping
			continue
		}

		holding, ok := asset.Holdings[ballot.Address]
		if !ok {
			// skipping
			continue
		}

		if holding.Balance == 0 {
			// skipping
			continue
		}

		tokens = uint64(holding.Balance)

		// if this is an asset level vote, the voter must have that asset

		// get the vote values the user sent
		values := ballot.Vote[:vo.VoteMax]

		for i, val := range values {
			// 0 - Standard Scoring (+1 * # of tokens owned),
			// 1 - Weighted Scoring (1st choice * Vote Max * # of tokens held,
			//     2nd choice * Vote Max-1 * # of tokens held,..etc.)

			// assuming VoteLogic == "0", as a valid VoteLogic has already
			// been verified.
			voteValue := tokens

			if vo.VoteLogic == protocol.VoteLogicWeighted {
				max := uint64(int(vo.VoteMax) - i)
				voteValue = max * tokens
			}

			result[val] += voteValue
		}

	}

	// discard any incorrect selections
	for k := range result {
		// delete any key that is not in the vote options
		found := false

		for _, o := range vo.VoteOptions {
			if o == k {
				found = true
				break
			}
		}

		if !found {
			// the option that was voted for wasn't found, remove it.
			delete(result, k)
		}
	}

	return result
}
