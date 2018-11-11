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
			state: ContractIssuerUpdate + ContractBindingInitiatives + ContractAssetWhitelist,
			flags: ContractIssuerUpdate + ContractBindingInitiatives + ContractAssetWhitelist,
			want:  true,
		},
		{
			name:  "more state than flags",
			state: ContractIssuerUpdate + ContractOwnerAmendments + ContractBindingInitiatives + ContractAssetWhitelist,
			flags: ContractIssuerUpdate + ContractBindingInitiatives + ContractAssetWhitelist,
			want:  true,
		},
		{
			name:  "insufficient flags",
			state: ContractBindingInitiatives + ContractAssetWhitelist,
			flags: ContractIssuerUpdate + ContractBindingInitiatives + ContractAssetWhitelist,
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
