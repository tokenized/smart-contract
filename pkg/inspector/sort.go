package inspector

// Timestamp returns the timestamp of the response action. Other action types will return nil.
// The timestamp is used to ensure the order that the smart contract originally processed the
//   request is retained.
func (itx *Transaction) Timestamp() *uint64 {
	return GetProtocolTimestamp(itx, itx.MsgProto)
}

// Implement sort.Interface to sort outgoing inspector transactions by timestamp. This is so during
//   recovery of off chain state from on chain txs, the outgoing txs can be processed in the
//   original order.
type TransactionList []*Transaction

// Len is part of sort.Interface.
func (s *TransactionList) Len() int {
	return len(*s)
}

// Swap is part of sort.Interface.
func (s *TransactionList) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

// Less is part of sort.Interface.
func (s *TransactionList) Less(i, j int) bool {
	iTime := (*s)[i].Timestamp()
	if iTime == nil {
		return false
	}
	jTime := (*s)[j].Timestamp()
	if jTime == nil {
		return false
	}
	return *iTime < *jTime
}
