package state

import "github.com/tokenized/smart-contract/pkg/protocol"

func NewRegistry(data protocol.Registry, textEncoding uint8) Registry {
	result := Registry{
		Name:      data.Name.EncodedString(textEncoding),
		URL:       data.URL.EncodedString(textEncoding),
		PublicKey: data.PublicKey.EncodedString(textEncoding),
	}
	return result
}

func NewKeyRole(data protocol.KeyRole, textEncoding uint8) KeyRole {
	result := KeyRole{
		Type: data.Type,
		Name: data.Name.EncodedString(textEncoding),
	}
	return result
}

func NewNotableRole(data protocol.NotableRole, textEncoding uint8) NotableRole {
	result := NotableRole{
		Type: data.Type,
		Name: data.Name.EncodedString(textEncoding),
	}
	return result
}

func NewVotingSystem(data protocol.VotingSystem, textEncoding uint8) VotingSystem {
	result := VotingSystem{
		Name:                        data.Name.EncodedString(textEncoding),
		System:                      data.System,
		Method:                      data.Method,
		Logic:                       data.Logic,
		ThresholdPercentage:         data.ThresholdPercentage,
		VoteMultiplierPermitted:     data.VoteMultiplierPermitted,
		InitiativeThreshold:         data.InitiativeThreshold,
		InitiativeThresholdCurrency: data.InitiativeThresholdCurrency,
	}
	return result
}
