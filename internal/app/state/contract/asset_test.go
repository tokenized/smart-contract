package contract

import (
	"testing"
	"time"
)

func TestHoldingStatus_Expired(t *testing.T) {
	tests := []struct {
		name    string
		expires uint64
		want    bool
	}{
		{
			name:    "past",
			expires: uint64(time.Now().Unix() - 1),
			want:    true,
		},
		{
			name:    "future",
			expires: uint64(time.Now().Unix() + 10000000),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holdingStatus := HoldingStatus{
				Expires: tt.expires,
			}

			got := holdingStatus.Expired()

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
