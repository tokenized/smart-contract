package vote

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var (
	// ErrNotFound abstracts the standard not found error.
	ErrNotFound = errors.New("Vote not found")
)

// Retrieve gets the specified vote from the database.
func Retrieve(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, voteID *protocol.TxId) (*state.Vote, error) {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Retrieve")
	defer span.End()

	// Find vote in storage
	v, err := Fetch(ctx, dbConn, contractPKH, voteID)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// Create the vote
func Create(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash, voteID *protocol.TxId, nv *NewVote,
	now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Create")
	defer span.End()

	// Set up vote
	var v state.Vote

	// Get current state
	err := node.Convert(ctx, &nv, &v)
	if err != nil {
		return err
	}

	v.CreatedAt = now
	v.UpdatedAt = now

	return Save(ctx, dbConn, contractPKH, &v)
}

// Update the vote
func Update(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	voteTxId *protocol.TxId, uv *UpdateVote, now protocol.Timestamp) error {
	ctx, span := trace.StartSpan(ctx, "internal.vote.Update")
	defer span.End()

	// Find vote
	v, err := Fetch(ctx, dbConn, contractPKH, voteTxId)
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
	if uv.AppliedTxId != nil {
		v.AppliedTxId = *uv.AppliedTxId
	}
	if uv.NewBallot != nil {
		v.Ballots = append(v.Ballots, uv.NewBallot)
	}

	v.UpdatedAt = now

	return Save(ctx, dbConn, contractPKH, v)
}

func CheckBallot(ctx context.Context, vt *state.Vote, holderPKH *protocol.PublicKeyHash) error {
	for _, bt := range vt.Ballots {
		if bytes.Equal(bt.PKH.Bytes(), holderPKH.Bytes()) {
			return errors.New("Ballot already accepted for pkh")
		}
	}

	return nil
}

func AddBallot(ctx context.Context, dbConn *db.DB, contractPKH *protocol.PublicKeyHash,
	vt *state.Vote, ballot *state.Ballot, now protocol.Timestamp) error {
	for _, bt := range vt.Ballots {
		if bytes.Equal(bt.PKH.Bytes(), ballot.PKH.Bytes()) {
			return errors.New("Ballot already accepted for pkh")
		}
	}

	uv := UpdateVote{NewBallot: ballot}

	if err := Update(ctx, dbConn, contractPKH, &vt.VoteTxId, &uv, now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}

	vt.Ballots = append(vt.Ballots, ballot)
	return nil
}

// CalculateResults calculates the result of a completed vote.
func CalculateResults(ctx context.Context, vt *state.Vote, proposal *protocol.Proposal, votingSystem *protocol.VotingSystem) ([]uint64, string, error) {

	results := make([]uint64, proposal.VoteMax)
	votedQuantity := uint64(0)
	var score uint64
	for _, ballot := range vt.Ballots {
		for i, choice := range ballot.Vote {
			switch votingSystem.TallyLogic {
			case 0: // Standard
				score = ballot.Quantity
			case 1: // Weighted
				score = ballot.Quantity * uint64(int(proposal.VoteMax)-i)
			default:
				return nil, "", fmt.Errorf("Unsupported tally logic : %d", votingSystem.TallyLogic)
			}

			for j, option := range proposal.VoteOptions {
				if option == choice {
					results[j] += score
					votedQuantity += score
					break
				}
			}
		}
	}

	var winners bytes.Buffer
	var highestIndex int
	var highestScore uint64
	scored := make(map[int]bool)
	for {
		highestIndex = -1
		highestScore = 0
		for i, result := range results {
			_, exists := scored[i]
			if exists {
				continue
			}

			if result <= highestScore {
				continue
			}

			switch votingSystem.VoteType {
			case 'R': // Relative
				if float32(result)/float32(votedQuantity) >= float32(votingSystem.ThresholdPercentage)/100.0 {
					highestIndex = i
					highestScore = result
				}
			case 'A': // Absolute
				if float32(result)/float32(vt.TokenQty) >= float32(votingSystem.ThresholdPercentage)/100.0 {
					highestIndex = i
					highestScore = result
				}
			case 'P': // Plurality
				highestIndex = i
				highestScore = result
			}
		}

		if highestIndex == -1 {
			break // No more valid results
		}
		winners.WriteByte(proposal.VoteOptions[highestIndex])
		scored[highestIndex] = true
	}

	return results, winners.String(), nil
}

func ValidateVotingSystem(system *protocol.VotingSystem) error {
	if system.VoteType != 'R' && system.VoteType != 'A' && system.VoteType != 'P' {
		return fmt.Errorf("Threshold Percentage out of range : %c", system.VoteType)
	}
	if system.ThresholdPercentage == 0 || system.ThresholdPercentage >= 100 {
		return fmt.Errorf("Threshold Percentage out of range : %d", system.ThresholdPercentage)
	}
	if system.TallyLogic != 0 && system.TallyLogic != 1 {
		return fmt.Errorf("Tally Logic invalid : %d", system.TallyLogic)
	}
	return nil
}
