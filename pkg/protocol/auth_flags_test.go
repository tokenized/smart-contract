package protocol

import (
	"testing"
)

func TestIsAuthorized(t *testing.T) {
	tests := []struct {
		name  string
		state uint16
		flags uint16
		want  bool
	}{
		{
			name:  "exact match",
			state: FlagAmendment + FlagVoteRequired,
			flags: FlagAmendment + FlagVoteRequired,
			want:  true,
		},
		{
			name:  "more state than flags",
			state: FlagAmendment + FlagVoteRequired + FlagAmendAuthFlags,
			flags: FlagAmendment + FlagVoteRequired,
			want:  true,
		},
		{
			name:  "insufficient flags",
			state: FlagAmendment + FlagVoteRequired,
			flags: FlagAmendment + FlagVoteRequired + FlagAmendAuthFlags,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthorized(tt.state, tt.flags)

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
