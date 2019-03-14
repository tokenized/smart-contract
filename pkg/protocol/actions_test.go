package protocol

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestAssetDefinition(t *testing.T) {
	// Create a randomized object
	initialMessage := AssetDefinition{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// AssetAuthFlags (bin)
	// bin test not setup

	// TransfersPermitted (bool)
	// bool test not setup

	// TradeRestrictions (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+4))
		}
		initialMessage.TradeRestrictions = string(text)
	}

	// EnforcementOrdersPermitted (bool)
	// bool test not setup

	// VoteMultiplier (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	// bool test not setup

	// InitiativeProposal (bool)
	// bool test not setup

	// AssetModificationGovernance (bool)
	// bool test not setup

	// TokenQty (uint)
	// uint test not setup

	// ContractFeeCurrency (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+11))
		}
		initialMessage.ContractFeeCurrency = string(text)
	}

	// ContractFeeVar (float)
	// float test not setup

	// ContractFeeFixed (float)
	// float test not setup

	// AssetPayload (varbin)
	initialMessage.AssetPayload = make([]byte, 0, 16)
	for i := uint64(0); i < 16; i++ {
		initialMessage.AssetPayload = append(initialMessage.AssetPayload, byte(65+i+14))
	}

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := AssetDefinition{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// AssetAuthFlags (bin)
	// bin test compare not setup

	// TransfersPermitted (bool)
	// bool test compare not setup

	// TradeRestrictions (fixedchar)
	if initialMessage.TradeRestrictions != decodedMessage.TradeRestrictions {
		t.Errorf("TradeRestrictions doesn't match : %s != %s", initialMessage.TradeRestrictions, decodedMessage.TradeRestrictions)
	}

	// EnforcementOrdersPermitted (bool)
	// bool test compare not setup

	// VoteMultiplier (uint)
	// uint test compare not setup

	// ReferendumProposal (bool)
	// bool test compare not setup

	// InitiativeProposal (bool)
	// bool test compare not setup

	// AssetModificationGovernance (bool)
	// bool test compare not setup

	// TokenQty (uint)
	// uint test compare not setup

	// ContractFeeCurrency (fixedchar)
	if initialMessage.ContractFeeCurrency != decodedMessage.ContractFeeCurrency {
		t.Errorf("ContractFeeCurrency doesn't match : %s != %s", initialMessage.ContractFeeCurrency, decodedMessage.ContractFeeCurrency)
	}

	// ContractFeeVar (float)
	// float test compare not setup

	// ContractFeeFixed (float)
	// float test compare not setup

	// AssetPayload (varbin)
	if !bytes.Equal(initialMessage.AssetPayload, decodedMessage.AssetPayload) {
		t.Errorf("AssetPayload doesn't match : %x != %x", initialMessage.AssetPayload, decodedMessage.AssetPayload)
	}
}

func TestAssetCreation(t *testing.T) {
	// Create a randomized object
	initialMessage := AssetCreation{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// AssetAuthFlags (bin)
	// bin test not setup

	// TransfersPermitted (bool)
	// bool test not setup

	// TradeRestrictions (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+4))
		}
		initialMessage.TradeRestrictions = string(text)
	}

	// EnforcementOrdersPermitted (bool)
	// bool test not setup

	// VoteMultiplier (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	// bool test not setup

	// InitiativeProposal (bool)
	// bool test not setup

	// AssetModificationGovernance (bool)
	// bool test not setup

	// TokenQty (uint)
	// uint test not setup

	// ContractFeeCurrency (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+11))
		}
		initialMessage.ContractFeeCurrency = string(text)
	}

	// ContractFeeVar (float)
	// float test not setup

	// ContractFeeFixed (float)
	// float test not setup

	// AssetPayload (varbin)
	initialMessage.AssetPayload = make([]byte, 0, 16)
	for i := uint64(0); i < 16; i++ {
		initialMessage.AssetPayload = append(initialMessage.AssetPayload, byte(65+i+14))
	}

	// AssetRevision (uint)
	// uint test not setup

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := AssetCreation{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// AssetAuthFlags (bin)
	// bin test compare not setup

	// TransfersPermitted (bool)
	// bool test compare not setup

	// TradeRestrictions (fixedchar)
	if initialMessage.TradeRestrictions != decodedMessage.TradeRestrictions {
		t.Errorf("TradeRestrictions doesn't match : %s != %s", initialMessage.TradeRestrictions, decodedMessage.TradeRestrictions)
	}

	// EnforcementOrdersPermitted (bool)
	// bool test compare not setup

	// VoteMultiplier (uint)
	// uint test compare not setup

	// ReferendumProposal (bool)
	// bool test compare not setup

	// InitiativeProposal (bool)
	// bool test compare not setup

	// AssetModificationGovernance (bool)
	// bool test compare not setup

	// TokenQty (uint)
	// uint test compare not setup

	// ContractFeeCurrency (fixedchar)
	if initialMessage.ContractFeeCurrency != decodedMessage.ContractFeeCurrency {
		t.Errorf("ContractFeeCurrency doesn't match : %s != %s", initialMessage.ContractFeeCurrency, decodedMessage.ContractFeeCurrency)
	}

	// ContractFeeVar (float)
	// float test compare not setup

	// ContractFeeFixed (float)
	// float test compare not setup

	// AssetPayload (varbin)
	if !bytes.Equal(initialMessage.AssetPayload, decodedMessage.AssetPayload) {
		t.Errorf("AssetPayload doesn't match : %x != %x", initialMessage.AssetPayload, decodedMessage.AssetPayload)
	}

	// AssetRevision (uint)
	// uint test compare not setup

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestAssetModification(t *testing.T) {
	// Create a randomized object
	initialMessage := AssetModification{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// AssetRevision (uint)
	// uint test not setup

	// Modifications (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.Modifications = append(initialMessage.Modifications, Amendment{})
	}

	// RefTxID (sha256)
	// sha256 test not setup

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := AssetModification{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// AssetRevision (uint)
	// uint test compare not setup

	// Modifications (Amendment[])
	if len(initialMessage.Modifications) != len(decodedMessage.Modifications) {
		t.Errorf("Modifications lengths don't match : %d != %d", len(initialMessage.Modifications), len(decodedMessage.Modifications))
	}

	// RefTxID (sha256)
	// sha256 test compare not setup
}

func TestContractOffer(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractOffer{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// ContractFileType (uint)
	// uint test not setup

	// ContractFile (varbin)
	initialMessage.ContractFile = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.ContractFile = append(initialMessage.ContractFile, byte(65+i+2))
	}

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+3))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+4))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// ContractExpiration (time)
	// time test not setup

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 6"

	// IssuerName (varchar)
	initialMessage.IssuerName = "Text 7"

	// IssuerType (fixedchar)
	// fixedchar test not setup

	// IssuerLogoURL (varchar)
	initialMessage.IssuerLogoURL = "Text 9"

	// ContractOperatorID (varchar)
	initialMessage.ContractOperatorID = "Text 10"

	// ContractAuthFlags (bin)
	// bin test not setup

	// VotingSystems (VotingSystem[])
	for i := 0; i < 2; i++ {
		initialMessage.VotingSystems = append(initialMessage.VotingSystems, VotingSystem{})
	}

	// RestrictedQtyAssets (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	// bool test not setup

	// InitiativeProposal (bool)
	// bool test not setup

	// Registries (Registry[])
	for i := 0; i < 2; i++ {
		initialMessage.Registries = append(initialMessage.Registries, Registry{})
	}

	// IssuerAddress (bool)
	// bool test not setup

	// UnitNumber (varchar)
	initialMessage.UnitNumber = "Text 18"

	// BuildingNumber (varchar)
	initialMessage.BuildingNumber = "Text 19"

	// Street (varchar)
	initialMessage.Street = "Text 20"

	// SuburbCity (varchar)
	initialMessage.SuburbCity = "Text 21"

	// TerritoryStateProvinceCode (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+22))
		}
		initialMessage.TerritoryStateProvinceCode = string(text)
	}

	// CountryCode (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+23))
		}
		initialMessage.CountryCode = string(text)
	}

	// PostalZIPCode (varchar)
	initialMessage.PostalZIPCode = "Text 24"

	// EmailAddress (varchar)
	initialMessage.EmailAddress = "Text 25"

	// PhoneNumber (varchar)
	initialMessage.PhoneNumber = "Text 26"

	// KeyRoles (KeyRole[])
	for i := 0; i < 2; i++ {
		initialMessage.KeyRoles = append(initialMessage.KeyRoles, KeyRole{})
	}

	// NotableRoles (NotableRole[])
	for i := 0; i < 2; i++ {
		initialMessage.NotableRoles = append(initialMessage.NotableRoles, NotableRole{})
	}

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := ContractOffer{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// ContractFileType (uint)
	// uint test compare not setup

	// ContractFile (varbin)
	if !bytes.Equal(initialMessage.ContractFile, decodedMessage.ContractFile) {
		t.Errorf("ContractFile doesn't match : %x != %x", initialMessage.ContractFile, decodedMessage.ContractFile)
	}

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// ContractExpiration (time)
	// time test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// IssuerName (varchar)
	if initialMessage.IssuerName != decodedMessage.IssuerName {
		t.Errorf("IssuerName doesn't match : %s != %s", initialMessage.IssuerName, decodedMessage.IssuerName)
	}

	// IssuerType (fixedchar)
	// fixedchar test compare not setup

	// IssuerLogoURL (varchar)
	if initialMessage.IssuerLogoURL != decodedMessage.IssuerLogoURL {
		t.Errorf("IssuerLogoURL doesn't match : %s != %s", initialMessage.IssuerLogoURL, decodedMessage.IssuerLogoURL)
	}

	// ContractOperatorID (varchar)
	if initialMessage.ContractOperatorID != decodedMessage.ContractOperatorID {
		t.Errorf("ContractOperatorID doesn't match : %s != %s", initialMessage.ContractOperatorID, decodedMessage.ContractOperatorID)
	}

	// ContractAuthFlags (bin)
	// bin test compare not setup

	// VotingSystems (VotingSystem[])
	if len(initialMessage.VotingSystems) != len(decodedMessage.VotingSystems) {
		t.Errorf("VotingSystems lengths don't match : %d != %d", len(initialMessage.VotingSystems), len(decodedMessage.VotingSystems))
	}

	// RestrictedQtyAssets (uint)
	// uint test compare not setup

	// ReferendumProposal (bool)
	// bool test compare not setup

	// InitiativeProposal (bool)
	// bool test compare not setup

	// Registries (Registry[])
	if len(initialMessage.Registries) != len(decodedMessage.Registries) {
		t.Errorf("Registries lengths don't match : %d != %d", len(initialMessage.Registries), len(decodedMessage.Registries))
	}

	// IssuerAddress (bool)
	// bool test compare not setup

	// UnitNumber (varchar)
	if initialMessage.UnitNumber != decodedMessage.UnitNumber {
		t.Errorf("UnitNumber doesn't match : %s != %s", initialMessage.UnitNumber, decodedMessage.UnitNumber)
	}

	// BuildingNumber (varchar)
	if initialMessage.BuildingNumber != decodedMessage.BuildingNumber {
		t.Errorf("BuildingNumber doesn't match : %s != %s", initialMessage.BuildingNumber, decodedMessage.BuildingNumber)
	}

	// Street (varchar)
	if initialMessage.Street != decodedMessage.Street {
		t.Errorf("Street doesn't match : %s != %s", initialMessage.Street, decodedMessage.Street)
	}

	// SuburbCity (varchar)
	if initialMessage.SuburbCity != decodedMessage.SuburbCity {
		t.Errorf("SuburbCity doesn't match : %s != %s", initialMessage.SuburbCity, decodedMessage.SuburbCity)
	}

	// TerritoryStateProvinceCode (fixedchar)
	if initialMessage.TerritoryStateProvinceCode != decodedMessage.TerritoryStateProvinceCode {
		t.Errorf("TerritoryStateProvinceCode doesn't match : %s != %s", initialMessage.TerritoryStateProvinceCode, decodedMessage.TerritoryStateProvinceCode)
	}

	// CountryCode (fixedchar)
	if initialMessage.CountryCode != decodedMessage.CountryCode {
		t.Errorf("CountryCode doesn't match : %s != %s", initialMessage.CountryCode, decodedMessage.CountryCode)
	}

	// PostalZIPCode (varchar)
	if initialMessage.PostalZIPCode != decodedMessage.PostalZIPCode {
		t.Errorf("PostalZIPCode doesn't match : %s != %s", initialMessage.PostalZIPCode, decodedMessage.PostalZIPCode)
	}

	// EmailAddress (varchar)
	if initialMessage.EmailAddress != decodedMessage.EmailAddress {
		t.Errorf("EmailAddress doesn't match : %s != %s", initialMessage.EmailAddress, decodedMessage.EmailAddress)
	}

	// PhoneNumber (varchar)
	if initialMessage.PhoneNumber != decodedMessage.PhoneNumber {
		t.Errorf("PhoneNumber doesn't match : %s != %s", initialMessage.PhoneNumber, decodedMessage.PhoneNumber)
	}

	// KeyRoles (KeyRole[])
	if len(initialMessage.KeyRoles) != len(decodedMessage.KeyRoles) {
		t.Errorf("KeyRoles lengths don't match : %d != %d", len(initialMessage.KeyRoles), len(decodedMessage.KeyRoles))
	}

	// NotableRoles (NotableRole[])
	if len(initialMessage.NotableRoles) != len(decodedMessage.NotableRoles) {
		t.Errorf("NotableRoles lengths don't match : %d != %d", len(initialMessage.NotableRoles), len(decodedMessage.NotableRoles))
	}
}

func TestContractFormation(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractFormation{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// ContractFileType (uint)
	// uint test not setup

	// ContractFile (varbin)
	initialMessage.ContractFile = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.ContractFile = append(initialMessage.ContractFile, byte(65+i+2))
	}

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+3))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+4))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// ContractExpiration (time)
	// time test not setup

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 6"

	// IssuerName (varchar)
	initialMessage.IssuerName = "Text 7"

	// IssuerType (fixedchar)
	// fixedchar test not setup

	// IssuerLogoURL (varchar)
	initialMessage.IssuerLogoURL = "Text 9"

	// ContractOperatorID (varchar)
	initialMessage.ContractOperatorID = "Text 10"

	// ContractAuthFlags (bin)
	// bin test not setup

	// VotingSystems (VotingSystem[])
	for i := 0; i < 2; i++ {
		initialMessage.VotingSystems = append(initialMessage.VotingSystems, VotingSystem{})
	}

	// RestrictedQtyAssets (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	// bool test not setup

	// InitiativeProposal (bool)
	// bool test not setup

	// Registries (Registry[])
	for i := 0; i < 2; i++ {
		initialMessage.Registries = append(initialMessage.Registries, Registry{})
	}

	// IssuerAddress (bool)
	// bool test not setup

	// UnitNumber (varchar)
	initialMessage.UnitNumber = "Text 18"

	// BuildingNumber (varchar)
	initialMessage.BuildingNumber = "Text 19"

	// Street (varchar)
	initialMessage.Street = "Text 20"

	// SuburbCity (varchar)
	initialMessage.SuburbCity = "Text 21"

	// TerritoryStateProvinceCode (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+22))
		}
		initialMessage.TerritoryStateProvinceCode = string(text)
	}

	// CountryCode (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+23))
		}
		initialMessage.CountryCode = string(text)
	}

	// PostalZIPCode (varchar)
	initialMessage.PostalZIPCode = "Text 24"

	// EmailAddress (varchar)
	initialMessage.EmailAddress = "Text 25"

	// PhoneNumber (varchar)
	initialMessage.PhoneNumber = "Text 26"

	// KeyRoles (KeyRole[])
	for i := 0; i < 2; i++ {
		initialMessage.KeyRoles = append(initialMessage.KeyRoles, KeyRole{})
	}

	// NotableRoles (NotableRole[])
	for i := 0; i < 2; i++ {
		initialMessage.NotableRoles = append(initialMessage.NotableRoles, NotableRole{})
	}

	// ContractRevision (uint)
	// uint test not setup

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := ContractFormation{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// ContractFileType (uint)
	// uint test compare not setup

	// ContractFile (varbin)
	if !bytes.Equal(initialMessage.ContractFile, decodedMessage.ContractFile) {
		t.Errorf("ContractFile doesn't match : %x != %x", initialMessage.ContractFile, decodedMessage.ContractFile)
	}

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// ContractExpiration (time)
	// time test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// IssuerName (varchar)
	if initialMessage.IssuerName != decodedMessage.IssuerName {
		t.Errorf("IssuerName doesn't match : %s != %s", initialMessage.IssuerName, decodedMessage.IssuerName)
	}

	// IssuerType (fixedchar)
	// fixedchar test compare not setup

	// IssuerLogoURL (varchar)
	if initialMessage.IssuerLogoURL != decodedMessage.IssuerLogoURL {
		t.Errorf("IssuerLogoURL doesn't match : %s != %s", initialMessage.IssuerLogoURL, decodedMessage.IssuerLogoURL)
	}

	// ContractOperatorID (varchar)
	if initialMessage.ContractOperatorID != decodedMessage.ContractOperatorID {
		t.Errorf("ContractOperatorID doesn't match : %s != %s", initialMessage.ContractOperatorID, decodedMessage.ContractOperatorID)
	}

	// ContractAuthFlags (bin)
	// bin test compare not setup

	// VotingSystems (VotingSystem[])
	if len(initialMessage.VotingSystems) != len(decodedMessage.VotingSystems) {
		t.Errorf("VotingSystems lengths don't match : %d != %d", len(initialMessage.VotingSystems), len(decodedMessage.VotingSystems))
	}

	// RestrictedQtyAssets (uint)
	// uint test compare not setup

	// ReferendumProposal (bool)
	// bool test compare not setup

	// InitiativeProposal (bool)
	// bool test compare not setup

	// Registries (Registry[])
	if len(initialMessage.Registries) != len(decodedMessage.Registries) {
		t.Errorf("Registries lengths don't match : %d != %d", len(initialMessage.Registries), len(decodedMessage.Registries))
	}

	// IssuerAddress (bool)
	// bool test compare not setup

	// UnitNumber (varchar)
	if initialMessage.UnitNumber != decodedMessage.UnitNumber {
		t.Errorf("UnitNumber doesn't match : %s != %s", initialMessage.UnitNumber, decodedMessage.UnitNumber)
	}

	// BuildingNumber (varchar)
	if initialMessage.BuildingNumber != decodedMessage.BuildingNumber {
		t.Errorf("BuildingNumber doesn't match : %s != %s", initialMessage.BuildingNumber, decodedMessage.BuildingNumber)
	}

	// Street (varchar)
	if initialMessage.Street != decodedMessage.Street {
		t.Errorf("Street doesn't match : %s != %s", initialMessage.Street, decodedMessage.Street)
	}

	// SuburbCity (varchar)
	if initialMessage.SuburbCity != decodedMessage.SuburbCity {
		t.Errorf("SuburbCity doesn't match : %s != %s", initialMessage.SuburbCity, decodedMessage.SuburbCity)
	}

	// TerritoryStateProvinceCode (fixedchar)
	if initialMessage.TerritoryStateProvinceCode != decodedMessage.TerritoryStateProvinceCode {
		t.Errorf("TerritoryStateProvinceCode doesn't match : %s != %s", initialMessage.TerritoryStateProvinceCode, decodedMessage.TerritoryStateProvinceCode)
	}

	// CountryCode (fixedchar)
	if initialMessage.CountryCode != decodedMessage.CountryCode {
		t.Errorf("CountryCode doesn't match : %s != %s", initialMessage.CountryCode, decodedMessage.CountryCode)
	}

	// PostalZIPCode (varchar)
	if initialMessage.PostalZIPCode != decodedMessage.PostalZIPCode {
		t.Errorf("PostalZIPCode doesn't match : %s != %s", initialMessage.PostalZIPCode, decodedMessage.PostalZIPCode)
	}

	// EmailAddress (varchar)
	if initialMessage.EmailAddress != decodedMessage.EmailAddress {
		t.Errorf("EmailAddress doesn't match : %s != %s", initialMessage.EmailAddress, decodedMessage.EmailAddress)
	}

	// PhoneNumber (varchar)
	if initialMessage.PhoneNumber != decodedMessage.PhoneNumber {
		t.Errorf("PhoneNumber doesn't match : %s != %s", initialMessage.PhoneNumber, decodedMessage.PhoneNumber)
	}

	// KeyRoles (KeyRole[])
	if len(initialMessage.KeyRoles) != len(decodedMessage.KeyRoles) {
		t.Errorf("KeyRoles lengths don't match : %d != %d", len(initialMessage.KeyRoles), len(decodedMessage.KeyRoles))
	}

	// NotableRoles (NotableRole[])
	if len(initialMessage.NotableRoles) != len(decodedMessage.NotableRoles) {
		t.Errorf("NotableRoles lengths don't match : %d != %d", len(initialMessage.NotableRoles), len(decodedMessage.NotableRoles))
	}

	// ContractRevision (uint)
	// uint test compare not setup

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestContractAmendment(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractAmendment{}
	// ChangeIssuerAddress (bool)
	// bool test not setup

	// ChangeOperatorAddress (bool)
	// bool test not setup

	// ContractRevision (uint)
	// uint test not setup

	// Amendments (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.Amendments = append(initialMessage.Amendments, Amendment{})
	}

	// RefTxID (SHA256)
	// SHA256 test not setup

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := ContractAmendment{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ChangeIssuerAddress (bool)
	// bool test compare not setup

	// ChangeOperatorAddress (bool)
	// bool test compare not setup

	// ContractRevision (uint)
	// uint test compare not setup

	// Amendments (Amendment[])
	if len(initialMessage.Amendments) != len(decodedMessage.Amendments) {
		t.Errorf("Amendments lengths don't match : %d != %d", len(initialMessage.Amendments), len(decodedMessage.Amendments))
	}

	// RefTxID (SHA256)
	// SHA256 test compare not setup
}

func TestStaticContractFormation(t *testing.T) {
	// Create a randomized object
	initialMessage := StaticContractFormation{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// ContractType (varchar)
	initialMessage.ContractType = "Text 1"

	// ContractFileType (uint)
	// uint test not setup

	// ContractFile (varbin)
	initialMessage.ContractFile = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.ContractFile = append(initialMessage.ContractFile, byte(65+i+3))
	}

	// ContractRevision (uint)
	// uint test not setup

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+5))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+6))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// EffectiveDate (time)
	// time test not setup

	// ContractExpiration (time)
	// time test not setup

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 9"

	// PrevRevTxID (sha256)
	// sha256 test not setup

	// Entities (Entity[])
	for i := 0; i < 2; i++ {
		initialMessage.Entities = append(initialMessage.Entities, Entity{})
	}

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := StaticContractFormation{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// ContractType (varchar)
	if initialMessage.ContractType != decodedMessage.ContractType {
		t.Errorf("ContractType doesn't match : %s != %s", initialMessage.ContractType, decodedMessage.ContractType)
	}

	// ContractFileType (uint)
	// uint test compare not setup

	// ContractFile (varbin)
	if !bytes.Equal(initialMessage.ContractFile, decodedMessage.ContractFile) {
		t.Errorf("ContractFile doesn't match : %x != %x", initialMessage.ContractFile, decodedMessage.ContractFile)
	}

	// ContractRevision (uint)
	// uint test compare not setup

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// EffectiveDate (time)
	// time test compare not setup

	// ContractExpiration (time)
	// time test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// PrevRevTxID (sha256)
	// sha256 test compare not setup

	// Entities (Entity[])
	if len(initialMessage.Entities) != len(decodedMessage.Entities) {
		t.Errorf("Entities lengths don't match : %d != %d", len(initialMessage.Entities), len(decodedMessage.Entities))
	}
}

func TestOrder(t *testing.T) {
	// Create a randomized object
	initialMessage := Order{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// ComplianceAction (fixedchar)
	// fixedchar test not setup

	// TargetAddresses (TargetAddress[])
	for i := 0; i < 2; i++ {
		initialMessage.TargetAddresses = append(initialMessage.TargetAddresses, TargetAddress{})
	}

	// DepositAddress (Address)
	initialMessage.DepositAddress = Address{}

	// AuthorityName (varchar)
	initialMessage.AuthorityName = "Text 5"

	// SigAlgoAddressList (uint)
	// uint test not setup

	// AuthorityPublicKey (varchar)
	initialMessage.AuthorityPublicKey = "Text 7"

	// OrderSignature (varchar)
	initialMessage.OrderSignature = "Text 8"

	// SupportingEvidenceHash (sha256)
	// sha256 test not setup

	// RefTxnID (sha256)
	// sha256 test not setup

	// FreezePeriod (time)
	// time test not setup

	// Message (varchar)
	initialMessage.Message = "Text 12"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Order{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// ComplianceAction (fixedchar)
	// fixedchar test compare not setup

	// TargetAddresses (TargetAddress[])
	if len(initialMessage.TargetAddresses) != len(decodedMessage.TargetAddresses) {
		t.Errorf("TargetAddresses lengths don't match : %d != %d", len(initialMessage.TargetAddresses), len(decodedMessage.TargetAddresses))
	}

	// DepositAddress (Address)
	if initialMessage.DepositAddress != decodedMessage.DepositAddress {
		t.Errorf("DepositAddress doesn't match : %v != %v", initialMessage.DepositAddress, decodedMessage.DepositAddress)
	}

	// AuthorityName (varchar)
	if initialMessage.AuthorityName != decodedMessage.AuthorityName {
		t.Errorf("AuthorityName doesn't match : %s != %s", initialMessage.AuthorityName, decodedMessage.AuthorityName)
	}

	// SigAlgoAddressList (uint)
	// uint test compare not setup

	// AuthorityPublicKey (varchar)
	if initialMessage.AuthorityPublicKey != decodedMessage.AuthorityPublicKey {
		t.Errorf("AuthorityPublicKey doesn't match : %s != %s", initialMessage.AuthorityPublicKey, decodedMessage.AuthorityPublicKey)
	}

	// OrderSignature (varchar)
	if initialMessage.OrderSignature != decodedMessage.OrderSignature {
		t.Errorf("OrderSignature doesn't match : %s != %s", initialMessage.OrderSignature, decodedMessage.OrderSignature)
	}

	// SupportingEvidenceHash (sha256)
	// sha256 test compare not setup

	// RefTxnID (sha256)
	// sha256 test compare not setup

	// FreezePeriod (time)
	// time test compare not setup

	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestFreeze(t *testing.T) {
	// Create a randomized object
	initialMessage := Freeze{}
	// Addresses (Address[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, Address{})
	}

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Freeze{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (Address[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestThaw(t *testing.T) {
	// Create a randomized object
	initialMessage := Thaw{}
	// Addresses (Address[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, Address{})
	}

	// RefTxnID (sha256)
	// sha256 test not setup

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Thaw{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (Address[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// RefTxnID (sha256)
	// sha256 test compare not setup

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestConfiscation(t *testing.T) {
	// Create a randomized object
	initialMessage := Confiscation{}
	// Addresses (Address[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, Address{})
	}

	// DepositQty (uint)
	// uint test not setup

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Confiscation{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (Address[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// DepositQty (uint)
	// uint test compare not setup

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestReconciliation(t *testing.T) {
	// Create a randomized object
	initialMessage := Reconciliation{}
	// Addresses (Address[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, Address{})
	}

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Reconciliation{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (Address[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestInitiative(t *testing.T) {
	// Create a randomized object
	initialMessage := Initiative{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// VoteSystem (uint)
	// uint test not setup

	// Proposal (bool)
	// bool test not setup

	// ProposedChanges (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.ProposedChanges = append(initialMessage.ProposedChanges, Amendment{})
	}

	// VoteOptions (varchar)
	initialMessage.VoteOptions = "Text 5"

	// VoteMax (uint)
	// uint test not setup

	// ProposalDescription (varchar)
	initialMessage.ProposalDescription = "Text 7"

	// ProposalDocumentHash (sha256)
	// sha256 test not setup

	// VoteCutOffTimestamp (time)
	// time test not setup

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Initiative{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// VoteSystem (uint)
	// uint test compare not setup

	// Proposal (bool)
	// bool test compare not setup

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteOptions (varchar)
	if initialMessage.VoteOptions != decodedMessage.VoteOptions {
		t.Errorf("VoteOptions doesn't match : %s != %s", initialMessage.VoteOptions, decodedMessage.VoteOptions)
	}

	// VoteMax (uint)
	// uint test compare not setup

	// ProposalDescription (varchar)
	if initialMessage.ProposalDescription != decodedMessage.ProposalDescription {
		t.Errorf("ProposalDescription doesn't match : %s != %s", initialMessage.ProposalDescription, decodedMessage.ProposalDescription)
	}

	// ProposalDocumentHash (sha256)
	// sha256 test compare not setup

	// VoteCutOffTimestamp (time)
	// time test compare not setup
}

func TestReferendum(t *testing.T) {
	// Create a randomized object
	initialMessage := Referendum{}
	// AssetSpecificVote (bool)
	// bool test not setup

	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+2))
		}
		initialMessage.AssetID = string(text)
	}

	// VoteSystem (uint)
	// uint test not setup

	// Proposal (bool)
	// bool test not setup

	// ProposedChanges (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.ProposedChanges = append(initialMessage.ProposedChanges, Amendment{})
	}

	// VoteOptions (varchar)
	initialMessage.VoteOptions = "Text 6"

	// VoteMax (uint)
	// uint test not setup

	// ProposalDescription (varchar)
	initialMessage.ProposalDescription = "Text 8"

	// ProposalDocumentHash (sha256)
	// sha256 test not setup

	// VoteCutOffTimestamp (time)
	// time test not setup

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Referendum{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetSpecificVote (bool)
	// bool test compare not setup

	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// VoteSystem (uint)
	// uint test compare not setup

	// Proposal (bool)
	// bool test compare not setup

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteOptions (varchar)
	if initialMessage.VoteOptions != decodedMessage.VoteOptions {
		t.Errorf("VoteOptions doesn't match : %s != %s", initialMessage.VoteOptions, decodedMessage.VoteOptions)
	}

	// VoteMax (uint)
	// uint test compare not setup

	// ProposalDescription (varchar)
	if initialMessage.ProposalDescription != decodedMessage.ProposalDescription {
		t.Errorf("ProposalDescription doesn't match : %s != %s", initialMessage.ProposalDescription, decodedMessage.ProposalDescription)
	}

	// ProposalDocumentHash (sha256)
	// sha256 test compare not setup

	// VoteCutOffTimestamp (time)
	// time test compare not setup
}

func TestVote(t *testing.T) {
	// Create a randomized object
	initialMessage := Vote{}
	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Vote{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestBallotCast(t *testing.T) {
	// Create a randomized object
	initialMessage := BallotCast{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// VoteTxnID (sha256)
	// sha256 test not setup

	// Vote (varchar)
	initialMessage.Vote = "Text 3"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := BallotCast{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// VoteTxnID (sha256)
	// sha256 test compare not setup

	// Vote (varchar)
	if initialMessage.Vote != decodedMessage.Vote {
		t.Errorf("Vote doesn't match : %s != %s", initialMessage.Vote, decodedMessage.Vote)
	}
}

func TestBallotCounted(t *testing.T) {
	// Create a randomized object
	initialMessage := BallotCounted{}
	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := BallotCounted{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestResult(t *testing.T) {
	// Create a randomized object
	initialMessage := Result{}
	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+0))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetID (fixedchar)
	{
		text := make([]byte, 0, 32)
		for i := uint64(0); i < 32; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetID = string(text)
	}

	// Proposal (bool)
	// bool test not setup

	// ProposedChanges (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.ProposedChanges = append(initialMessage.ProposedChanges, Amendment{})
	}

	// VoteTxnID (sha256)
	// sha256 test not setup

	// VoteOptionsCount (uint)
	// uint test not setup

	// OptionXTally (uint)
	// uint test not setup

	// Result (varchar)
	initialMessage.Result = "Text 7"

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Result{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetID (fixedchar)
	if initialMessage.AssetID != decodedMessage.AssetID {
		t.Errorf("AssetID doesn't match : %s != %s", initialMessage.AssetID, decodedMessage.AssetID)
	}

	// Proposal (bool)
	// bool test compare not setup

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteTxnID (sha256)
	// sha256 test compare not setup

	// VoteOptionsCount (uint)
	// uint test compare not setup

	// OptionXTally (uint)
	// uint test compare not setup

	// Result (varchar)
	if initialMessage.Result != decodedMessage.Result {
		t.Errorf("Result doesn't match : %s != %s", initialMessage.Result, decodedMessage.Result)
	}

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestMessage(t *testing.T) {
	// Create a randomized object
	initialMessage := Message{}
	// QtyReceivingAddresses (uint8)
	// uint8 test not setup

	// AddressIndexes (uint16[])
	for i := 0; i < 5; i++ {
		var item uint16
		initialMessage.AddressIndexes = append(initialMessage.AddressIndexes, item)
	}

	// MessageType (fixedchar)
	{
		text := make([]byte, 0, 2)
		for i := uint64(0); i < 2; i++ {
			text = append(text, byte(65+i+2))
		}
		initialMessage.MessageType = string(text)
	}

	// MessagePayload (varchar)
	initialMessage.MessagePayload = "Text 3"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Message{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// QtyReceivingAddresses (uint8)
	// uint8 test compare not setup

	// AddressIndexes (uint16[])
	if len(initialMessage.AddressIndexes) != len(decodedMessage.AddressIndexes) {
		t.Errorf("AddressIndexes lengths don't match : %d != %d", len(initialMessage.AddressIndexes), len(decodedMessage.AddressIndexes))
	}
	for i, value := range initialMessage.AddressIndexes {
		if value != decodedMessage.AddressIndexes[i] {
			t.Errorf("AddressIndexes value %d doesn't match : %v != %v", i, value, decodedMessage.AddressIndexes[i])
		}
	}

	// MessageType (fixedchar)
	if initialMessage.MessageType != decodedMessage.MessageType {
		t.Errorf("MessageType doesn't match : %s != %s", initialMessage.MessageType, decodedMessage.MessageType)
	}

	// MessagePayload (varchar)
	if initialMessage.MessagePayload != decodedMessage.MessagePayload {
		t.Errorf("MessagePayload doesn't match : %s != %s", initialMessage.MessagePayload, decodedMessage.MessagePayload)
	}
}

func TestRejection(t *testing.T) {
	// Create a randomized object
	initialMessage := Rejection{}
	// QtyReceivingAddresses (uint8)
	// uint8 test not setup

	// AddressIndexes (uint16[])
	for i := 0; i < 5; i++ {
		var item uint16
		initialMessage.AddressIndexes = append(initialMessage.AddressIndexes, item)
	}

	// RejectionType (uint)
	// uint test not setup

	// MessagePayload (varchar)
	initialMessage.MessagePayload = "Text 3"

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Rejection{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// QtyReceivingAddresses (uint8)
	// uint8 test compare not setup

	// AddressIndexes (uint16[])
	if len(initialMessage.AddressIndexes) != len(decodedMessage.AddressIndexes) {
		t.Errorf("AddressIndexes lengths don't match : %d != %d", len(initialMessage.AddressIndexes), len(decodedMessage.AddressIndexes))
	}
	for i, value := range initialMessage.AddressIndexes {
		if value != decodedMessage.AddressIndexes[i] {
			t.Errorf("AddressIndexes value %d doesn't match : %v != %v", i, value, decodedMessage.AddressIndexes[i])
		}
	}

	// RejectionType (uint)
	// uint test compare not setup

	// MessagePayload (varchar)
	if initialMessage.MessagePayload != decodedMessage.MessagePayload {
		t.Errorf("MessagePayload doesn't match : %s != %s", initialMessage.MessagePayload, decodedMessage.MessagePayload)
	}

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}

func TestEstablishment(t *testing.T) {
	// Create a randomized object
	initialMessage := Establishment{}
	// Message (varchar)
	initialMessage.Message = "Text 0"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Establishment{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestAddition(t *testing.T) {
	// Create a randomized object
	initialMessage := Addition{}
	// Message (varchar)
	initialMessage.Message = "Text 0"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Addition{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestAlteration(t *testing.T) {
	// Create a randomized object
	initialMessage := Alteration{}
	// Message (varchar)
	initialMessage.Message = "Text 0"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Alteration{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestRemoval(t *testing.T) {
	// Create a randomized object
	initialMessage := Removal{}
	// Message (varchar)
	initialMessage.Message = "Text 0"

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Removal{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestTransfer(t *testing.T) {
	// Create a randomized object
	initialMessage := Transfer{}
	// Assets (AssetTransfer[])
	for i := 0; i < 2; i++ {
		initialMessage.Assets = append(initialMessage.Assets, AssetTransfer{})
	}

	// OfferExpiry (time)
	// time test not setup

	// ExchangeFeeCurrency (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+2))
		}
		initialMessage.ExchangeFeeCurrency = string(text)
	}

	// ExchangeFeeVar (float)
	// float test not setup

	// ExchangeFeeFixed (float)
	// float test not setup

	// ExchangeFeeAddress (Address)
	initialMessage.ExchangeFeeAddress = Address{}

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Transfer{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Assets (AssetTransfer[])
	if len(initialMessage.Assets) != len(decodedMessage.Assets) {
		t.Errorf("Assets lengths don't match : %d != %d", len(initialMessage.Assets), len(decodedMessage.Assets))
	}

	// OfferExpiry (time)
	// time test compare not setup

	// ExchangeFeeCurrency (fixedchar)
	if initialMessage.ExchangeFeeCurrency != decodedMessage.ExchangeFeeCurrency {
		t.Errorf("ExchangeFeeCurrency doesn't match : %s != %s", initialMessage.ExchangeFeeCurrency, decodedMessage.ExchangeFeeCurrency)
	}

	// ExchangeFeeVar (float)
	// float test compare not setup

	// ExchangeFeeFixed (float)
	// float test compare not setup

	// ExchangeFeeAddress (Address)
	if initialMessage.ExchangeFeeAddress != decodedMessage.ExchangeFeeAddress {
		t.Errorf("ExchangeFeeAddress doesn't match : %v != %v", initialMessage.ExchangeFeeAddress, decodedMessage.ExchangeFeeAddress)
	}
}

func TestSettlement(t *testing.T) {
	// Create a randomized object
	initialMessage := Settlement{}
	// Assets (AssetSettlement[])
	for i := 0; i < 2; i++ {
		initialMessage.Assets = append(initialMessage.Assets, AssetSettlement{})
	}

	// Timestamp (timestamp)
	initialMessage.Timestamp = uint64(time.Now().UnixNano())

	// Encode message
	initialEncoding, err := initialMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial encoding : %d bytes", len(initialEncoding))

	// Decode message
	decodedMessage := Settlement{}

	n, err := decodedMessage.write(initialEncoding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Decoded : %d bytes", n)

	if n != len(initialEncoding) {
		t.Fatalf("got %v, want %v", n, len(initialEncoding))
	}

	// Serializing the message should give us the same bytes
	secondEncoding, err := decodedMessage.serialize()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(initialEncoding, secondEncoding) {
		t.Errorf("got\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !reflect.DeepEqual(initialMessage, decodedMessage) {
	// 	t.Errorf("\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Assets (AssetSettlement[])
	if len(initialMessage.Assets) != len(decodedMessage.Assets) {
		t.Errorf("Assets lengths don't match : %d != %d", len(initialMessage.Assets), len(decodedMessage.Assets))
	}

	// Timestamp (timestamp)
	if initialMessage.Timestamp != decodedMessage.Timestamp {
		t.Errorf("Timestamp doesn't match : %d != %d", initialMessage.Timestamp, decodedMessage.Timestamp)
	}
}
