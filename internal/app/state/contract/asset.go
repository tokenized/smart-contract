package contract

import (
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
)

type Asset struct {
	ID                 string             `json:"id"`
	Type               string             `json:"type"`
	Revision           uint16             `json:"revision"`
	AuthorizationFlags []byte             `json:"auth_flags"`
	VotingSystem       byte               `json:"voting_system"`
	VoteMultiplier     uint8              `json:"vote_multiplier"`
	Qty                uint64             `json:"qty"`
	TxnFeeType         byte               `json:"txn_fee_type"`
	TxnFeeCurrency     string             `json:"txn_fee_currency"`
	TxnFeeVar          float32            `json:"txn_fee_var,omitempty"`
	TxnFeeFixed        float32            `json:"txn_fee_fixed,omitempty"`
	Holdings           map[string]Holding `json:"holdings"`
	CreatedAt          int64              `json:"created_at"`
}

func NewAssetFromAssetModification(a Asset,
	am *protocol.AssetModification) Asset {

	a.Type = string(am.AssetType)
	a.AuthorizationFlags = am.AuthorizationFlags
	a.VotingSystem = am.VotingSystem
	a.VoteMultiplier = am.VoteMultiplier
	a.Qty = am.Qty
	// a.TxnFeeCurrency = string(am.TxnFeeCurrency)
	// a.TxnFeeVar = am.TxnFeeVar
	// a.TxnFeeFixed = am.TxnFeeFixed

	if a.AuthorizationFlags == nil {
		a.AuthorizationFlags = []byte{}
	}

	return a
}

func NewAssetFromAssetDefinition(am *protocol.AssetDefinition,
	holding Holding) Asset {

	holdings := map[string]Holding{
		holding.Address: holding,
	}

	a := Asset{
		ID:                 string(am.AssetID),
		Type:               string(am.AssetType),
		Revision:           0,
		AuthorizationFlags: am.AuthorizationFlags,
		VoteMultiplier:     am.VoteMultiplier,
		Qty:                am.Qty,
		// TxnFeeCurrency:     string(am.TxnFeeCurrency),
		// TxnFeeVar:          am.TxnFeeVar,
		// TxnFeeFixed:        am.TxnFeeFixed,
		Holdings:  holdings,
		CreatedAt: time.Now().UnixNano(),
	}

	if a.AuthorizationFlags == nil {
		a.AuthorizationFlags = []byte{}
	}

	return a
}
