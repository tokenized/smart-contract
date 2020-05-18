package wire

type Block interface {
	GetHeader() BlockHeader
	IsMerkleRootValid() bool

	GetTxCount() uint64
	GetNextTx() (*MsgTx, error)
	ResetTxs()
}
