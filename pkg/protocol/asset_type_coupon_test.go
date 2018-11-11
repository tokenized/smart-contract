package protocol

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestAssetTypeCoupon_Write(t *testing.T) {
	raw := "0000000048792d5665650000000000000000000000000000000000000000000000000000000001a804a6472e00000165e51f032d4675656c205361766572204361726400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	b, err := hex.DecodeString(raw)

	if err != nil {
		t.Fatal(err)
	}

	m := NewAssetTypeCoupon()
	if _, err := m.Write(b); err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("m = %+v\n", m)
	wantRedeemingEntity := []byte("Hy-Vee")
	if !reflect.DeepEqual(m.RedeemingEntity, wantRedeemingEntity) {
		t.Errorf("got\n%v\nwant\n%v", m.RedeemingEntity, wantRedeemingEntity)
	}

	wantDescription := []byte("Fuel Saver Card")
	if !reflect.DeepEqual(m.Description, wantDescription) {
		t.Errorf("got\n%v\nwant\n%v", m.Description, wantDescription)
	}

	wantExpiryDate := uint64(1821144139566)
	if m.ExpiryDate != wantExpiryDate {
		t.Errorf("got %v, want %v", m.ExpiryDate, wantExpiryDate)
	}

	wantIssueDate := uint64(1537147339565)
	if m.IssueDate != wantIssueDate {
		t.Errorf("got %v, want %v", m.IssueDate, wantIssueDate)
	}
}
