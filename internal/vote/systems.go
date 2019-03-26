package vote

import (
	"context"
	"fmt"
	"sort"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/state"

	"github.com/pkg/errors"
)

var (
	votingSystems = map[string]VotingSystem{
		"M": MajorityVote{},
		"A": AbsoluteMajority{},
		"P": PluralityVotingSystem{},
		"S": SuperMajority{},
		"T": AbsoluteSuperMajority{},
		"N": NoVotingRights{},
	}
)

// GetVotingSystemCode returns the most appropriate code for a VotingSystem
// given the Contract and Vote.
// func GetVotingSystemCode(c state.Contract, v state.Vote) (*string, error) {
// code := c.VotingSystem

// // This is not an asset vote, so we are using the Contract voting system
// if len(v.AssetCode) == 0 {
// return &code, nil
// }

// // This is an asset vote
// asset, ok := c.Assets[v.AssetCode]
// if !ok {
// return nil, errors.New("Asset not found")
// }

// // The asset has a voting system, use it
// if len(asset.VotingSystem) > 0 {
// return &asset.VotingSystem, nil
// }

// // Default to the contract voting system
// return &code, nil
// }

// VotingSystem defines the interface the various voting systems noted in
// the whitepaper.
type VotingSystem interface {
	// Winners returns a slice of options containing the winner, or in the
	// case of a draw, multiple winners.
	//
	// Unless there is a single winner, the vote has not been successful.
	Winners(context.Context, *db.DB, *state.Contract, *state.Vote) ([]uint8, error)
}

type baseVotingSystem struct{}

// Sort the slice of []uint8.
func (base baseVotingSystem) sort(b []uint8) []uint8 {
	sort.Slice(b, func(i, j int) bool {
		return b[i] < b[j]
	})

	return b
}

// NewVotingSystem returns the appropriate VotingSystem identified by the
// given code.
//
// If no matching VotingSystem is found an error is returned.
func NewVotingSystem(code string) (VotingSystem, error) {
	var vs VotingSystem

	vs, ok := votingSystems[code]
	if !ok {
		return nil, fmt.Errorf("No voting system found for : %v", code)
	}

	return vs, nil
}

// MajorityVote is the implementation of a "Majority Vote (M)" voting system.
//
// More than half (=>50%) wins. Abstentions/spoiled votes are not
// counted. If only 10% of the token owners vote, then it would take > 5.0%
// of the total possible votes to win the Majority.
type MajorityVote struct {
	baseVotingSystem
}

func (m MajorityVote) Winners(_ context.Context, _ *db.DB, contract *state.Contract, vote *state.Vote) ([]uint8, error) {
	// Get the totals
	totalValue := uint64(0)

	for _, v := range *vote.Result {
		totalValue += v
	}

	// To be a super majority, the total value of the vote of an option must
	// be >= 67% of the total vote value.
	minimum := float64(totalValue) * 0.5

	winners := []uint8{}

	for k, v := range *vote.Result {
		if float64(v) > minimum {
			winners = append(winners, k)

			// There will be only 1 winner for this voting system. A draw
			// isn't possible.
			break
		}
	}

	return m.sort(winners), nil
}

// AbsoluteMajority is the implementation of the "Absolute Majority (A)"
// voting system.
//
// More than half (>50%) wins. >50% of all token owners must vote for the
// vote to pass. Abstentions/spoiled votes only detract from the likelihood
// of the vote passing.
type AbsoluteMajority struct {
	baseVotingSystem
}

func (a AbsoluteMajority) Winners(ctx context.Context, dbConn *db.DB, c *state.Contract, v *state.Vote) ([]uint8, error) {
	// Number of asset holders
	tokenHolderCount := 0

	if v.AssetCode.IsZero() {
		for _, a := range c.AssetCodes {
			asset, err := asset.Retrieve(ctx, dbConn, &c.ID, &a)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to count holdings")
			}
			tokenHolderCount += len(asset.Holdings)
		}
	} else {
		asset, err := asset.Retrieve(ctx, dbConn, &c.ID, &v.AssetCode)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to count holdings")
		}

		tokenHolderCount += len(asset.Holdings)
	}

	// Get the ballot count
	ballotCount := len(v.Ballots)

	// Ballot count must be > 50% of possible ballots
	if float64(ballotCount)/float64(tokenHolderCount) <= 0.5 {
		return []uint8{}, nil
	}

	// Get the totals
	totalValue := uint64(0)

	for _, val := range *v.Result {
		totalValue += val
	}

	minimum := float64(totalValue) * 0.5

	winners := []uint8{}

	for k, val := range *v.Result {
		if float64(val) > minimum {
			winners = append(winners, k)
		}
	}

	return a.sort(winners), nil
}

// PluralityVotingSystem is the implementation of a Plurality Vote (P).
//
// The most favoured option is selected, regardless of the percentage of
// votes.
type PluralityVotingSystem struct {
	baseVotingSystem
}

func (p PluralityVotingSystem) Winners(_ context.Context, _ *db.DB, c *state.Contract, vote *state.Vote) ([]uint8, error) {
	// Get the highest vote
	max := ResultMaximum(*vote.Result)

	winners := []uint8{}

	for k, v := range *vote.Result {
		if v == max {
			winners = append(winners, k)
		}
	}

	return p.sort(winners), nil
}

// SuperMajority is the implemented of the "Supermajority (S)" voting system.
//
// More than two thirds (>67%) wins. Abstentions/spoiled votes are not
// counted. If only 10% of the token owners vote, then it would take 6.7% of
// the total possible votes to win the Supermajority.
type SuperMajority struct {
	baseVotingSystem
}

func (s SuperMajority) Winners(_ context.Context, _ *db.DB, c *state.Contract, v *state.Vote) ([]uint8, error) {
	// Get the totals
	totalValue := uint64(0)

	for _, val := range *v.Result {
		totalValue += val
	}

	// To be a super majority, the total value of the vote of an option must
	// be >= 67% of the total vote value.
	minimum := float64(totalValue) * 0.67

	winners := []uint8{}

	for k, val := range *v.Result {
		if float64(val) >= minimum {
			winners = append(winners, k)

			// There will be only 1 winner for this voting system. A draw
			// isn't possible.
			break
		}
	}

	return s.sort(winners), nil
}

// AbsoluteSuperMajority is the implementation of the "Absolute
// Supermajority (T)" voting system.
//
// More than two thirds (>67%) wins. >67% of all token owners must vote
// for the vote to pass. Abstentions/spoiled votes only detract from the
// likelihood of the vote passing.
type AbsoluteSuperMajority struct {
	baseVotingSystem
}

func (a AbsoluteSuperMajority) Winners(ctx context.Context, dbConn *db.DB, c *state.Contract, v *state.Vote) ([]uint8, error) {
	// Number of asset holders
	tokenHolderCount := 0

	if v.AssetCode.IsZero() {
		for _, a := range c.AssetCodes {
			asset, err := asset.Retrieve(ctx, dbConn, &c.ID, &a)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to count holdings")
			}
			tokenHolderCount += len(asset.Holdings)
		}
	} else {
		asset, err := asset.Retrieve(ctx, dbConn, &c.ID, &v.AssetCode)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to count holdings")
		}

		tokenHolderCount += len(asset.Holdings)
	}

	// Get the ballot count
	ballotCount := len(v.Ballots)

	// Ballot count must be >= 67% of possible ballots
	if float64(ballotCount)/float64(tokenHolderCount) < 0.67 {
		return []uint8{}, nil
	}

	// Get the totals
	totalValue := uint64(0)

	for _, val := range *v.Result {
		totalValue += val
	}

	// To win a option must have >= 0.67 of the vote
	minimum := float64(totalValue) * 0.67

	winners := []uint8{}

	for k, val := range *v.Result {
		if float64(val) >= minimum {
			winners = append(winners, k)
		}
	}

	return a.sort(winners), nil
}

// NoVotingRightsVotingSystem is the implementation of the "No Voting
// Rights" (N) voting system.
type NoVotingRights struct{}

func (v NoVotingRights) Winners(_ context.Context, _ *db.DB, _ *state.Contract, _ *state.Vote) ([]uint8, error) {
	return []uint8{}, nil
}
