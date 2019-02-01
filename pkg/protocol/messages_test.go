package protocol

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

func hexToBytes(s string) []byte {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return []byte(decoded)
}

func TestAssetDefinition_NewAssetDefinition(t *testing.T) {
	cf := NewAssetDefinition()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeAssetDefinition)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeAssetDefinition)
	}
}

func TestAssetDefinition_Read(t *testing.T) {
	m := NewAssetDefinition()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("A1")
	m.Version = 0
	m.AssetType = []byte("COU")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.AuthorizationFlags = []byte{0x11, 0xbf}
	m.VotingSystem = byte('M')
	m.VoteMultiplier = 3
	m.Qty = 1000000.00
	m.ContractFeeCurrency = []byte("AUD")
	m.ContractFeeVar = 0.0050
	m.ContractFeeFixed = 0.01
	m.Payload = hexToBytes("00415553576f6f6c776f72746873000000000000000000000000000000000000000000000000017655f6b4000000015195342000496e74276c2c20353025206f666620326e64204974656d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	want := "0x6a4cd200000020413100434f5561706d3271737a6e686b7332337a38643833753431733830313968797269336911bf4d0300000000000f42404155443ba3d70a3c23d70a00415553576f6f6c776f72746873000000000000000000000000000000000000000000000000017655f6b4000000015195342000496e74276c2c20353025206f666620326e64204974656d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 213 {
		t.Errorf("got %v, want %v\n", n, 213)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAssetDefinition_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cd200000020413100434f5561706d3271737a6e686b7332337a38643833753431733830313968797269336911bf4d0300000000000f42404155443ba3d70a3c23d70a00415553576f6f6c776f72746873000000000000000000000000000000000000000000000000017655f6b4000000015195342000496e74276c2c20353025206f666620326e64204974656d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := AssetDefinition{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 213 {
		t.Fatalf("got %v, want %v", len(payload), 213)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 213 {
		t.Errorf("got %v, want %v", n, 213)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAssetCreation_NewAssetCreation(t *testing.T) {
	cf := NewAssetCreation()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeAssetCreation)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeAssetCreation)
	}
}

func TestAssetCreation_Read(t *testing.T) {
	m := NewAssetCreation()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("A2")
	m.Version = 0
	m.AssetType = []byte("SHC")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.AssetRevision = 0
	m.Timestamp = 0x0000016331e3c200
	m.AuthorizationFlags = []byte{0x11, 0xbf}
	m.VotingSystem = byte('M')
	m.VoteMultiplier = 3
	m.Qty = 1000000.00
	m.ContractFeeCurrency = []byte("AUD")
	m.ContractFeeVar = 0.0050
	m.ContractFeeFixed = 0.01
	m.Payload = hexToBytes("00415553433ca3d70a461c400051594d534600005553303030343032363235304150504c43202d204170706c6520636f7270202d20436f6d6d6f6e20736861726520697373756520636c6173732043000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	want := "0x6a4cdc0000002041320053484361706d3271737a6e686b7332337a38643833753431733830313968797269336900000000016331e3c20011bf4d0300000000000f42404155443ba3d70a3c23d70a00415553433ca3d70a461c400051594d534600005553303030343032363235304150504c43202d204170706c6520636f7270202d20436f6d6d6f6e20736861726520697373756520636c6173732043000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAssetCreation_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc0000002041320053484361706d3271737a6e686b7332337a38643833753431733830313968797269336900000000016331e3c20011bf4d0300000000000f42404155443ba3d70a3c23d70a00415553433ca3d70a461c400051594d534600005553303030343032363235304150504c43202d204170706c6520636f7270202d20436f6d6d6f6e20736861726520697373756520636c6173732043000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := AssetCreation{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAssetModification_NewAssetModification(t *testing.T) {
	cf := NewAssetModification()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeAssetModification)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeAssetModification)
	}
}

func TestAssetModification_Read(t *testing.T) {
	m := NewAssetModification()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("A3")
	m.Version = 0
	m.AssetType = []byte("TIC")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.AssetRevision = 0
	m.AuthorizationFlags = []byte{0x11, 0xbf}
	m.VotingSystem = byte('M')
	m.VoteMultiplier = 3
	m.Qty = 1000000.00
	m.ContractFeeCurrency = []byte("AUD")
	m.ContractFeeVar = 0.0050
	m.ContractFeeFixed = 0.01
	m.Payload = hexToBytes("0031324731382b000000000166e0f4518000000166f58dc180436f696e6765656b20436f6e666572656e6365202d204c6f6e646f6e20284e6f76656d6265722032303138292e20526567756c61722061646d697373696f6e2e2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	want := "0x6a4cd40000002041330054494361706d3271737a6e686b7332337a386438337534317338303139687972693369000011bf4d0300000000000f42404155443ba3d70a3c23d70a0031324731382b000000000166e0f4518000000166f58dc180436f696e6765656b20436f6e666572656e6365202d204c6f6e646f6e20284e6f76656d6265722032303138292e20526567756c61722061646d697373696f6e2e2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 215 {
		t.Errorf("got %v, want %v\n", n, 215)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAssetModification_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cd40000002041330054494361706d3271737a6e686b7332337a386438337534317338303139687972693369000011bf4d0300000000000f42404155443ba3d70a3c23d70a0031324731382b000000000166e0f4518000000166f58dc180436f696e6765656b20436f6e666572656e6365202d204c6f6e646f6e20284e6f76656d6265722032303138292e20526567756c61722061646d697373696f6e2e2000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := AssetModification{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 215 {
		t.Fatalf("got %v, want %v", len(payload), 215)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 215 {
		t.Errorf("got %v, want %v", n, 215)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractOffer_NewContractOffer(t *testing.T) {
	cf := NewContractOffer()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeContractOffer)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeContractOffer)
	}
}

func TestContractOffer_Read(t *testing.T) {
	m := NewContractOffer()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("C1")
	m.Version = 0
	m.ContractName = []byte("Tesla - Shareholder Agreement")
	m.ContractFileHash = hexToBytes("c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1")
	m.GoverningLaw = []byte("USA")
	m.Jurisdiction = []byte("US-CA")
	m.ContractExpiration = 0x00000163400d3f00
	m.URI = []byte("https://tokenized.cash/Contract/3qeoSCg7JmfSnJesJFojj")
	m.IssuerID = []byte("Tesla, Inc")
	m.IssuerType = byte('P')
	m.ContractOperatorID = []byte("Tokenized")
	m.AuthorizationFlags = []byte{0x11, 0xfd}
	m.VotingSystem = byte('M')
	m.InitiativeThreshold = 100.00
	m.InitiativeThresholdCurrency = []byte("AUD")
	m.RestrictedQty = 1

	want := "0x6a4cd2000000204331005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d434100000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a00000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c800004155440000000000000001"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 213 {
		t.Errorf("got %v, want %v\n", n, 213)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractOffer_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cd2000000204331005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d434100000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a00000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c800004155440000000000000001"
	cf := ContractOffer{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 213 {
		t.Fatalf("got %v, want %v", len(payload), 213)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 213 {
		t.Errorf("got %v, want %v", n, 213)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractFormation_NewContractFormation(t *testing.T) {
	cf := NewContractFormation()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeContractFormation)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeContractFormation)
	}
}

func TestContractFormation_Read(t *testing.T) {
	m := NewContractFormation()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("C2")
	m.Version = 0
	m.ContractName = []byte("Tesla - Shareholder Agreement")
	m.ContractFileHash = hexToBytes("c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1")
	m.GoverningLaw = []byte("USA")
	m.Jurisdiction = []byte("US-CA")
	m.Timestamp = 0x0000016331e3c200
	m.ContractExpiration = 0x00000163400d3f00
	m.URI = []byte("https://tokenized.cash/Contract/3qeoSCg7JmfSnJesJFojj")
	m.ContractRevision = 0
	m.IssuerID = []byte("Tesla, Inc")
	m.IssuerType = byte('P')
	m.ContractOperatorID = []byte("Tokenized")
	m.AuthorizationFlags = []byte{0x11, 0xfd}
	m.VotingSystem = byte('M')
	m.InitiativeThreshold = 100.00
	m.InitiativeThresholdCurrency = []byte("AUD")
	m.RestrictedQty = 1

	want := "0x6a4cdc000000204332005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d43410000016331e3c20000000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a000000000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c800004155440000000000000001"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractFormation_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc000000204332005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d43410000016331e3c20000000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a000000000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c800004155440000000000000001"
	cf := ContractFormation{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractAmendment_NewContractAmendment(t *testing.T) {
	cf := NewContractAmendment()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeContractAmendment)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeContractAmendment)
	}
}

func TestContractAmendment_Read(t *testing.T) {
	m := NewContractAmendment()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("C3")
	m.Version = 0
	m.ContractName = []byte("Tesla - Shareholder Agreement")
	m.ContractFileHash = hexToBytes("c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1")
	m.GoverningLaw = []byte("USA")
	m.Jurisdiction = []byte("US-CA")
	m.ContractExpiration = 0x00000163400d3f00
	m.URI = []byte("https://tokenized.cash/Contract/3qeoSCg7JmfSnJesJFojj")
	m.ContractRevision = 0
	m.IssuerID = []byte("Tesla, Inc")
	m.IssuerType = byte('P')
	m.ContractOperatorID = []byte("Tokenized")
	m.AuthorizationFlags = []byte{0x11, 0xfd}
	m.VotingSystem = byte('M')
	m.InitiativeThreshold = 100.10
	m.InitiativeThresholdCurrency = []byte("AUD")
	m.RestrictedQty = 1

	want := "0x6a4cd4000000204333005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d434100000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a000000000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c833334155440000000000000001"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 215 {
		t.Errorf("got %v, want %v\n", n, 215)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestContractAmendment_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cd4000000204333005465736c61202d205368617265686f6c6465722041677265656d656e74000000c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1555341000055532d434100000163400d3f0068747470733a2f2f746f6b656e697a65642e636173682f436f6e74726163742f3371656f534367374a6d66536e4a65734a466f6a6a000000000000000000000000000000000000005465736c612c20496e6300000000000050546f6b656e697a65640000000000000011fd4d42c833334155440000000000000001"
	cf := ContractAmendment{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 215 {
		t.Fatalf("got %v, want %v", len(payload), 215)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 215 {
		t.Errorf("got %v, want %v", n, 215)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestOrder_NewOrder(t *testing.T) {
	cf := NewOrder()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeOrder)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeOrder)
	}
}

func TestOrder_Read(t *testing.T) {
	m := NewOrder()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("E1")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.ComplianceAction = byte('F')
	m.TargetAddress = []byte("1HQ2ULuD7T5ykaucZ3KmTo4i29925Qa6ic")
	m.DepositAddress = []byte("17zAWabipcUHn5XP9w8GEc3PKvG5bYGBMe")
	m.SupportingEvidenceHash = hexToBytes("c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1")
	m.Qty = 10000
	m.Expiration = 0x00000166550ce380
	m.Message = []byte("Sorry, but the court order made me.")

	want := "0x6a4cdc0000002045310052524561706d3271737a6e686b7332337a3864383375343173383031396879726933694631485132554c7544375435796b6175635a334b6d546f34693239393235516136696331377a4157616269706355486e35585039773847456333504b764735625947424d65c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1000000000000271000000166550ce380536f7272792c206275742074686520636f757274206f72646572206d616465206d652e0000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestOrder_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc0000002045310052524561706d3271737a6e686b7332337a3864383375343173383031396879726933694631485132554c7544375435796b6175635a334b6d546f34693239393235516136696331377a4157616269706355486e35585039773847456333504b764735625947424d65c236f77c7abd7249489e7d2bb6c7e46ba3f4095956e78a584af753ece56cf6d1000000000000271000000166550ce380536f7272792c206275742074686520636f757274206f72646572206d616465206d652e0000000000000000000000000000000000000000000000000000"
	cf := Order{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestFreeze_NewFreeze(t *testing.T) {
	cf := NewFreeze()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeFreeze)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeFreeze)
	}
}

func TestFreeze_Read(t *testing.T) {
	m := NewFreeze()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("E2")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.Timestamp = 0x0000016331e3c200
	m.Qty = 10000
	m.Expiration = 0x00000166550ce380
	m.Message = []byte("Sorry, the police made me do it.")

	want := "0x6a4c7f0000002045320052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c200000000000000271000000166550ce380536f7272792c2074686520706f6c696365206d616465206d6520646f2069742e0000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 130 {
		t.Errorf("got %v, want %v\n", n, 130)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestFreeze_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c7f0000002045320052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c200000000000000271000000166550ce380536f7272792c2074686520706f6c696365206d616465206d6520646f2069742e0000000000000000000000000000000000000000000000000000000000"
	cf := Freeze{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 130 {
		t.Fatalf("got %v, want %v", len(payload), 130)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 130 {
		t.Errorf("got %v, want %v", n, 130)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestThaw_NewThaw(t *testing.T) {
	cf := NewThaw()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeThaw)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeThaw)
	}
}

func TestThaw_Read(t *testing.T) {
	m := NewThaw()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("E3")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.Timestamp = 0x0000016331e3c200
	m.Qty = 10000
	m.Message = []byte("Tokens now unfrozen. Apologies for the mixup.")

	want := "0x6a4c770000002045330052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c2000000000000002710546f6b656e73206e6f7720756e66726f7a656e2e2041706f6c6f6769657320666f7220746865206d697875702e00000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 122 {
		t.Errorf("got %v, want %v\n", n, 122)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestThaw_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c770000002045330052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c2000000000000002710546f6b656e73206e6f7720756e66726f7a656e2e2041706f6c6f6769657320666f7220746865206d697875702e00000000000000000000000000000000"
	cf := Thaw{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 122 {
		t.Fatalf("got %v, want %v", len(payload), 122)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 122 {
		t.Errorf("got %v, want %v", n, 122)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestConfiscation_NewConfiscation(t *testing.T) {
	cf := NewConfiscation()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeConfiscation)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeConfiscation)
	}
}

func TestConfiscation_Read(t *testing.T) {
	m := NewConfiscation()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("E4")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.Timestamp = 0x0000016331e3c200
	m.TargetsQty = 0
	m.DepositsQty = 10000
	m.Message = []byte("Sorry bud, we have to take your tokens. Don't steal next time")

	want := "0x6a4c7f0000002045340052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c20000000000000000000000000000002710536f727279206275642c207765206861766520746f2074616b6520796f757220746f6b656e732e20446f6e277420737465616c206e6578742074696d65"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 130 {
		t.Errorf("got %v, want %v\n", n, 130)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestConfiscation_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c7f0000002045340052524561706d3271737a6e686b7332337a3864383375343173383031396879726933690000016331e3c20000000000000000000000000000002710536f727279206275642c207765206861766520746f2074616b6520796f757220746f6b656e732e20446f6e277420737465616c206e6578742074696d65"
	cf := Confiscation{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 130 {
		t.Fatalf("got %v, want %v", len(payload), 130)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 130 {
		t.Errorf("got %v, want %v", n, 130)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestReconciliation_NewReconciliation(t *testing.T) {
	cf := NewReconciliation()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeReconciliation)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeReconciliation)
	}
}

func TestReconciliation_Read(t *testing.T) {
	m := NewReconciliation()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("E5")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.RefTxnID = hexToBytes("f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37")
	m.TargetAddressQty = 0
	m.Timestamp = 0x0000016331e3c200
	m.Message = []byte("")

	want := "0x6a4c970000002045350052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb3700000000000000000000016331e3c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 154 {
		t.Errorf("got %v, want %v\n", n, 154)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestReconciliation_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c970000002045350052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb3700000000000000000000016331e3c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Reconciliation{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 154 {
		t.Fatalf("got %v, want %v", len(payload), 154)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 154 {
		t.Errorf("got %v, want %v", n, 154)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestInitiative_NewInitiative(t *testing.T) {
	cf := NewInitiative()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeInitiative)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeInitiative)
	}
}

func TestInitiative_Read(t *testing.T) {
	m := NewInitiative()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G1")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteType = byte('C')
	m.VoteOptions = []byte("ABCDEFGHIJKLMNO")
	m.VoteMax = 15
	m.VoteLogic = byte('0')
	m.ProposalDescription = []byte("Change the name of the Contract.")
	m.ProposalDocumentHash = hexToBytes("77201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec")
	m.VoteCutOffTimestamp = 0x0000016482fd5d80

	want := "0x6a4cb60000002047310052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec0000016482fd5d80"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 185 {
		t.Errorf("got %v, want %v\n", n, 185)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestInitiative_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cb60000002047310052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec0000016482fd5d80"
	cf := Initiative{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 185 {
		t.Fatalf("got %v, want %v", len(payload), 185)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 185 {
		t.Errorf("got %v, want %v", n, 185)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestReferendum_NewReferendum(t *testing.T) {
	cf := NewReferendum()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeReferendum)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeReferendum)
	}
}

func TestReferendum_Read(t *testing.T) {
	m := NewReferendum()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G2")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteType = byte('C')
	m.VoteOptions = []byte("ABCDEFGHIJKLMNO")
	m.VoteMax = 15
	m.VoteLogic = byte('0')
	m.ProposalDescription = []byte("Change the name of the Contract.")
	m.ProposalDocumentHash = hexToBytes("77201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec")
	m.VoteCutOffTimestamp = 0x0000016482fd5d80

	want := "0x6a4cb60000002047320052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec0000016482fd5d80"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 185 {
		t.Errorf("got %v, want %v\n", n, 185)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestReferendum_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cb60000002047320052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec0000016482fd5d80"
	cf := Referendum{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 185 {
		t.Fatalf("got %v, want %v", len(payload), 185)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 185 {
		t.Errorf("got %v, want %v", n, 185)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestVote_NewVote(t *testing.T) {
	cf := NewVote()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeVote)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeVote)
	}
}

func TestVote_Read(t *testing.T) {
	m := NewVote()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G3")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteType = byte('C')
	m.VoteOptions = []byte("ABCDEFGHIJKLMNO")
	m.VoteMax = 15
	m.VoteLogic = byte('0')
	m.ProposalDescription = []byte("Change the name of the Contract.")
	m.ProposalDocumentHash = hexToBytes("77201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec")
	m.VoteCutOffTimestamp = 0x00000166550ce380
	m.Timestamp = 0x0000016331e3c200

	want := "0x6a4cbe0000002047330052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec00000166550ce3800000016331e3c200"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 193 {
		t.Errorf("got %v, want %v\n", n, 193)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestVote_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cbe0000002047330052524561706d3271737a6e686b7332337a386438337534317338303139687972693369434142434445464748494a4b4c4d4e4f0f304368616e676520746865206e616d65206f662074686520436f6e74726163742e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000077201b0094f50df309f0343e4f44dae64d0de503c91038faf2c6b039f9f18aec00000166550ce3800000016331e3c200"
	cf := Vote{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 193 {
		t.Fatalf("got %v, want %v", len(payload), 193)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 193 {
		t.Errorf("got %v, want %v", n, 193)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestBallotCast_NewBallotCast(t *testing.T) {
	cf := NewBallotCast()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeBallotCast)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeBallotCast)
	}
}

func TestBallotCast_Read(t *testing.T) {
	m := NewBallotCast()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G4")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteTxnID = hexToBytes("f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37")
	m.Vote = []byte("1")

	want := "0x6a4c590000002047340052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37310000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 92 {
		t.Errorf("got %v, want %v\n", n, 92)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestBallotCast_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c590000002047340052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37310000000000000000000000000000"
	cf := BallotCast{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 92 {
		t.Fatalf("got %v, want %v", len(payload), 92)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 92 {
		t.Errorf("got %v, want %v", n, 92)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestBallotCounted_NewBallotCounted(t *testing.T) {
	cf := NewBallotCounted()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeBallotCounted)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeBallotCounted)
	}
}

func TestBallotCounted_Read(t *testing.T) {
	m := NewBallotCounted()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G5")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteTxnID = hexToBytes("f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37")
	m.Vote = []byte("1")
	m.Timestamp = 0x0000016331e3c200

	want := "0x6a4c620000002047350052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37310000000000000000000000000000000000016331e3c200"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 101 {
		t.Errorf("got %v, want %v\n", n, 101)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestBallotCounted_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c620000002047350052524561706d3271737a6e686b7332337a386438337534317338303139687972693369f3318be9fb3f73e53b29868beae46b42911c2116f979a5d3284face90746cb37310000000000000000000000000000000000016331e3c200"
	cf := BallotCounted{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 101 {
		t.Fatalf("got %v, want %v", len(payload), 101)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 101 {
		t.Errorf("got %v, want %v", n, 101)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestResult_NewResult(t *testing.T) {
	cf := NewResult()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeResult)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeResult)
	}
}

func TestResult_Read(t *testing.T) {
	m := NewResult()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("G6")
	m.Version = 0
	m.AssetType = []byte("SHC")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.VoteType = byte('C')
	m.VoteTxnID = hexToBytes("f2318be9fb3f73e53a29868beae46b42911c2116f979a5d3284face90746cb37")
	m.Timestamp = 0x0000016331e3c200
	m.Option1Tally = 3000
	m.Option2Tally = 56433
	m.Option3Tally = 43
	m.Option4Tally = 0
	m.Option5Tally = 0
	m.Option6Tally = 334
	m.Option7Tally = 5555
	m.Option8Tally = 39222
	m.Option9Tally = 0
	m.Option10Tally = 0
	m.Option11Tally = 0
	m.Option12Tally = 0
	m.Option13Tally = 0
	m.Option14Tally = 0
	m.Option15Tally = 0
	m.Result = []byte("2")

	want := "0x6a4cdb0000002047360053484361706d3271737a6e686b7332337a38643833753431733830313968797269336943f2318be9fb3f73e53a29868beae46b42911c2116f979a5d3284face90746cb370000016331e3c2000000000000000bb8000000000000dc71000000000000002b00000000000000000000000000000000000000000000014e00000000000015b30000000000009936000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 222 {
		t.Errorf("got %v, want %v\n", n, 222)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestResult_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdb0000002047360053484361706d3271737a6e686b7332337a38643833753431733830313968797269336943f2318be9fb3f73e53a29868beae46b42911c2116f979a5d3284face90746cb370000016331e3c2000000000000000bb8000000000000dc71000000000000002b00000000000000000000000000000000000000000000014e00000000000015b30000000000009936000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000"
	cf := Result{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 222 {
		t.Fatalf("got %v, want %v", len(payload), 222)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 222 {
		t.Errorf("got %v, want %v", n, 222)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestMessage_NewMessage(t *testing.T) {
	cf := NewMessage()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeMessage)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeMessage)
	}
}

func TestMessage_Read(t *testing.T) {
	m := NewMessage()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("M1")
	m.Version = 0
	m.Timestamp = 0x0000016331e3c200
	m.MessageType = []byte("6")
	m.Message = []byte("Hello world!")

	want := "0x6a4cdc000000204d31000000016331e3c200360048656c6c6f20776f726c64210000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestMessage_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc000000204d31000000016331e3c200360048656c6c6f20776f726c64210000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Message{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestRejection_NewRejection(t *testing.T) {
	cf := NewRejection()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeRejection)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeRejection)
	}
}

func TestRejection_Read(t *testing.T) {
	m := NewRejection()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("M2")
	m.Version = 0
	m.Timestamp = 0x0000016331e3c200
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.RejectionType = byte('F')
	m.Message = []byte("Sorry, you don't have enough tokens.")

	want := "0x6a4cdc000000204d32000000016331e3c20052524561706d3271737a6e686b7332337a38643833753431733830313968797269336946536f7272792c20796f7520646f6e2774206861766520656e6f75676820746f6b656e732e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestRejection_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc000000204d32000000016331e3c20052524561706d3271737a6e686b7332337a38643833753431733830313968797269336946536f7272792c20796f7520646f6e2774206861766520656e6f75676820746f6b656e732e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Rejection{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestEstablishment_NewEstablishment(t *testing.T) {
	cf := NewEstablishment()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeEstablishment)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeEstablishment)
	}
}

func TestEstablishment_Read(t *testing.T) {
	m := NewEstablishment()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("R1")
	m.Version = 0
	m.Registrar = []byte("Coinbase")
	m.RegisterType = 0x00
	m.KYCJurisdiction = []byte("AUS")
	m.DOB = 0x0000007ff7a48180
	m.CountryOfResidence = []byte("AUS")
	m.SupportingDocumentationHash = hexToBytes("98ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4")
	m.Message = []byte("North America Whitelist")

	want := "0x6a4cdc00000020523100436f696e6261736500000000000000000041555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be44e6f72746820416d65726963612057686974656c6973740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestEstablishment_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc00000020523100436f696e6261736500000000000000000041555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be44e6f72746820416d65726963612057686974656c6973740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Establishment{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAddition_NewAddition(t *testing.T) {
	cf := NewAddition()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeAddition)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeAddition)
	}
}

func TestAddition_Read(t *testing.T) {
	m := NewAddition()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("R2")
	m.Version = 0
	m.Sublist = []byte("")
	m.KYC = byte('Y')
	m.KYCJurisdiction = []byte("AUS")
	m.DOB = 0x0000007ff7a48180
	m.CountryOfResidence = []byte("AUS")
	m.SupportingDocumentationHash = hexToBytes("98ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4")
	m.Message = []byte("username")

	want := "0x6a4cd000000020523200000000005941555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4757365726e616d650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 211 {
		t.Errorf("got %v, want %v\n", n, 211)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAddition_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cd000000020523200000000005941555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4757365726e616d650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Addition{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 211 {
		t.Fatalf("got %v, want %v", len(payload), 211)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 211 {
		t.Errorf("got %v, want %v", n, 211)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAlteration_NewAlteration(t *testing.T) {
	cf := NewAlteration()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeAlteration)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeAlteration)
	}
}

func TestAlteration_Read(t *testing.T) {
	m := NewAlteration()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("R3")
	m.Version = 0
	m.Sublist = []byte("")
	m.KYC = byte('Y')
	m.KYCJurisdiction = []byte("AUS")
	m.DOB = 0x0000007ff7a48180
	m.CountryOfResidence = []byte("AUS")
	m.SupportingDocumentationHash = hexToBytes("98ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4")
	m.Message = []byte("Changed Country of Residence")

	want := "0x6a4cdc00000020523300000000005941555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be44368616e67656420436f756e747279206f66205265736964656e6365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestAlteration_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc00000020523300000000005941555300000000007ff7a4818041555398ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be44368616e67656420436f756e747279206f66205265736964656e6365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Alteration{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestRemoval_NewRemoval(t *testing.T) {
	cf := NewRemoval()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeRemoval)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeRemoval)
	}
}

func TestRemoval_Read(t *testing.T) {
	m := NewRemoval()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("R4")
	m.Version = 0
	m.SupportingDocumentationHash = hexToBytes("98ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be4")
	m.Message = []byte("Removed due to violation of company policy.")

	want := "0x6a4cdc0000002052340098ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be452656d6f7665642064756520746f2076696f6c6174696f6e206f6620636f6d70616e7920706f6c6963792e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v\n", n, 223)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestRemoval_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4cdc0000002052340098ea6e4f216f2fb4b69fff9b3a44842c38686ca685f3f55dc48c5d3fb1107be452656d6f7665642064756520746f2076696f6c6174696f6e206f6620636f6d70616e7920706f6c6963792e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	cf := Removal{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 223 {
		t.Fatalf("got %v, want %v", len(payload), 223)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 223 {
		t.Errorf("got %v, want %v", n, 223)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSend_NewSend(t *testing.T) {
	cf := NewSend()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeSend)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeSend)
	}
}

func TestSend_Read(t *testing.T) {
	m := NewSend()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("T1")
	m.Version = 0
	m.AssetType = []byte("SHC")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.TokenQty = 1000000

	want := "0x6a320000002054310053484361706d3271737a6e686b7332337a38643833753431733830313968797269336900000000000f4240"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 52 {
		t.Errorf("got %v, want %v\n", n, 52)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSend_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a320000002054310053484361706d3271737a6e686b7332337a38643833753431733830313968797269336900000000000f4240"
	cf := Send{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 52 {
		t.Fatalf("got %v, want %v", len(payload), 52)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 52 {
		t.Errorf("got %v, want %v", n, 52)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestExchange_NewExchange(t *testing.T) {
	cf := NewExchange()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeExchange)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeExchange)
	}
}

func TestExchange_Read(t *testing.T) {
	m := NewExchange()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("T2")
	m.Version = 0
	m.Party1AssetType = []byte("RRE")
	m.Party1AssetID = []byte("apm2qsznhks23z")
	m.Party1TokenQty = 21000
	m.OfferValidUntil = 0x0000016331e3c200
	m.ExchangeFeeCurrency = []byte("AUD")
	m.ExchangeFeeVar = 0.0050
	m.ExchangeFeeFixed = 0.01
	m.ExchangeFeeAddress = []byte("1HQ2ULuD7T5ykaucZ3KmTo4i29925Qa6ic")

	want := "0x6a4c670000002054320052524561706d3271737a6e686b7332337a00000000000000000000000000000000000000000000000052080000016331e3c2004155443ba3d70a3c23d70a31485132554c7544375435796b6175635a334b6d546f346932393932355161366963"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 106 {
		t.Errorf("got %v, want %v\n", n, 106)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestExchange_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c670000002054320052524561706d3271737a6e686b7332337a00000000000000000000000000000000000000000000000052080000016331e3c2004155443ba3d70a3c23d70a31485132554c7544375435796b6175635a334b6d546f346932393932355161366963"
	cf := Exchange{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 106 {
		t.Fatalf("got %v, want %v", len(payload), 106)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 106 {
		t.Errorf("got %v, want %v", n, 106)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSwap_NewSwap(t *testing.T) {
	cf := NewSwap()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeSwap)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeSwap)
	}
}

func TestSwap_Read(t *testing.T) {
	m := NewSwap()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("T3")
	m.Version = 0
	m.Party1AssetType = []byte("RRE")
	m.Party1AssetID = []byte("ran2qsznhis53z")
	m.Party1TokenQty = 5000
	m.OfferValidUntil = 0x0000016331e3c200
	m.Party2AssetType = []byte("SHC")
	m.Party2AssetID = []byte("apm2qsznhks23z")
	m.Party2TokenQty = 1000
	m.ExchangeFeeCurrency = []byte("AUD")
	m.ExchangeFeeVar = 0.0050
	m.ExchangeFeeFixed = 0.01
	m.ExchangeFeeAddress = []byte("1HQ2ULuD7T5ykaucZ3KmTo4i29925Qa6ic")

	want := "0x6a4c920000002054330052524572616e3271737a6e68697335337a00000000000000000000000000000000000000000000000013880000016331e3c20053484361706d3271737a6e686b7332337a00000000000000000000000000000000000000000000000003e84155443ba3d70a3c23d70a31485132554c7544375435796b6175635a334b6d546f346932393932355161366963"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 149 {
		t.Errorf("got %v, want %v\n", n, 149)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSwap_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a4c920000002054330052524572616e3271737a6e68697335337a00000000000000000000000000000000000000000000000013880000016331e3c20053484361706d3271737a6e686b7332337a00000000000000000000000000000000000000000000000003e84155443ba3d70a3c23d70a31485132554c7544375435796b6175635a334b6d546f346932393932355161366963"
	cf := Swap{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 149 {
		t.Fatalf("got %v, want %v", len(payload), 149)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 149 {
		t.Errorf("got %v, want %v", n, 149)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSettlement_NewSettlement(t *testing.T) {
	cf := NewSettlement()

	if cf.ProtocolID != ProtocolID {
		t.Errorf("got %v, want %v", cf.ProtocolID, ProtocolID)
	}

	if !reflect.DeepEqual(cf.ActionPrefix, []byte(CodeSettlement)) {
		t.Errorf("got %v, want %v", cf.ActionPrefix, CodeSettlement)
	}
}

func TestSettlement_Read(t *testing.T) {
	m := NewSettlement()

	m.ProtocolID = 32
	m.ActionPrefix = []byte("T4")
	m.Version = 0
	m.AssetType = []byte("RRE")
	m.AssetID = []byte("apm2qsznhks23z8d83u41s8019hyri3i")
	m.Party1TokenQty = 21000
	m.Party2TokenQty = 1000000
	m.Timestamp = 0x0000016331e3c200

	want := "0x6a420000002054340052524561706d3271737a6e686b7332337a386438337534317338303139687972693369000000000000520800000000000f42400000016331e3c200"

	b := make([]byte, m.Len(), m.Len())
	n, err := m.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	if n != 68 {
		t.Errorf("got %v, want %v\n", n, 68)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}

func TestSettlement_Write(t *testing.T) {
	// this is the same as the above fixture, with the "0x" removed
	data := "6a420000002054340052524561706d3271737a6e686b7332337a386438337534317338303139687972693369000000000000520800000000000f42400000016331e3c200"
	cf := Settlement{}

	payload, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}

	// make sure bytes are correct length before testing
	if len(payload) != 68 {
		t.Fatalf("got %v, want %v", len(payload), 68)
	}

	n, err := cf.Write(payload)
	if err != nil {
		t.Fatalf("got %v, expected nil", err)
	}

	if n != 68 {
		t.Errorf("got %v, want %v", n, 68)
	}

	want := fmt.Sprintf("0x%s", data)
	b, err := cf.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	hex := fmt.Sprintf("%#x", b)

	if want != hex {
		t.Errorf("got\n    %v\n, want\n    %v", hex, want)
	}
}
