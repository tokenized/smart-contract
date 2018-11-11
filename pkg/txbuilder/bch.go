package txbuilder

const (
	satsPerUnit = 100000000.0
)

func ConvertBCHToSatoshis(f float32) uint64 {
	return uint64(f * satsPerUnit)
}
