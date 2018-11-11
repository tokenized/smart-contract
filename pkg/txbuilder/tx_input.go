package txbuilder

type TxInput struct {
	PkHash      []byte
	PrevIndex   int64
	PrevHash    string
	Value       uint64
}
