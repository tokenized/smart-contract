package protocol

import (
	"bytes"
	"testing"
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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// AssetAuthFlags (bin)
	// bin test not setup

	// TransfersPermitted (bool)
	initialMessage.TransfersPermitted = true

	// TradeRestrictions (Polity)
	initialMessage.TradeRestrictions = Polity{}

	// EnforcementOrdersPermitted (bool)
	initialMessage.EnforcementOrdersPermitted = true

	// VoteMultiplier (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	initialMessage.ReferendumProposal = true

	// InitiativeProposal (bool)
	initialMessage.InitiativeProposal = true

	// AssetModificationGovernance (bool)
	initialMessage.AssetModificationGovernance = true

	// TokenQty (uint)
	// uint test not setup

	// AssetPayload (varbin)
	initialMessage.AssetPayload = make([]byte, 0, 16)
	for i := uint64(0); i < 16; i++ {
		initialMessage.AssetPayload = append(initialMessage.AssetPayload, byte(65+i+11))
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// AssetAuthFlags (bin)
	if initialMessage.AssetAuthFlags != decodedMessage.AssetAuthFlags {
		t.Errorf("AssetAuthFlags doesn't match : %v != %v", initialMessage.AssetAuthFlags, decodedMessage.AssetAuthFlags)
	}

	// TransfersPermitted (bool)
	if initialMessage.TransfersPermitted != decodedMessage.TransfersPermitted {
		t.Errorf("TransfersPermitted doesn't match : %v != %v", initialMessage.TransfersPermitted, decodedMessage.TransfersPermitted)
	}

	// TradeRestrictions (Polity)
	if len(initialMessage.TradeRestrictions.Items) != len(decodedMessage.TradeRestrictions.Items) {
		t.Errorf("TradeRestrictions length doesn't match : %d != %d", initialMessage.TradeRestrictions, decodedMessage.TradeRestrictions)
	}
	for i, value := range initialMessage.TradeRestrictions.Items {
		if !bytes.Equal(value[:], decodedMessage.TradeRestrictions.Items[i][:]) {
			t.Errorf("TradeRestrictions.Items[%d] doesn't match : %s != %s", i, string(value[:]), string(decodedMessage.TradeRestrictions.Items[i][:]))
		}
	}

	// EnforcementOrdersPermitted (bool)
	if initialMessage.EnforcementOrdersPermitted != decodedMessage.EnforcementOrdersPermitted {
		t.Errorf("EnforcementOrdersPermitted doesn't match : %v != %v", initialMessage.EnforcementOrdersPermitted, decodedMessage.EnforcementOrdersPermitted)
	}

	// VoteMultiplier (uint)
	if initialMessage.VoteMultiplier != decodedMessage.VoteMultiplier {
		t.Errorf("VoteMultiplier doesn't match : %v != %v", initialMessage.VoteMultiplier, decodedMessage.VoteMultiplier)
	}

	// ReferendumProposal (bool)
	if initialMessage.ReferendumProposal != decodedMessage.ReferendumProposal {
		t.Errorf("ReferendumProposal doesn't match : %v != %v", initialMessage.ReferendumProposal, decodedMessage.ReferendumProposal)
	}

	// InitiativeProposal (bool)
	if initialMessage.InitiativeProposal != decodedMessage.InitiativeProposal {
		t.Errorf("InitiativeProposal doesn't match : %v != %v", initialMessage.InitiativeProposal, decodedMessage.InitiativeProposal)
	}

	// AssetModificationGovernance (bool)
	if initialMessage.AssetModificationGovernance != decodedMessage.AssetModificationGovernance {
		t.Errorf("AssetModificationGovernance doesn't match : %v != %v", initialMessage.AssetModificationGovernance, decodedMessage.AssetModificationGovernance)
	}

	// TokenQty (uint)
	if initialMessage.TokenQty != decodedMessage.TokenQty {
		t.Errorf("TokenQty doesn't match : %v != %v", initialMessage.TokenQty, decodedMessage.TokenQty)
	}

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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// AssetAuthFlags (bin)
	// bin test not setup

	// TransfersPermitted (bool)
	initialMessage.TransfersPermitted = true

	// TradeRestrictions (Polity)
	initialMessage.TradeRestrictions = Polity{}

	// EnforcementOrdersPermitted (bool)
	initialMessage.EnforcementOrdersPermitted = true

	// VoteMultiplier (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	initialMessage.ReferendumProposal = true

	// InitiativeProposal (bool)
	initialMessage.InitiativeProposal = true

	// AssetModificationGovernance (bool)
	initialMessage.AssetModificationGovernance = true

	// TokenQty (uint)
	// uint test not setup

	// AssetPayload (varbin)
	initialMessage.AssetPayload = make([]byte, 0, 16)
	for i := uint64(0); i < 16; i++ {
		initialMessage.AssetPayload = append(initialMessage.AssetPayload, byte(65+i+11))
	}

	// AssetRevision (uint)
	// uint test not setup

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// AssetAuthFlags (bin)
	if initialMessage.AssetAuthFlags != decodedMessage.AssetAuthFlags {
		t.Errorf("AssetAuthFlags doesn't match : %v != %v", initialMessage.AssetAuthFlags, decodedMessage.AssetAuthFlags)
	}

	// TransfersPermitted (bool)
	if initialMessage.TransfersPermitted != decodedMessage.TransfersPermitted {
		t.Errorf("TransfersPermitted doesn't match : %v != %v", initialMessage.TransfersPermitted, decodedMessage.TransfersPermitted)
	}

	// TradeRestrictions (Polity)
	if len(initialMessage.TradeRestrictions.Items) != len(decodedMessage.TradeRestrictions.Items) {
		t.Errorf("TradeRestrictions length doesn't match : %d != %d", initialMessage.TradeRestrictions, decodedMessage.TradeRestrictions)
	}
	for i, value := range initialMessage.TradeRestrictions.Items {
		if !bytes.Equal(value[:], decodedMessage.TradeRestrictions.Items[i][:]) {
			t.Errorf("TradeRestrictions.Items[%d] doesn't match : %s != %s", i, string(value[:]), string(decodedMessage.TradeRestrictions.Items[i][:]))
		}
	}

	// EnforcementOrdersPermitted (bool)
	if initialMessage.EnforcementOrdersPermitted != decodedMessage.EnforcementOrdersPermitted {
		t.Errorf("EnforcementOrdersPermitted doesn't match : %v != %v", initialMessage.EnforcementOrdersPermitted, decodedMessage.EnforcementOrdersPermitted)
	}

	// VoteMultiplier (uint)
	if initialMessage.VoteMultiplier != decodedMessage.VoteMultiplier {
		t.Errorf("VoteMultiplier doesn't match : %v != %v", initialMessage.VoteMultiplier, decodedMessage.VoteMultiplier)
	}

	// ReferendumProposal (bool)
	if initialMessage.ReferendumProposal != decodedMessage.ReferendumProposal {
		t.Errorf("ReferendumProposal doesn't match : %v != %v", initialMessage.ReferendumProposal, decodedMessage.ReferendumProposal)
	}

	// InitiativeProposal (bool)
	if initialMessage.InitiativeProposal != decodedMessage.InitiativeProposal {
		t.Errorf("InitiativeProposal doesn't match : %v != %v", initialMessage.InitiativeProposal, decodedMessage.InitiativeProposal)
	}

	// AssetModificationGovernance (bool)
	if initialMessage.AssetModificationGovernance != decodedMessage.AssetModificationGovernance {
		t.Errorf("AssetModificationGovernance doesn't match : %v != %v", initialMessage.AssetModificationGovernance, decodedMessage.AssetModificationGovernance)
	}

	// TokenQty (uint)
	if initialMessage.TokenQty != decodedMessage.TokenQty {
		t.Errorf("TokenQty doesn't match : %v != %v", initialMessage.TokenQty, decodedMessage.TokenQty)
	}

	// AssetPayload (varbin)
	if !bytes.Equal(initialMessage.AssetPayload, decodedMessage.AssetPayload) {
		t.Errorf("AssetPayload doesn't match : %x != %x", initialMessage.AssetPayload, decodedMessage.AssetPayload)
	}

	// AssetRevision (uint)
	if initialMessage.AssetRevision != decodedMessage.AssetRevision {
		t.Errorf("AssetRevision doesn't match : %v != %v", initialMessage.AssetRevision, decodedMessage.AssetRevision)
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// AssetRevision (uint)
	// uint test not setup

	// Modifications (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.Modifications = append(initialMessage.Modifications, Amendment{})
	}

	// RefTxID (TxId)
	initialMessage.RefTxID = TxId{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// AssetRevision (uint)
	if initialMessage.AssetRevision != decodedMessage.AssetRevision {
		t.Errorf("AssetRevision doesn't match : %v != %v", initialMessage.AssetRevision, decodedMessage.AssetRevision)
	}

	// Modifications (Amendment[])
	if len(initialMessage.Modifications) != len(decodedMessage.Modifications) {
		t.Errorf("Modifications lengths don't match : %d != %d", len(initialMessage.Modifications), len(decodedMessage.Modifications))
	}

	// RefTxID (TxId)
	// TxId test compare not setup
}

func TestContractOffer(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractOffer{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// BodyOfAgreementType (uint)
	// uint test not setup

	// BodyOfAgreement (varbin)
	initialMessage.BodyOfAgreement = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.BodyOfAgreement = append(initialMessage.BodyOfAgreement, byte(65+i+2))
	}

	// ContractType (varchar)
	initialMessage.ContractType = "Text 3"

	// SupportingDocsFileType (uint)
	// uint test not setup

	// SupportingDocs (varbin)
	initialMessage.SupportingDocs = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.SupportingDocs = append(initialMessage.SupportingDocs, byte(65+i+5))
	}

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+6))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+7))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// ContractExpiration (Timestamp)
	initialMessage.ContractExpiration = Timestamp{}

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 9"

	// Issuer (Entity)
	initialMessage.Issuer = Entity{}

	// IssuerLogoURL (varchar)
	initialMessage.IssuerLogoURL = "Text 11"

	// ContractOperatorIncluded (bool)
	initialMessage.ContractOperatorIncluded = true

	// ContractOperator (Entity)
	initialMessage.ContractOperator = Entity{}

	// ContractAuthFlags (bin)
	// bin test not setup

	// ContractFee (uint)
	// uint test not setup

	// VotingSystems (VotingSystem[])
	for i := 0; i < 2; i++ {
		initialMessage.VotingSystems = append(initialMessage.VotingSystems, VotingSystem{})
	}

	// RestrictedQtyAssets (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	initialMessage.ReferendumProposal = true

	// InitiativeProposal (bool)
	initialMessage.InitiativeProposal = true

	// Registries (Registry[])
	for i := 0; i < 2; i++ {
		initialMessage.Registries = append(initialMessage.Registries, Registry{})
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// BodyOfAgreementType (uint)
	if initialMessage.BodyOfAgreementType != decodedMessage.BodyOfAgreementType {
		t.Errorf("BodyOfAgreementType doesn't match : %v != %v", initialMessage.BodyOfAgreementType, decodedMessage.BodyOfAgreementType)
	}

	// BodyOfAgreement (varbin)
	if !bytes.Equal(initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement) {
		t.Errorf("BodyOfAgreement doesn't match : %x != %x", initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement)
	}

	// ContractType (varchar)
	if initialMessage.ContractType != decodedMessage.ContractType {
		t.Errorf("ContractType doesn't match : %s != %s", initialMessage.ContractType, decodedMessage.ContractType)
	}

	// SupportingDocsFileType (uint)
	if initialMessage.SupportingDocsFileType != decodedMessage.SupportingDocsFileType {
		t.Errorf("SupportingDocsFileType doesn't match : %v != %v", initialMessage.SupportingDocsFileType, decodedMessage.SupportingDocsFileType)
	}

	// SupportingDocs (varbin)
	if !bytes.Equal(initialMessage.SupportingDocs, decodedMessage.SupportingDocs) {
		t.Errorf("SupportingDocs doesn't match : %x != %x", initialMessage.SupportingDocs, decodedMessage.SupportingDocs)
	}

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// ContractExpiration (Timestamp)
	// Timestamp test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// Issuer (Entity)
	// Entity test compare not setup

	// IssuerLogoURL (varchar)
	if initialMessage.IssuerLogoURL != decodedMessage.IssuerLogoURL {
		t.Errorf("IssuerLogoURL doesn't match : %s != %s", initialMessage.IssuerLogoURL, decodedMessage.IssuerLogoURL)
	}

	// ContractOperatorIncluded (bool)
	if initialMessage.ContractOperatorIncluded != decodedMessage.ContractOperatorIncluded {
		t.Errorf("ContractOperatorIncluded doesn't match : %v != %v", initialMessage.ContractOperatorIncluded, decodedMessage.ContractOperatorIncluded)
	}

	// ContractOperator (Entity)
	// Entity test compare not setup

	// ContractAuthFlags (bin)
	if initialMessage.ContractAuthFlags != decodedMessage.ContractAuthFlags {
		t.Errorf("ContractAuthFlags doesn't match : %v != %v", initialMessage.ContractAuthFlags, decodedMessage.ContractAuthFlags)
	}

	// ContractFee (uint)
	if initialMessage.ContractFee != decodedMessage.ContractFee {
		t.Errorf("ContractFee doesn't match : %v != %v", initialMessage.ContractFee, decodedMessage.ContractFee)
	}

	// VotingSystems (VotingSystem[])
	if len(initialMessage.VotingSystems) != len(decodedMessage.VotingSystems) {
		t.Errorf("VotingSystems lengths don't match : %d != %d", len(initialMessage.VotingSystems), len(decodedMessage.VotingSystems))
	}

	// RestrictedQtyAssets (uint)
	if initialMessage.RestrictedQtyAssets != decodedMessage.RestrictedQtyAssets {
		t.Errorf("RestrictedQtyAssets doesn't match : %v != %v", initialMessage.RestrictedQtyAssets, decodedMessage.RestrictedQtyAssets)
	}

	// ReferendumProposal (bool)
	if initialMessage.ReferendumProposal != decodedMessage.ReferendumProposal {
		t.Errorf("ReferendumProposal doesn't match : %v != %v", initialMessage.ReferendumProposal, decodedMessage.ReferendumProposal)
	}

	// InitiativeProposal (bool)
	if initialMessage.InitiativeProposal != decodedMessage.InitiativeProposal {
		t.Errorf("InitiativeProposal doesn't match : %v != %v", initialMessage.InitiativeProposal, decodedMessage.InitiativeProposal)
	}

	// Registries (Registry[])
	if len(initialMessage.Registries) != len(decodedMessage.Registries) {
		t.Errorf("Registries lengths don't match : %d != %d", len(initialMessage.Registries), len(decodedMessage.Registries))
	}
}

func TestContractFormation(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractFormation{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// BodyOfAgreementType (uint)
	// uint test not setup

	// BodyOfAgreement (varbin)
	initialMessage.BodyOfAgreement = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.BodyOfAgreement = append(initialMessage.BodyOfAgreement, byte(65+i+2))
	}

	// ContractType (varchar)
	initialMessage.ContractType = "Text 3"

	// SupportingDocsFileType (uint)
	// uint test not setup

	// SupportingDocs (varbin)
	initialMessage.SupportingDocs = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.SupportingDocs = append(initialMessage.SupportingDocs, byte(65+i+5))
	}

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+6))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+7))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// ContractExpiration (Timestamp)
	initialMessage.ContractExpiration = Timestamp{}

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 9"

	// Issuer (Entity)
	initialMessage.Issuer = Entity{}

	// IssuerLogoURL (varchar)
	initialMessage.IssuerLogoURL = "Text 11"

	// ContractOperatorIncluded (bool)
	initialMessage.ContractOperatorIncluded = true

	// ContractOperator (Entity)
	initialMessage.ContractOperator = Entity{}

	// ContractAuthFlags (bin)
	// bin test not setup

	// ContractFee (uint)
	// uint test not setup

	// VotingSystems (VotingSystem[])
	for i := 0; i < 2; i++ {
		initialMessage.VotingSystems = append(initialMessage.VotingSystems, VotingSystem{})
	}

	// RestrictedQtyAssets (uint)
	// uint test not setup

	// ReferendumProposal (bool)
	initialMessage.ReferendumProposal = true

	// InitiativeProposal (bool)
	initialMessage.InitiativeProposal = true

	// Registries (Registry[])
	for i := 0; i < 2; i++ {
		initialMessage.Registries = append(initialMessage.Registries, Registry{})
	}

	// ContractRevision (uint)
	// uint test not setup

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// BodyOfAgreementType (uint)
	if initialMessage.BodyOfAgreementType != decodedMessage.BodyOfAgreementType {
		t.Errorf("BodyOfAgreementType doesn't match : %v != %v", initialMessage.BodyOfAgreementType, decodedMessage.BodyOfAgreementType)
	}

	// BodyOfAgreement (varbin)
	if !bytes.Equal(initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement) {
		t.Errorf("BodyOfAgreement doesn't match : %x != %x", initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement)
	}

	// ContractType (varchar)
	if initialMessage.ContractType != decodedMessage.ContractType {
		t.Errorf("ContractType doesn't match : %s != %s", initialMessage.ContractType, decodedMessage.ContractType)
	}

	// SupportingDocsFileType (uint)
	if initialMessage.SupportingDocsFileType != decodedMessage.SupportingDocsFileType {
		t.Errorf("SupportingDocsFileType doesn't match : %v != %v", initialMessage.SupportingDocsFileType, decodedMessage.SupportingDocsFileType)
	}

	// SupportingDocs (varbin)
	if !bytes.Equal(initialMessage.SupportingDocs, decodedMessage.SupportingDocs) {
		t.Errorf("SupportingDocs doesn't match : %x != %x", initialMessage.SupportingDocs, decodedMessage.SupportingDocs)
	}

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// ContractExpiration (Timestamp)
	// Timestamp test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// Issuer (Entity)
	// Entity test compare not setup

	// IssuerLogoURL (varchar)
	if initialMessage.IssuerLogoURL != decodedMessage.IssuerLogoURL {
		t.Errorf("IssuerLogoURL doesn't match : %s != %s", initialMessage.IssuerLogoURL, decodedMessage.IssuerLogoURL)
	}

	// ContractOperatorIncluded (bool)
	if initialMessage.ContractOperatorIncluded != decodedMessage.ContractOperatorIncluded {
		t.Errorf("ContractOperatorIncluded doesn't match : %v != %v", initialMessage.ContractOperatorIncluded, decodedMessage.ContractOperatorIncluded)
	}

	// ContractOperator (Entity)
	// Entity test compare not setup

	// ContractAuthFlags (bin)
	if initialMessage.ContractAuthFlags != decodedMessage.ContractAuthFlags {
		t.Errorf("ContractAuthFlags doesn't match : %v != %v", initialMessage.ContractAuthFlags, decodedMessage.ContractAuthFlags)
	}

	// ContractFee (uint)
	if initialMessage.ContractFee != decodedMessage.ContractFee {
		t.Errorf("ContractFee doesn't match : %v != %v", initialMessage.ContractFee, decodedMessage.ContractFee)
	}

	// VotingSystems (VotingSystem[])
	if len(initialMessage.VotingSystems) != len(decodedMessage.VotingSystems) {
		t.Errorf("VotingSystems lengths don't match : %d != %d", len(initialMessage.VotingSystems), len(decodedMessage.VotingSystems))
	}

	// RestrictedQtyAssets (uint)
	if initialMessage.RestrictedQtyAssets != decodedMessage.RestrictedQtyAssets {
		t.Errorf("RestrictedQtyAssets doesn't match : %v != %v", initialMessage.RestrictedQtyAssets, decodedMessage.RestrictedQtyAssets)
	}

	// ReferendumProposal (bool)
	if initialMessage.ReferendumProposal != decodedMessage.ReferendumProposal {
		t.Errorf("ReferendumProposal doesn't match : %v != %v", initialMessage.ReferendumProposal, decodedMessage.ReferendumProposal)
	}

	// InitiativeProposal (bool)
	if initialMessage.InitiativeProposal != decodedMessage.InitiativeProposal {
		t.Errorf("InitiativeProposal doesn't match : %v != %v", initialMessage.InitiativeProposal, decodedMessage.InitiativeProposal)
	}

	// Registries (Registry[])
	if len(initialMessage.Registries) != len(decodedMessage.Registries) {
		t.Errorf("Registries lengths don't match : %d != %d", len(initialMessage.Registries), len(decodedMessage.Registries))
	}

	// ContractRevision (uint)
	if initialMessage.ContractRevision != decodedMessage.ContractRevision {
		t.Errorf("ContractRevision doesn't match : %v != %v", initialMessage.ContractRevision, decodedMessage.ContractRevision)
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestContractAmendment(t *testing.T) {
	// Create a randomized object
	initialMessage := ContractAmendment{}
	// ChangeIssuerAddress (bool)
	initialMessage.ChangeIssuerAddress = true

	// ChangeOperatorAddress (bool)
	initialMessage.ChangeOperatorAddress = true

	// ContractRevision (uint)
	// uint test not setup

	// Amendments (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.Amendments = append(initialMessage.Amendments, Amendment{})
	}

	// RefTxID (TxId)
	initialMessage.RefTxID = TxId{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ChangeIssuerAddress (bool)
	if initialMessage.ChangeIssuerAddress != decodedMessage.ChangeIssuerAddress {
		t.Errorf("ChangeIssuerAddress doesn't match : %v != %v", initialMessage.ChangeIssuerAddress, decodedMessage.ChangeIssuerAddress)
	}

	// ChangeOperatorAddress (bool)
	if initialMessage.ChangeOperatorAddress != decodedMessage.ChangeOperatorAddress {
		t.Errorf("ChangeOperatorAddress doesn't match : %v != %v", initialMessage.ChangeOperatorAddress, decodedMessage.ChangeOperatorAddress)
	}

	// ContractRevision (uint)
	if initialMessage.ContractRevision != decodedMessage.ContractRevision {
		t.Errorf("ContractRevision doesn't match : %v != %v", initialMessage.ContractRevision, decodedMessage.ContractRevision)
	}

	// Amendments (Amendment[])
	if len(initialMessage.Amendments) != len(decodedMessage.Amendments) {
		t.Errorf("Amendments lengths don't match : %d != %d", len(initialMessage.Amendments), len(decodedMessage.Amendments))
	}

	// RefTxID (TxId)
	// TxId test compare not setup
}

func TestStaticContractFormation(t *testing.T) {
	// Create a randomized object
	initialMessage := StaticContractFormation{}
	// ContractName (varchar)
	initialMessage.ContractName = "Text 0"

	// ContractCode (ContractCode)
	initialMessage.ContractCode = ContractCode{}

	// BodyOfAgreementType (uint)
	// uint test not setup

	// BodyOfAgreement (varbin)
	initialMessage.BodyOfAgreement = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.BodyOfAgreement = append(initialMessage.BodyOfAgreement, byte(65+i+3))
	}

	// ContractType (varchar)
	initialMessage.ContractType = "Text 4"

	// SupportingDocsFileType (uint)
	// uint test not setup

	// SupportingDocs (varchar)
	initialMessage.SupportingDocs = "Text 6"

	// ContractRevision (uint)
	// uint test not setup

	// GoverningLaw (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+8))
		}
		initialMessage.GoverningLaw = string(text)
	}

	// Jurisdiction (fixedchar)
	{
		text := make([]byte, 0, 5)
		for i := uint64(0); i < 5; i++ {
			text = append(text, byte(65+i+9))
		}
		initialMessage.Jurisdiction = string(text)
	}

	// EffectiveDate (Timestamp)
	initialMessage.EffectiveDate = Timestamp{}

	// ContractExpiration (Timestamp)
	initialMessage.ContractExpiration = Timestamp{}

	// ContractURI (varchar)
	initialMessage.ContractURI = "Text 12"

	// PrevRevTxID (TxId)
	initialMessage.PrevRevTxID = TxId{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// ContractName (varchar)
	if initialMessage.ContractName != decodedMessage.ContractName {
		t.Errorf("ContractName doesn't match : %s != %s", initialMessage.ContractName, decodedMessage.ContractName)
	}

	// ContractCode (ContractCode)
	// ContractCode test compare not setup

	// BodyOfAgreementType (uint)
	if initialMessage.BodyOfAgreementType != decodedMessage.BodyOfAgreementType {
		t.Errorf("BodyOfAgreementType doesn't match : %v != %v", initialMessage.BodyOfAgreementType, decodedMessage.BodyOfAgreementType)
	}

	// BodyOfAgreement (varbin)
	if !bytes.Equal(initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement) {
		t.Errorf("BodyOfAgreement doesn't match : %x != %x", initialMessage.BodyOfAgreement, decodedMessage.BodyOfAgreement)
	}

	// ContractType (varchar)
	if initialMessage.ContractType != decodedMessage.ContractType {
		t.Errorf("ContractType doesn't match : %s != %s", initialMessage.ContractType, decodedMessage.ContractType)
	}

	// SupportingDocsFileType (uint)
	if initialMessage.SupportingDocsFileType != decodedMessage.SupportingDocsFileType {
		t.Errorf("SupportingDocsFileType doesn't match : %v != %v", initialMessage.SupportingDocsFileType, decodedMessage.SupportingDocsFileType)
	}

	// SupportingDocs (varchar)
	if initialMessage.SupportingDocs != decodedMessage.SupportingDocs {
		t.Errorf("SupportingDocs doesn't match : %s != %s", initialMessage.SupportingDocs, decodedMessage.SupportingDocs)
	}

	// ContractRevision (uint)
	if initialMessage.ContractRevision != decodedMessage.ContractRevision {
		t.Errorf("ContractRevision doesn't match : %v != %v", initialMessage.ContractRevision, decodedMessage.ContractRevision)
	}

	// GoverningLaw (fixedchar)
	if initialMessage.GoverningLaw != decodedMessage.GoverningLaw {
		t.Errorf("GoverningLaw doesn't match : %s != %s", initialMessage.GoverningLaw, decodedMessage.GoverningLaw)
	}

	// Jurisdiction (fixedchar)
	if initialMessage.Jurisdiction != decodedMessage.Jurisdiction {
		t.Errorf("Jurisdiction doesn't match : %s != %s", initialMessage.Jurisdiction, decodedMessage.Jurisdiction)
	}

	// EffectiveDate (Timestamp)
	// Timestamp test compare not setup

	// ContractExpiration (Timestamp)
	// Timestamp test compare not setup

	// ContractURI (varchar)
	if initialMessage.ContractURI != decodedMessage.ContractURI {
		t.Errorf("ContractURI doesn't match : %s != %s", initialMessage.ContractURI, decodedMessage.ContractURI)
	}

	// PrevRevTxID (TxId)
	// TxId test compare not setup

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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// ComplianceAction (fixedchar)
	// fixedchar test not setup

	// TargetAddresses (TargetAddress[])
	for i := 0; i < 2; i++ {
		initialMessage.TargetAddresses = append(initialMessage.TargetAddresses, TargetAddress{})
	}

	// DepositAddress (PublicKeyHash)
	initialMessage.DepositAddress = PublicKeyHash{}

	// AuthorityName (varchar)
	initialMessage.AuthorityName = "Text 5"

	// SigAlgoAddressList (uint)
	// uint test not setup

	// AuthorityPublicKey (varchar)
	initialMessage.AuthorityPublicKey = "Text 7"

	// OrderSignature (varchar)
	initialMessage.OrderSignature = "Text 8"

	// SupportingEvidenceTxId (TxId)
	initialMessage.SupportingEvidenceTxId = TxId{}

	// RefTxns (TxId)
	initialMessage.RefTxns = TxId{}

	// FreezePeriod (Timestamp)
	initialMessage.FreezePeriod = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// ComplianceAction (fixedchar)
	if initialMessage.ComplianceAction != decodedMessage.ComplianceAction {
		t.Errorf("ComplianceAction doesn't match : %v != %v", initialMessage.ComplianceAction, decodedMessage.ComplianceAction)
	}

	// TargetAddresses (TargetAddress[])
	if len(initialMessage.TargetAddresses) != len(decodedMessage.TargetAddresses) {
		t.Errorf("TargetAddresses lengths don't match : %d != %d", len(initialMessage.TargetAddresses), len(decodedMessage.TargetAddresses))
	}

	// DepositAddress (PublicKeyHash)
	// PublicKeyHash test compare not setup

	// AuthorityName (varchar)
	if initialMessage.AuthorityName != decodedMessage.AuthorityName {
		t.Errorf("AuthorityName doesn't match : %s != %s", initialMessage.AuthorityName, decodedMessage.AuthorityName)
	}

	// SigAlgoAddressList (uint)
	if initialMessage.SigAlgoAddressList != decodedMessage.SigAlgoAddressList {
		t.Errorf("SigAlgoAddressList doesn't match : %v != %v", initialMessage.SigAlgoAddressList, decodedMessage.SigAlgoAddressList)
	}

	// AuthorityPublicKey (varchar)
	if initialMessage.AuthorityPublicKey != decodedMessage.AuthorityPublicKey {
		t.Errorf("AuthorityPublicKey doesn't match : %s != %s", initialMessage.AuthorityPublicKey, decodedMessage.AuthorityPublicKey)
	}

	// OrderSignature (varchar)
	if initialMessage.OrderSignature != decodedMessage.OrderSignature {
		t.Errorf("OrderSignature doesn't match : %s != %s", initialMessage.OrderSignature, decodedMessage.OrderSignature)
	}

	// SupportingEvidenceTxId (TxId)
	// TxId test compare not setup

	// RefTxns (TxId)
	// TxId test compare not setup

	// FreezePeriod (Timestamp)
	// Timestamp test compare not setup

	// Message (varchar)
	if initialMessage.Message != decodedMessage.Message {
		t.Errorf("Message doesn't match : %s != %s", initialMessage.Message, decodedMessage.Message)
	}
}

func TestFreeze(t *testing.T) {
	// Create a randomized object
	initialMessage := Freeze{}
	// Addresses (PublicKeyHash[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, PublicKeyHash{})
	}

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (PublicKeyHash[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestThaw(t *testing.T) {
	// Create a randomized object
	initialMessage := Thaw{}
	// Addresses (PublicKeyHash[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, PublicKeyHash{})
	}

	// RefTxID (TxId)
	initialMessage.RefTxID = TxId{}

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (PublicKeyHash[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// RefTxID (TxId)
	// TxId test compare not setup

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestConfiscation(t *testing.T) {
	// Create a randomized object
	initialMessage := Confiscation{}
	// Addresses (PublicKeyHash[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, PublicKeyHash{})
	}

	// DepositQty (uint)
	// uint test not setup

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (PublicKeyHash[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// DepositQty (uint)
	if initialMessage.DepositQty != decodedMessage.DepositQty {
		t.Errorf("DepositQty doesn't match : %v != %v", initialMessage.DepositQty, decodedMessage.DepositQty)
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestReconciliation(t *testing.T) {
	// Create a randomized object
	initialMessage := Reconciliation{}
	// Addresses (PublicKeyHash[])
	for i := 0; i < 2; i++ {
		initialMessage.Addresses = append(initialMessage.Addresses, PublicKeyHash{})
	}

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Addresses (PublicKeyHash[])
	if len(initialMessage.Addresses) != len(decodedMessage.Addresses) {
		t.Errorf("Addresses lengths don't match : %d != %d", len(initialMessage.Addresses), len(decodedMessage.Addresses))
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// VoteSystem (uint)
	// uint test not setup

	// Proposal (bool)
	initialMessage.Proposal = true

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

	// ProposalDocumentHash (bin)
	// bin test not setup

	// VoteCutOffTimestamp (Timestamp)
	initialMessage.VoteCutOffTimestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// VoteSystem (uint)
	if initialMessage.VoteSystem != decodedMessage.VoteSystem {
		t.Errorf("VoteSystem doesn't match : %v != %v", initialMessage.VoteSystem, decodedMessage.VoteSystem)
	}

	// Proposal (bool)
	if initialMessage.Proposal != decodedMessage.Proposal {
		t.Errorf("Proposal doesn't match : %v != %v", initialMessage.Proposal, decodedMessage.Proposal)
	}

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteOptions (varchar)
	if initialMessage.VoteOptions != decodedMessage.VoteOptions {
		t.Errorf("VoteOptions doesn't match : %s != %s", initialMessage.VoteOptions, decodedMessage.VoteOptions)
	}

	// VoteMax (uint)
	if initialMessage.VoteMax != decodedMessage.VoteMax {
		t.Errorf("VoteMax doesn't match : %v != %v", initialMessage.VoteMax, decodedMessage.VoteMax)
	}

	// ProposalDescription (varchar)
	if initialMessage.ProposalDescription != decodedMessage.ProposalDescription {
		t.Errorf("ProposalDescription doesn't match : %s != %s", initialMessage.ProposalDescription, decodedMessage.ProposalDescription)
	}

	// ProposalDocumentHash (bin)
	if initialMessage.ProposalDocumentHash != decodedMessage.ProposalDocumentHash {
		t.Errorf("ProposalDocumentHash doesn't match : %v != %v", initialMessage.ProposalDocumentHash, decodedMessage.ProposalDocumentHash)
	}

	// VoteCutOffTimestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestReferendum(t *testing.T) {
	// Create a randomized object
	initialMessage := Referendum{}
	// AssetSpecificVote (bool)
	initialMessage.AssetSpecificVote = true

	// AssetType (fixedchar)
	{
		text := make([]byte, 0, 3)
		for i := uint64(0); i < 3; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.AssetType = string(text)
	}

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// VoteSystem (uint)
	// uint test not setup

	// Proposal (bool)
	initialMessage.Proposal = true

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

	// ProposalDocumentHash (bin)
	// bin test not setup

	// VoteCutOffTimestamp (Timestamp)
	initialMessage.VoteCutOffTimestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetSpecificVote (bool)
	if initialMessage.AssetSpecificVote != decodedMessage.AssetSpecificVote {
		t.Errorf("AssetSpecificVote doesn't match : %v != %v", initialMessage.AssetSpecificVote, decodedMessage.AssetSpecificVote)
	}

	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// VoteSystem (uint)
	if initialMessage.VoteSystem != decodedMessage.VoteSystem {
		t.Errorf("VoteSystem doesn't match : %v != %v", initialMessage.VoteSystem, decodedMessage.VoteSystem)
	}

	// Proposal (bool)
	if initialMessage.Proposal != decodedMessage.Proposal {
		t.Errorf("Proposal doesn't match : %v != %v", initialMessage.Proposal, decodedMessage.Proposal)
	}

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteOptions (varchar)
	if initialMessage.VoteOptions != decodedMessage.VoteOptions {
		t.Errorf("VoteOptions doesn't match : %s != %s", initialMessage.VoteOptions, decodedMessage.VoteOptions)
	}

	// VoteMax (uint)
	if initialMessage.VoteMax != decodedMessage.VoteMax {
		t.Errorf("VoteMax doesn't match : %v != %v", initialMessage.VoteMax, decodedMessage.VoteMax)
	}

	// ProposalDescription (varchar)
	if initialMessage.ProposalDescription != decodedMessage.ProposalDescription {
		t.Errorf("ProposalDescription doesn't match : %s != %s", initialMessage.ProposalDescription, decodedMessage.ProposalDescription)
	}

	// ProposalDocumentHash (bin)
	if initialMessage.ProposalDocumentHash != decodedMessage.ProposalDocumentHash {
		t.Errorf("ProposalDocumentHash doesn't match : %v != %v", initialMessage.ProposalDocumentHash, decodedMessage.ProposalDocumentHash)
	}

	// VoteCutOffTimestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestVote(t *testing.T) {
	// Create a randomized object
	initialMessage := Vote{}
	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Timestamp (Timestamp)
	// Timestamp test compare not setup
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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// VoteTxID (TxId)
	initialMessage.VoteTxID = TxId{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// VoteTxID (TxId)
	// TxId test compare not setup

	// Vote (varchar)
	if initialMessage.Vote != decodedMessage.Vote {
		t.Errorf("Vote doesn't match : %s != %s", initialMessage.Vote, decodedMessage.Vote)
	}
}

func TestBallotCounted(t *testing.T) {
	// Create a randomized object
	initialMessage := BallotCounted{}
	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Timestamp (Timestamp)
	// Timestamp test compare not setup
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

	// AssetCode (AssetCode)
	initialMessage.AssetCode = AssetCode{}

	// Proposal (bool)
	initialMessage.Proposal = true

	// ProposedChanges (Amendment[])
	for i := 0; i < 2; i++ {
		initialMessage.ProposedChanges = append(initialMessage.ProposedChanges, Amendment{})
	}

	// VoteTxID (TxId)
	initialMessage.VoteTxID = TxId{}

	// VoteOptionsCount (uint)
	// uint test not setup

	// OptionXTally (uint)
	// uint test not setup

	// Result (varchar)
	initialMessage.Result = "Text 7"

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// AssetType (fixedchar)
	if initialMessage.AssetType != decodedMessage.AssetType {
		t.Errorf("AssetType doesn't match : %s != %s", initialMessage.AssetType, decodedMessage.AssetType)
	}

	// AssetCode (AssetCode)
	// AssetCode test compare not setup

	// Proposal (bool)
	if initialMessage.Proposal != decodedMessage.Proposal {
		t.Errorf("Proposal doesn't match : %v != %v", initialMessage.Proposal, decodedMessage.Proposal)
	}

	// ProposedChanges (Amendment[])
	if len(initialMessage.ProposedChanges) != len(decodedMessage.ProposedChanges) {
		t.Errorf("ProposedChanges lengths don't match : %d != %d", len(initialMessage.ProposedChanges), len(decodedMessage.ProposedChanges))
	}

	// VoteTxID (TxId)
	// TxId test compare not setup

	// VoteOptionsCount (uint)
	if initialMessage.VoteOptionsCount != decodedMessage.VoteOptionsCount {
		t.Errorf("VoteOptionsCount doesn't match : %v != %v", initialMessage.VoteOptionsCount, decodedMessage.VoteOptionsCount)
	}

	// OptionXTally (uint)
	if initialMessage.OptionXTally != decodedMessage.OptionXTally {
		t.Errorf("OptionXTally doesn't match : %v != %v", initialMessage.OptionXTally, decodedMessage.OptionXTally)
	}

	// Result (varchar)
	if initialMessage.Result != decodedMessage.Result {
		t.Errorf("Result doesn't match : %s != %s", initialMessage.Result, decodedMessage.Result)
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}

func TestMessage(t *testing.T) {
	// Create a randomized object
	initialMessage := Message{}
	// AddressIndexes (uint16[])
	for i := 0; i < 5; i++ {
		var item uint16
		initialMessage.AddressIndexes = append(initialMessage.AddressIndexes, item)
	}

	// MessageType (fixedchar)
	{
		text := make([]byte, 0, 4)
		for i := uint64(0); i < 4; i++ {
			text = append(text, byte(65+i+1))
		}
		initialMessage.MessageType = string(text)
	}

	// MessagePayload (varbin)
	initialMessage.MessagePayload = make([]byte, 0, 32)
	for i := uint64(0); i < 32; i++ {
		initialMessage.MessagePayload = append(initialMessage.MessagePayload, byte(65+i+2))
	}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
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

	// MessagePayload (varbin)
	if !bytes.Equal(initialMessage.MessagePayload, decodedMessage.MessagePayload) {
		t.Errorf("MessagePayload doesn't match : %x != %x", initialMessage.MessagePayload, decodedMessage.MessagePayload)
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

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// QtyReceivingAddresses (uint8)
	if initialMessage.QtyReceivingAddresses != decodedMessage.QtyReceivingAddresses {
		t.Errorf("QtyReceivingAddresses doesn't match : %v != %v", initialMessage.QtyReceivingAddresses, decodedMessage.QtyReceivingAddresses)
	}

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
	if initialMessage.RejectionType != decodedMessage.RejectionType {
		t.Errorf("RejectionType doesn't match : %v != %v", initialMessage.RejectionType, decodedMessage.RejectionType)
	}

	// MessagePayload (varchar)
	if initialMessage.MessagePayload != decodedMessage.MessagePayload {
		t.Errorf("MessagePayload doesn't match : %s != %s", initialMessage.MessagePayload, decodedMessage.MessagePayload)
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
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

	// OfferExpiry (Timestamp)
	initialMessage.OfferExpiry = Timestamp{}

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

	// ExchangeFeeAddress (PublicKeyHash)
	initialMessage.ExchangeFeeAddress = PublicKeyHash{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Assets (AssetTransfer[])
	if len(initialMessage.Assets) != len(decodedMessage.Assets) {
		t.Errorf("Assets lengths don't match : %d != %d", len(initialMessage.Assets), len(decodedMessage.Assets))
	}

	// OfferExpiry (Timestamp)
	// Timestamp test compare not setup

	// ExchangeFeeCurrency (fixedchar)
	if initialMessage.ExchangeFeeCurrency != decodedMessage.ExchangeFeeCurrency {
		t.Errorf("ExchangeFeeCurrency doesn't match : %s != %s", initialMessage.ExchangeFeeCurrency, decodedMessage.ExchangeFeeCurrency)
	}

	// ExchangeFeeVar (float)
	if initialMessage.ExchangeFeeVar != decodedMessage.ExchangeFeeVar {
		t.Errorf("ExchangeFeeVar doesn't match : %v != %v", initialMessage.ExchangeFeeVar, decodedMessage.ExchangeFeeVar)
	}

	// ExchangeFeeFixed (float)
	if initialMessage.ExchangeFeeFixed != decodedMessage.ExchangeFeeFixed {
		t.Errorf("ExchangeFeeFixed doesn't match : %v != %v", initialMessage.ExchangeFeeFixed, decodedMessage.ExchangeFeeFixed)
	}

	// ExchangeFeeAddress (PublicKeyHash)
	// PublicKeyHash test compare not setup
}

func TestSettlement(t *testing.T) {
	// Create a randomized object
	initialMessage := Settlement{}
	// Assets (AssetSettlement[])
	for i := 0; i < 2; i++ {
		initialMessage.Assets = append(initialMessage.Assets, AssetSettlement{})
	}

	// Timestamp (Timestamp)
	initialMessage.Timestamp = Timestamp{}

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

	if !bytes.Equal(initialEncoding, secondEncoding) {
		t.Errorf("Encoded value doesn't match.\ngot\n%+v\nwant\n%+v", initialEncoding, secondEncoding)
	}

	// if !cmp.Equal(&initialMessage, &decodedMessage) {
	// 	t.Errorf("Decoded value doesn't match.\ninitial : %+v\ndecoded : %+v", initialMessage, decodedMessage)
	// }

	// Compare re-serialized values
	// Assets (AssetSettlement[])
	if len(initialMessage.Assets) != len(decodedMessage.Assets) {
		t.Errorf("Assets lengths don't match : %d != %d", len(initialMessage.Assets), len(decodedMessage.Assets))
	}

	// Timestamp (Timestamp)
	// Timestamp test compare not setup
}
