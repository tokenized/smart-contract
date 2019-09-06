package inspector

import (
	"testing"
)

const (
	issuerAddr0 = "1Eo67Dwb9F9xCBucvtz15oSQDFkyAh3gdz"
)

func TestGetQuantity(t *testing.T) {
	_ = decodeAddress(issuerAddr0)

	receiverAddr := "1KVM9oiiwKaEsgHKePoHE6qtcE4KAu7Jgd"
	_ = decodeAddress(receiverAddr)

	// testArr := []struct {
	// 	name    string
	// 	message actions.Action
	// 	tx      *Transaction
	// 	address bitcoin.Address
	// 	want    Balance
	// }{
	// 	// {
	// 	// name: "AssetCreation (A2)",
	// 	// message: &protocol.AssetCreation{
	// 	// Qty: 42,
	// 	// },
	// 	// address: issuer,
	// 	// want: Balance{
	// 	// Qty: 42,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Settlement (T4) Sender",
	// 	// tx: &Transaction{
	// 	// Outputs: []Output{
	// 	// Output{
	// 	// Address: decodeAddress(issuerAddr0),
	// 	// },
	// 	// },
	// 	// },
	// 	// message: &protocol.Settlement{
	// 	// Party1TokenQty: 58,
	// 	// Party2TokenQty: 42,
	// 	// },
	// 	// address: issuer,
	// 	// want: Balance{
	// 	// Qty: 58,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Settlement (T4) Receiver",
	// 	// tx: &Transaction{
	// 	// Outputs: []Output{
	// 	// Output{
	// 	// Address: decodeAddress(issuerAddr0),
	// 	// },
	// 	// Output{
	// 	// Address: decodeAddress(receiverAddr),
	// 	// },
	// 	// },
	// 	// },
	// 	// message: &protocol.Settlement{
	// 	// Party1TokenQty: 58,
	// 	// Party2TokenQty: 42,
	// 	// },
	// 	// address: receiver,
	// 	// want: Balance{
	// 	// Qty: 42,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Freeze (E2)",
	// 	// message: &protocol.Freeze{
	// 	// Qty: 42,
	// 	// },
	// 	// want: Balance{
	// 	// Qty:    42,
	// 	// Frozen: 42,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Thaw (E3)",
	// 	// message: &protocol.Thaw{
	// 	// Qty: 42,
	// 	// },
	// 	// want: Balance{
	// 	// Qty:    42,
	// 	// Frozen: 0,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Confiscation (E4) Target Balance Reduces",
	// 	// tx: &Transaction{
	// 	// Outputs: []Output{
	// 	// Output{
	// 	// Address: receiver,
	// 	// },
	// 	// Output{
	// 	// Address: issuer,
	// 	// },
	// 	// },
	// 	// },
	// 	// message: &protocol.Confiscation{
	// 	// TargetsQty:  0,
	// 	// DepositsQty: 1000,
	// 	// },
	// 	// address: receiver,
	// 	// want: Balance{
	// 	// Qty:    0,
	// 	// Frozen: 0,
	// 	// },
	// 	// },
	// 	// {
	// 	// name: "Reconciliation (E5)",
	// 	// message: &protocol.Reconciliation{
	// 	// TargetAddressQty: 42,
	// 	// },
	// 	// address: issuer,
	// 	// want: Balance{
	// 	// Qty:    42,
	// 	// Frozen: 0,
	// 	// },
	// 	// },
	// }

	// for _, tt := range testArr {
	// 	t.Run(tt.name, func(t *testing.T) {
	//
	// 		balance := GetProtocolQuantity(tt.tx, tt.message, tt.address)
	//
	// 		if !reflect.DeepEqual(balance, tt.want) {
	// 			t.Errorf("got\n%+v\nwant\n%+v", balance, tt.want)
	// 		}
	// 	})
	// }
}
