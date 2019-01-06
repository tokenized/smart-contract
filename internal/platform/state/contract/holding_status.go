package contract

import (
	"time"
)

type HoldingStatus struct {
	Code    string `json:"code"`
	Expires uint64 `json:"expires,omitempty"`
}

func (o HoldingStatus) Expired() bool {
	if o.Expires == 0 {
		return false
	}

	if time.Now().Unix() > int64(o.Expires) {
		// current time is after expiry, so this order has expired.
		return true
	}

	return false
}
