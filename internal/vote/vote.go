package vote

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Vote not found")
)

// Retrieve gets the specified vote from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteID *bitcoin.Hash32) (*state.Vote, error) {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Retrieve")
	defer span.End()

	// Find vote in storage
	v, err := Fetch(ctx, dbConn, contractAddress, voteID)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Create the vote
func Create(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteID *bitcoin.Hash32, nv *NewVote, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Create")
	defer span.End()

	// Set up vote
	var v state.Vote

	// Get current state
	err := node.Convert(ctx, nv, &v)
	if err != nil {
		return err
	}

	v.InstrumentType = nv.InstrumentType
	v.InstrumentCode = nv.InstrumentCode
	v.Ballots = nv.Ballots // Doesn't come through json convert because it isn't a marshalable type
	v.CreatedAt = now
	v.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, &v)
}

// Update the vote
func Update(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteTxId *bitcoin.Hash32, uv *UpdateVote, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Update")
	defer span.End()

	// Find vote
	v, err := Fetch(ctx, dbConn, contractAddress, voteTxId)
	if err != nil {
		return ErrNotFound
	}

	if !v.CompletedAt.Equal(protocol.NewTimestamp(0)) {
		return errors.New("Vote already complete")
	}

	// Update fields
	if uv.CompletedAt != nil {
		v.CompletedAt = *uv.CompletedAt
	}
	if uv.Result != nil {
		v.Result = *uv.Result
	}
	if uv.OptionTally != nil {
		v.OptionTally = *uv.OptionTally
	}
	if uv.AppliedTxId != nil {
		v.AppliedTxId = uv.AppliedTxId
	}
	if uv.NewBallot != nil {
		hash, err := uv.NewBallot.Address.Hash()
		if err != nil {
			return errors.Wrap(err, "address hash")
		}
		v.Ballots[*hash] = *uv.NewBallot
	}

	v.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, v)
}

// Mark the vote as applied
func MarkApplied(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	voteTxId *bitcoin.Hash32, appliedTxID *bitcoin.Hash32, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.MarkApplied")
	defer span.End()

	// Find vote
	v, err := Fetch(ctx, dbConn, contractAddress, voteTxId)
	if err != nil {
		return ErrNotFound
	}

	v.AppliedTxId = appliedTxID
	v.UpdatedAt = now

	return Save(ctx, dbConn, contractAddress, v)
}

func AddBallot(ctx context.Context, dbConn *db.DB, contractAddress bitcoin.RawAddress,
	vt *state.Vote, ballot *state.Ballot, now protocol.Timestamp) error {

	uv := UpdateVote{NewBallot: ballot}

	if err := Update(ctx, dbConn, contractAddress, vt.VoteTxId, &uv, now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}

	hash, err := ballot.Address.Hash()
	if err != nil {
		return errors.Wrap(err, "address hash")
	}
	vt.Ballots[*hash] = *ballot
	return nil
}

// CalculateResults calculates the result of a completed vote.
func CalculateResults(ctx context.Context, vt *state.Vote, proposal *actions.Proposal,
	votingSystem *actions.VotingSystemField) ([]uint64, string, error) {

	floatTallys := make([]float32, len(proposal.VoteOptions))
	votedQuantity := uint64(0)
	var score float32
	for _, ballot := range vt.Ballots {
		if len(ballot.Vote) == 0 {
			continue // Skip ballots that weren't completed
		}
		for i, choice := range ballot.Vote {
			switch votingSystem.TallyLogic {
			case 0: // Standard
				score = float32(ballot.Quantity)
			case 1: // Weighted
				score = float32(ballot.Quantity) * (float32(int(proposal.VoteMax)-i) / float32(proposal.VoteMax))
			default:
				return nil, "", fmt.Errorf("Unsupported tally logic : %d", votingSystem.TallyLogic)
			}

			for j, option := range proposal.VoteOptions {
				if option == choice {
					floatTallys[j] += score
					break
				}
			}
		}

		votedQuantity += ballot.Quantity
	}

	var winners bytes.Buffer
	var highestIndex int
	var highestScore float32
	scored := make(map[int]bool)
	for {
		highestIndex = -1
		highestScore = 0.0
		for i, floatTally := range floatTallys {
			_, exists := scored[i]
			if exists {
				continue
			}

			if floatTally <= highestScore {
				continue
			}

			switch votingSystem.VoteType {
			case "R": // Relative
				if floatTally/float32(votedQuantity) >= float32(votingSystem.ThresholdPercentage)/100.0 {
					highestIndex = i
					highestScore = floatTally
				}
			case "A": // Absolute
				if floatTally/float32(vt.TokenQty) >= float32(votingSystem.ThresholdPercentage)/100.0 {
					highestIndex = i
					highestScore = floatTally
				}
			case "P": // Plurality
				highestIndex = i
				highestScore = floatTally
			}
		}

		if highestIndex == -1 {
			break // No more valid tallys
		}
		winners.WriteByte(proposal.VoteOptions[highestIndex])
		scored[highestIndex] = true
	}

	// Convert tallys back to integers
	tallys := make([]uint64, len(proposal.VoteOptions))
	for i, floatTally := range floatTallys {
		logger.Verbose(ctx, "Vote result %c : %d", proposal.VoteOptions[i], uint64(floatTally))
		tallys[i] = uint64(floatTally)
	}

	logger.Verbose(ctx, "Processed vote : winners %s", winners.String())
	return tallys, winners.String(), nil
}

func ValidateVotingSystem(system *actions.VotingSystemField) error {
	if system.VoteType != "R" && system.VoteType != "A" && system.VoteType != "P" {
		return fmt.Errorf("Unsupported vote type : %s", system.VoteType)
	}
	if system.ThresholdPercentage == 0 || system.ThresholdPercentage >= 100 {
		return fmt.Errorf("Threshold Percentage out of range : %d", system.ThresholdPercentage)
	}
	if system.TallyLogic != 0 && system.TallyLogic != 1 {
		return fmt.Errorf("Tally Logic invalid : %d", system.TallyLogic)
	}
	return nil
}
