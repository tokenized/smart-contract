package protocol

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// AssetTypeLen is the size in bytes of all asset type variants.
	AssetTypeLen = 152

	// CodeCoupon identifies data as a Coupon message.
	CodeCoupon = "COU"

	// CodeCurrency identifies data as a Currency message.
	CodeCurrency = "CUR"

	// CodeLoyaltyPoints identifies data as a LoyaltyPoints message.
	CodeLoyaltyPoints = "LOY"

	// CodeMembership identifies data as a Membership message.
	CodeMembership = "MEM"

	// CodeShareCommon identifies data as a ShareCommon message.
	CodeShareCommon = "SHC"

	// CodeTicketAdmission identifies data as a TicketAdmission message.
	CodeTicketAdmission = "TIC"
)

// AssetTypeMapping holds a mapping of asset codes to asset types.
func AssetTypeMapping(code string) PayloadMessage {
	switch code {
	case CodeCoupon:
		result := Coupon{}
		return &result
	case CodeCurrency:
		result := Currency{}
		return &result
	case CodeLoyaltyPoints:
		result := LoyaltyPoints{}
		return &result
	case CodeMembership:
		result := Membership{}
		return &result
	case CodeShareCommon:
		result := ShareCommon{}
		return &result
	case CodeTicketAdmission:
		result := TicketAdmission{}
		return &result
	default:
		return nil
	}
}

// Coupon asset type.
type Coupon struct {
	Version         uint8     `json:"version,omitempty"`          // Payload Version
	RedeemingEntity string    `json:"redeeming_entity,omitempty"` //
	IssueDate       Timestamp `json:"issue_date,omitempty"`       //
	ExpiryDate      Timestamp `json:"expiry_date,omitempty"`      //
	Value           uint64    `json:"value,omitempty"`            //
	Currency        string    `json:"currency,omitempty"`         //
	Description     string    `json:"description,omitempty"`      //
}

// Type returns the type identifer for this message.
func (m Coupon) Type() string {
	return CodeCoupon
}

// Len returns the byte size of this message.
func (m Coupon) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *Coupon) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Coupon) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// RedeemingEntity (string)
	if err := WriteVarChar(buf, m.RedeemingEntity, 8); err != nil {
		return nil, err
	}

	// IssueDate (Timestamp)
	if err := write(buf, m.IssueDate); err != nil {
		return nil, err
	}

	// ExpiryDate (Timestamp)
	if err := write(buf, m.ExpiryDate); err != nil {
		return nil, err
	}

	// Value (uint64)
	if err := write(buf, m.Value); err != nil {
		return nil, err
	}

	// Currency (string)
	if err := WriteFixedChar(buf, m.Currency, 3); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in Coupon from the byte slice
func (m *Coupon) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// RedeemingEntity (string)
	{
		var err error
		m.RedeemingEntity, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// IssueDate (Timestamp)
	if err := m.IssueDate.Write(buf); err != nil {
		return 0, err
	}

	// ExpiryDate (Timestamp)
	if err := m.ExpiryDate.Write(buf); err != nil {
		return 0, err
	}

	// Value (uint64)
	if err := read(buf, &m.Value); err != nil {
		return 0, err
	}

	// Currency (string)
	{
		var err error
		m.Currency, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m Coupon) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("RedeemingEntity:%#+v", m.RedeemingEntity))
	vals = append(vals, fmt.Sprintf("IssueDate:%#+v", m.IssueDate))
	vals = append(vals, fmt.Sprintf("ExpiryDate:%#+v", m.ExpiryDate))
	vals = append(vals, fmt.Sprintf("Value:%v", m.Value))
	vals = append(vals, fmt.Sprintf("Currency:%#+v", m.Currency))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Currency asset type.
type Currency struct {
	Version           uint8  `json:"version,omitempty"`            // Payload Version
	ISOCode           string `json:"iso_code,omitempty"`           //
	MonetaryAuthority string `json:"monetary_authority,omitempty"` //
	Description       string `json:"description,omitempty"`        //
}

// Type returns the type identifer for this message.
func (m Currency) Type() string {
	return CodeCurrency
}

// Len returns the byte size of this message.
func (m Currency) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *Currency) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Currency) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// ISOCode (string)
	if err := WriteFixedChar(buf, m.ISOCode, 3); err != nil {
		return nil, err
	}

	// MonetaryAuthority (string)
	if err := WriteVarChar(buf, m.MonetaryAuthority, 8); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in Currency from the byte slice
func (m *Currency) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// ISOCode (string)
	{
		var err error
		m.ISOCode, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// MonetaryAuthority (string)
	{
		var err error
		m.MonetaryAuthority, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m Currency) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("ISOCode:%#+v", m.ISOCode))
	vals = append(vals, fmt.Sprintf("MonetaryAuthority:%#+v", m.MonetaryAuthority))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// LoyaltyPoints asset type.
type LoyaltyPoints struct {
	Version             uint8     `json:"version,omitempty"`              // Payload Version
	AgeRestriction      uint8     `json:"age_restriction,omitempty"`      //
	OfferType           byte      `json:"offer_type,omitempty"`           //
	OfferName           string    `json:"offer_name,omitempty"`           //
	ValidFrom           Timestamp `json:"valid_from,omitempty"`           //
	ExpirationTimestamp Timestamp `json:"expiration_timestamp,omitempty"` //
	Description         string    `json:"description,omitempty"`          //
}

// Type returns the type identifer for this message.
func (m LoyaltyPoints) Type() string {
	return CodeLoyaltyPoints
}

// Len returns the byte size of this message.
func (m LoyaltyPoints) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *LoyaltyPoints) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *LoyaltyPoints) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// AgeRestriction (uint8)
	if err := write(buf, m.AgeRestriction); err != nil {
		return nil, err
	}

	// OfferType (byte)
	if err := write(buf, m.OfferType); err != nil {
		return nil, err
	}

	// OfferName (string)
	if err := WriteVarChar(buf, m.OfferName, 8); err != nil {
		return nil, err
	}

	// ValidFrom (Timestamp)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in LoyaltyPoints from the byte slice
func (m *LoyaltyPoints) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// AgeRestriction (uint8)
	if err := read(buf, &m.AgeRestriction); err != nil {
		return 0, err
	}

	// OfferType (byte)
	if err := read(buf, &m.OfferType); err != nil {
		return 0, err
	}

	// OfferName (string)
	{
		var err error
		m.OfferName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ValidFrom (Timestamp)
	if err := m.ValidFrom.Write(buf); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := m.ExpirationTimestamp.Write(buf); err != nil {
		return 0, err
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m LoyaltyPoints) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#+v", m.AgeRestriction))
	vals = append(vals, fmt.Sprintf("OfferType:%#+v", m.OfferType))
	vals = append(vals, fmt.Sprintf("OfferName:%#+v", m.OfferName))
	vals = append(vals, fmt.Sprintf("ValidFrom:%#+v", m.ValidFrom))
	vals = append(vals, fmt.Sprintf("ExpirationTimestamp:%#+v", m.ExpirationTimestamp))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Membership asset type.
type Membership struct {
	Version             uint8     `json:"version,omitempty"`              // Payload Version
	AgeRestriction      uint8     `json:"age_restriction,omitempty"`      //
	ValidFrom           Timestamp `json:"valid_from,omitempty"`           //
	ExpirationTimestamp Timestamp `json:"expiration_timestamp,omitempty"` //
	ID                  string    `json:"id,omitempty"`                   //
	MembershipType      string    `json:"membership_type,omitempty"`      //
	Description         string    `json:"description,omitempty"`          //
}

// Type returns the type identifer for this message.
func (m Membership) Type() string {
	return CodeMembership
}

// Len returns the byte size of this message.
func (m Membership) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *Membership) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Membership) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// AgeRestriction (uint8)
	if err := write(buf, m.AgeRestriction); err != nil {
		return nil, err
	}

	// ValidFrom (Timestamp)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// ID (string)
	if err := WriteVarChar(buf, m.ID, 8); err != nil {
		return nil, err
	}

	// MembershipType (string)
	if err := WriteVarChar(buf, m.MembershipType, 8); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in Membership from the byte slice
func (m *Membership) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// AgeRestriction (uint8)
	if err := read(buf, &m.AgeRestriction); err != nil {
		return 0, err
	}

	// ValidFrom (Timestamp)
	if err := m.ValidFrom.Write(buf); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := m.ExpirationTimestamp.Write(buf); err != nil {
		return 0, err
	}

	// ID (string)
	{
		var err error
		m.ID, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// MembershipType (string)
	{
		var err error
		m.MembershipType, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m Membership) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#+v", m.AgeRestriction))
	vals = append(vals, fmt.Sprintf("ValidFrom:%#+v", m.ValidFrom))
	vals = append(vals, fmt.Sprintf("ExpirationTimestamp:%#+v", m.ExpirationTimestamp))
	vals = append(vals, fmt.Sprintf("ID:%#+v", m.ID))
	vals = append(vals, fmt.Sprintf("MembershipType:%#+v", m.MembershipType))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ShareCommon asset type.
type ShareCommon struct {
	Version         uint8     `json:"version,omitempty"`          // Payload Version
	TransferLockout Timestamp `json:"transfer_lockout,omitempty"` //
	Ticker          string    `json:"ticker,omitempty"`           //
	ISIN            string    `json:"isin,omitempty"`             //
	Description     string    `json:"description,omitempty"`      //
}

// Type returns the type identifer for this message.
func (m ShareCommon) Type() string {
	return CodeShareCommon
}

// Len returns the byte size of this message.
func (m ShareCommon) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *ShareCommon) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *ShareCommon) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TransferLockout (Timestamp)
	if err := write(buf, m.TransferLockout); err != nil {
		return nil, err
	}

	// Ticker (string)
	if err := WriteFixedChar(buf, m.Ticker, 5); err != nil {
		return nil, err
	}

	// ISIN (string)
	if err := WriteFixedChar(buf, m.ISIN, 12); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in ShareCommon from the byte slice
func (m *ShareCommon) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TransferLockout (Timestamp)
	if err := m.TransferLockout.Write(buf); err != nil {
		return 0, err
	}

	// Ticker (string)
	{
		var err error
		m.Ticker, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// ISIN (string)
	{
		var err error
		m.ISIN, err = ReadFixedChar(buf, 12)
		if err != nil {
			return 0, err
		}
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m ShareCommon) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("TransferLockout:%#+v", m.TransferLockout))
	vals = append(vals, fmt.Sprintf("Ticker:%#+v", m.Ticker))
	vals = append(vals, fmt.Sprintf("ISIN:%#+v", m.ISIN))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// TicketAdmission asset type.
type TicketAdmission struct {
	Version             uint8     `json:"version,omitempty"`              // Payload Version
	AgeRestriction      uint8     `json:"age_restriction,omitempty"`      //
	AdmissionType       string    `json:"admission_type,omitempty"`       //
	Venue               string    `json:"venue,omitempty"`                //
	Class               string    `json:"class,omitempty"`                //
	Area                string    `json:"area,omitempty"`                 //
	Seat                string    `json:"seat,omitempty"`                 //
	StartTimeDate       Timestamp `json:"start_time_date,omitempty"`      //
	ValidFrom           Timestamp `json:"valid_from,omitempty"`           //
	ExpirationTimestamp Timestamp `json:"expiration_timestamp,omitempty"` //
	Description         string    `json:"description,omitempty"`          //
}

// Type returns the type identifer for this message.
func (m TicketAdmission) Type() string {
	return CodeTicketAdmission
}

// Len returns the byte size of this message.
func (m TicketAdmission) Len() int64 {
	return AssetTypeLen
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m *TicketAdmission) Read(b []byte) (int, error) {
	data, err := m.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *TicketAdmission) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// AgeRestriction (uint8)
	if err := write(buf, m.AgeRestriction); err != nil {
		return nil, err
	}

	// AdmissionType (string)
	if err := WriteFixedChar(buf, m.AdmissionType, 3); err != nil {
		return nil, err
	}

	// Venue (string)
	if err := WriteVarChar(buf, m.Venue, 8); err != nil {
		return nil, err
	}

	// Class (string)
	if err := WriteVarChar(buf, m.Class, 8); err != nil {
		return nil, err
	}

	// Area (string)
	if err := WriteVarChar(buf, m.Area, 8); err != nil {
		return nil, err
	}

	// Seat (string)
	if err := WriteVarChar(buf, m.Seat, 8); err != nil {
		return nil, err
	}

	// StartTimeDate (Timestamp)
	if err := write(buf, m.StartTimeDate); err != nil {
		return nil, err
	}

	// ValidFrom (Timestamp)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write populates the fields in TicketAdmission from the byte slice
func (m *TicketAdmission) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// AgeRestriction (uint8)
	if err := read(buf, &m.AgeRestriction); err != nil {
		return 0, err
	}

	// AdmissionType (string)
	{
		var err error
		m.AdmissionType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// Venue (string)
	{
		var err error
		m.Venue, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Class (string)
	{
		var err error
		m.Class, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Area (string)
	{
		var err error
		m.Area, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Seat (string)
	{
		var err error
		m.Seat, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// StartTimeDate (Timestamp)
	if err := m.StartTimeDate.Write(buf); err != nil {
		return 0, err
	}

	// ValidFrom (Timestamp)
	if err := m.ValidFrom.Write(buf); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (Timestamp)
	if err := m.ExpirationTimestamp.Write(buf); err != nil {
		return 0, err
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m TicketAdmission) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#+v", m.AgeRestriction))
	vals = append(vals, fmt.Sprintf("AdmissionType:%#+v", m.AdmissionType))
	vals = append(vals, fmt.Sprintf("Venue:%#+v", m.Venue))
	vals = append(vals, fmt.Sprintf("Class:%#+v", m.Class))
	vals = append(vals, fmt.Sprintf("Area:%#+v", m.Area))
	vals = append(vals, fmt.Sprintf("Seat:%#+v", m.Seat))
	vals = append(vals, fmt.Sprintf("StartTimeDate:%#+v", m.StartTimeDate))
	vals = append(vals, fmt.Sprintf("ValidFrom:%#+v", m.ValidFrom))
	vals = append(vals, fmt.Sprintf("ExpirationTimestamp:%#+v", m.ExpirationTimestamp))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
