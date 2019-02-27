package protocol

const (
	DustLimit    = uint64(546)
	LimitDefault = uint64(2000)
)

var (
	// Minimum is the minimum value of a transaction for the type.
	Minimum = map[string]uint64{
		"C1":          LimitDefault,
		"C2":          DustLimit,
		"C3":          LimitDefault,
		"A1":          LimitDefault,
		"A2":          DustLimit,
		"A3":          LimitDefault,
		"T1":          LimitDefault,
		"T2":          LimitDefault,
		"T3":          LimitDefault,
		"T4":          DustLimit,
		"G1":          LimitDefault,
		"G2":          LimitDefault,
		"G3":          DustLimit,
		"G4":          LimitDefault,
		"E1":          LimitDefault,
		"E2":          DustLimit,
		"E3":          DustLimit,
		"E4":          DustLimit,
		"R1":          DustLimit,
		"R2":          DustLimit,
		"R3":          DustLimit,
		"R4":          DustLimit,
		"M1":          LimitDefault,
		CodeRejection: DustLimit,
	}
)
