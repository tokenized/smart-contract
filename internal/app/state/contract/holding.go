package contract

import (
	"time"
)

type Holding struct {
	Address       string         `json:"address"`
	Balance       uint64         `json:"balance"`
	HoldingStatus *HoldingStatus `json:"order_status,omitempty"`
	CreatedAt     int64          `json:"created_at"`
}

func NewHolding(address string, qty uint64) Holding {
	return Holding{
		Address:   address,
		Balance:   qty,
		CreatedAt: time.Now().UnixNano(),
	}
}
