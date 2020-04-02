package bsvalias

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// GetPaymentDestination gets a locking script that can be used to send bitcoin.
// If senderKey is not nil then it must be associated with senderHandle and will be used to add a
//   signature to the request.
func (i *Identity) GetPaymentDestination(senderName, senderHandle, purpose string,
	amount uint64, senderKey *bitcoin.Key) ([]byte, error) {

	if len(i.Site.Capabilities.PaymentDestination) == 0 {
		return nil, errors.Wrap(ErrNotCapable, "payment-destination")
	}

	request := struct {
		SenderName   string `json:"senderName"`
		SenderHandle string `json:"senderHandle"`
		DateTime     string `json:"dt"`
		Amount       uint64 `json:"amount"`
		Purpose      string `json:"purpose"`
		Signature    string `json:"signature"`
	}{
		SenderName:   senderName,
		SenderHandle: senderHandle,
		DateTime:     time.Now().UTC().Format(time.RFC3339),
		Amount:       amount,
		Purpose:      purpose,
	}

	if senderKey != nil {
		sigHash, err := SignatureHashForMessage(request.SenderHandle + request.DateTime +
			strconv.FormatUint(request.Amount, 10) + request.Purpose)
		if err != nil {
			return nil, errors.Wrap(err, "signature hash")
		}

		sig, err := senderKey.Sign(sigHash.Bytes())
		if err != nil {
			return nil, errors.Wrap(err, "sign")
		}

		request.Signature = sig.String()
	}

	var response struct {
		Output string `json:"output"`
	}

	url := strings.ReplaceAll(i.Site.Capabilities.PaymentDestination, "{alias}", i.Alias)
	url = strings.ReplaceAll(url, "{domain.tld}", i.Hostname)
	if err := post(url, request, &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	result, err := hex.DecodeString(response.Output)
	if err != nil {
		return nil, errors.Wrap(err, "parse script hex")
	}

	if len(result) == 0 {
		return nil, errors.New("Empty locking script")
	}

	return result, nil
}

type PaymentRequest struct {
	Tx      wire.MsgTx
	Outputs []wire.TxOut
}

// GetPaymentRequest gets a payment request from the identity.
//   senderHandle is required.
//   assetID can be empty or "BSV" to request bitcoin.
// If senderKey is not nil then it must be associated with senderHandle and will be used to add a
//   signature to the request.
func (i *Identity) GetPaymentRequest(senderName, senderHandle, purpose, assetID string,
	amount uint64, senderKey *bitcoin.Key) (PaymentRequest, error) {

	if len(i.Site.Capabilities.PaymentRequest) == 0 {
		return PaymentRequest{}, errors.Wrap(ErrNotCapable, "payment-request")
	}

	request := struct {
		SenderName   string `json:"senderName"`
		SenderHandle string `json:"senderHandle"`
		DateTime     string `json:"dt"`
		AssetID      string `json:"assetID"`
		Amount       uint64 `json:"amount"`
		Purpose      string `json:"purpose"`
		Signature    string `json:"signature"`
	}{
		SenderName:   senderName,
		SenderHandle: senderHandle,
		DateTime:     time.Now().UTC().Format(time.RFC3339),
		AssetID:      assetID,
		Amount:       amount,
		Purpose:      purpose,
	}

	if senderKey != nil {
		sigHash, err := SignatureHashForMessage(request.SenderHandle + request.DateTime +
			request.AssetID + strconv.FormatUint(request.Amount, 10) + request.Purpose)
		if err != nil {
			return PaymentRequest{}, errors.Wrap(err, "signature hash")
		}

		sig, err := senderKey.Sign(sigHash.Bytes())
		if err != nil {
			return PaymentRequest{}, errors.Wrap(err, "sign")
		}

		request.Signature = sig.String()
	}

	var response struct {
		PaymentRequest string   `json:"paymentRequest"`
		Outputs        []string `json:"outputs"`
	}

	url := strings.ReplaceAll(i.Site.Capabilities.PaymentRequest, "{alias}", i.Alias)
	url = strings.ReplaceAll(url, "{domain.tld}", i.Hostname)
	if err := post(url, request, &response); err != nil {
		return PaymentRequest{}, errors.Wrap(err, "http get")
	}

	b, err := hex.DecodeString(response.PaymentRequest)
	if err != nil {
		return PaymentRequest{}, errors.Wrap(err, "parse tx hex")
	}

	var result PaymentRequest
	if err := result.Tx.Deserialize(bytes.NewReader(b)); err != nil {
		return result, errors.Wrap(err, "deserialize tx")
	}

	for _, outputHex := range response.Outputs {
		b, err := hex.DecodeString(outputHex)
		if err != nil {
			return result, errors.Wrap(err, "parse output hex")
		}

		var output wire.TxOut
		if err := output.Deserialize(bytes.NewReader(b), 1, 1); err != nil {
			return result, errors.Wrap(err, "deserialize output")
		}

		result.Outputs = append(result.Outputs, output)
	}

	return result, nil
}
