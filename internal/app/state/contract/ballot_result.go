package contract

type BallotResult map[uint8]uint64

func NewBallotResult() BallotResult {
	return map[uint8]uint64{}
}
