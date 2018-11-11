package txbuilder

type TxOutputType uint

const (
	OutputTypeP2PK    TxOutputType = iota
	OutputTypeReturn
)

const (
	StringP2pk              = "p2pk"
	StringReturn            = "return"
)

func (s TxOutputType) String() string {
	switch s {
	case OutputTypeP2PK:
		return StringP2pk
	case OutputTypeReturn:
		return StringReturn
	default:
		return "unknown"
	}
}
