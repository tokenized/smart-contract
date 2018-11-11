package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// The code in this file is auto-generated. Do not edit it by hand as it will
// be overwritten when code is regenerated.

// AssetTypeCouponForm is a JSON friendly model for an asset type.
type AssetTypeCouponForm struct {
	Version            uint8  `json:"version,omitempty"`
	TradingRestriction string `json:"trading_restriction,omitempty"`
	RedeemingEntity    string `json:"redeeming_entity,omitempty"`
	ExpiryDate         string `json:"expiry_date,omitempty"`
	IssueDate          string `json:"issue_date,omitempty"`
	Description        string `json:"description,omitempty"`
}

// NewAssetTypeCouponForm returns a new AssetTypeCouponForm.
func NewAssetTypeCouponForm() *AssetTypeCouponForm {
	return &AssetTypeCouponForm{}
}

// Validate returns an error if validation fails, nil otherwise.
func (f AssetTypeCouponForm) Validate() error {
	return nil
}

func (f AssetTypeCouponForm) pad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...)
}

// write writes the value to the buffer.
func (f AssetTypeCouponForm) write(buf *bytes.Buffer,
	v interface{}) error {

	return binary.Write(buf, binary.BigEndian, v)
}

// writeBytes writes a string of fixed length to the buffer. If the string
// is longer that the length an error will be returned.
func (f AssetTypeCouponForm) writeBytes(buf *bytes.Buffer,
	s string, l int) error {

	if len(s) > l {
		return fmt.Errorf("length exceeds %v", l)
	}

	b := f.pad([]byte(s), l)

	return f.write(buf, b)
}

// Bytes returns the form as a []byte that can be read by a protocol message.
func (f AssetTypeCouponForm) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := f.write(buf, f.Version); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.TradingRestriction, 3); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.RedeemingEntity, 32); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ExpiryDate, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.IssueDate, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Description, 100); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// PayloadMessage returns a PayloadMessage from the form, if any.
func (f AssetTypeCouponForm) PayloadMessage(code []byte) (PayloadMessage, error) {
	m, err := NewPayloadMessageFromCode(code)
	if err != nil {
		return nil, err
	}

	if m != nil {
		return m, nil
	}

	return nil, errors.New("Not implemented")
}

// AssetTypeMovieTicketForm is a JSON friendly model for an asset type.
type AssetTypeMovieTicketForm struct {
	Version             uint8  `json:"version,omitempty"`
	TradingRestriction  string `json:"trading_restriction,omitempty"`
	AgeRestriction      string `json:"age_restriction,omitempty"`
	Venue               string `json:"venue,omitempty"`
	ValidFrom           string `json:"valid_from,omitempty"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Description         string `json:"description,omitempty"`
}

// NewAssetTypeMovieTicketForm returns a new AssetTypeMovieTicketForm.
func NewAssetTypeMovieTicketForm() *AssetTypeMovieTicketForm {
	return &AssetTypeMovieTicketForm{}
}

// Validate returns an error if validation fails, nil otherwise.
func (f AssetTypeMovieTicketForm) Validate() error {
	return nil
}

func (f AssetTypeMovieTicketForm) pad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...)
}

// write writes the value to the buffer.
func (f AssetTypeMovieTicketForm) write(buf *bytes.Buffer,
	v interface{}) error {

	return binary.Write(buf, binary.BigEndian, v)
}

// writeBytes writes a string of fixed length to the buffer. If the string
// is longer that the length an error will be returned.
func (f AssetTypeMovieTicketForm) writeBytes(buf *bytes.Buffer,
	s string, l int) error {

	if len(s) > l {
		return fmt.Errorf("length exceeds %v", l)
	}

	b := f.pad([]byte(s), l)

	return f.write(buf, b)
}

// Bytes returns the form as a []byte that can be read by a protocol message.
func (f AssetTypeMovieTicketForm) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := f.write(buf, f.Version); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.TradingRestriction, 3); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.AgeRestriction, 5); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Venue, 32); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ValidFrom, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ExpirationTimestamp, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Description, 95); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// PayloadMessage returns a PayloadMessage from the form, if any.
func (f AssetTypeMovieTicketForm) PayloadMessage(code []byte) (PayloadMessage, error) {
	m, err := NewPayloadMessageFromCode(code)
	if err != nil {
		return nil, err
	}

	if m != nil {
		return m, nil
	}

	return nil, errors.New("Not implemented")
}

// AssetTypeShareCommonForm is a JSON friendly model for an asset type.
type AssetTypeShareCommonForm struct {
	Version              uint8   `json:"version,omitempty"`
	TradingRestriction   string  `json:"trading_restriction,omitempty"`
	DividendType         string  `json:"dividend_type,omitempty"`
	DividendVar          float32 `json:"dividend_var,omitempty"`
	DividendFixed        float32 `json:"dividend_fixed,omitempty"`
	DistributionInterval string  `json:"distribution_interval,omitempty"`
	Guaranteed           string  `json:"guaranteed,omitempty"`
	Ticker               string  `json:"ticker,omitempty"`
	ISIN                 string  `json:"isin,omitempty"`
	Description          string  `json:"description,omitempty"`
}

// NewAssetTypeShareCommonForm returns a new AssetTypeShareCommonForm.
func NewAssetTypeShareCommonForm() *AssetTypeShareCommonForm {
	return &AssetTypeShareCommonForm{}
}

// Validate returns an error if validation fails, nil otherwise.
func (f AssetTypeShareCommonForm) Validate() error {
	return nil
}

func (f AssetTypeShareCommonForm) pad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...)
}

// write writes the value to the buffer.
func (f AssetTypeShareCommonForm) write(buf *bytes.Buffer,
	v interface{}) error {

	return binary.Write(buf, binary.BigEndian, v)
}

// writeBytes writes a string of fixed length to the buffer. If the string
// is longer that the length an error will be returned.
func (f AssetTypeShareCommonForm) writeBytes(buf *bytes.Buffer,
	s string, l int) error {

	if len(s) > l {
		return fmt.Errorf("length exceeds %v", l)
	}

	b := f.pad([]byte(s), l)

	return f.write(buf, b)
}

// Bytes returns the form as a []byte that can be read by a protocol message.
func (f AssetTypeShareCommonForm) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := f.write(buf, f.Version); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.TradingRestriction, 3); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.DividendType, 1); err != nil {
		return nil, err
	}

	if err := f.write(buf, f.DividendVar); err != nil {
		return nil, err
	}

	if err := f.write(buf, f.DividendFixed); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.DistributionInterval, 1); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Guaranteed, 1); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Ticker, 5); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ISIN, 12); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Description, 120); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// PayloadMessage returns a PayloadMessage from the form, if any.
func (f AssetTypeShareCommonForm) PayloadMessage(code []byte) (PayloadMessage, error) {
	m, err := NewPayloadMessageFromCode(code)
	if err != nil {
		return nil, err
	}

	if m != nil {
		return m, nil
	}

	return nil, errors.New("Not implemented")
}

// AssetTypeTicketAdmissionForm is a JSON friendly model for an asset type.
type AssetTypeTicketAdmissionForm struct {
	Version             uint8  `json:"version,omitempty"`
	TradingRestriction  string `json:"trading_restriction,omitempty"`
	AgeRestriction      string `json:"age_restriction,omitempty"`
	ValidFrom           string `json:"valid_from,omitempty"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Description         string `json:"description,omitempty"`
}

// NewAssetTypeTicketAdmissionForm returns a new AssetTypeTicketAdmissionForm.
func NewAssetTypeTicketAdmissionForm() *AssetTypeTicketAdmissionForm {
	return &AssetTypeTicketAdmissionForm{}
}

// Validate returns an error if validation fails, nil otherwise.
func (f AssetTypeTicketAdmissionForm) Validate() error {
	return nil
}

func (f AssetTypeTicketAdmissionForm) pad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...)
}

// write writes the value to the buffer.
func (f AssetTypeTicketAdmissionForm) write(buf *bytes.Buffer,
	v interface{}) error {

	return binary.Write(buf, binary.BigEndian, v)
}

// writeBytes writes a string of fixed length to the buffer. If the string
// is longer that the length an error will be returned.
func (f AssetTypeTicketAdmissionForm) writeBytes(buf *bytes.Buffer,
	s string, l int) error {

	if len(s) > l {
		return fmt.Errorf("length exceeds %v", l)
	}

	b := f.pad([]byte(s), l)

	return f.write(buf, b)
}

// Bytes returns the form as a []byte that can be read by a protocol message.
func (f AssetTypeTicketAdmissionForm) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := f.write(buf, f.Version); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.TradingRestriction, 3); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.AgeRestriction, 5); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ValidFrom, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.ExpirationTimestamp, 8); err != nil {
		return nil, err
	}

	if err := f.writeBytes(buf, f.Description, 127); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// PayloadMessage returns a PayloadMessage from the form, if any.
func (f AssetTypeTicketAdmissionForm) PayloadMessage(code []byte) (PayloadMessage, error) {
	m, err := NewPayloadMessageFromCode(code)
	if err != nil {
		return nil, err
	}

	if m != nil {
		return m, nil
	}

	return nil, errors.New("Not implemented")
}
