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

// Coupon asset type.
type Coupon struct {
	Version            uint8
	TradingRestriction []byte
	RedeemingEntity    string
	IssueDate          uint64
	ExpiryDate         uint64
	Value              uint64
	Currency           string
	Description        string
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
func (m *Coupon) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Coupon) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// RedeemingEntity (string)
	if err := WriteVarChar(buf, m.RedeemingEntity, 255); err != nil {
		return nil, err
	}

	// IssueDate (uint64)
	if err := write(buf, m.IssueDate); err != nil {
		return nil, err
	}

	// ExpiryDate (uint64)
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

// write populates the fields in Coupon from the byte slice
func (m *Coupon) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	// RedeemingEntity (string)
	{
		var err error
		m.RedeemingEntity, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// IssueDate (uint64)
	if err := read(buf, &m.IssueDate); err != nil {
		return 0, err
	}

	// ExpiryDate (uint64)
	if err := read(buf, &m.ExpiryDate); err != nil {
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
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("RedeemingEntity:%#+v", m.RedeemingEntity))
	vals = append(vals, fmt.Sprintf("IssueDate:%v", m.IssueDate))
	vals = append(vals, fmt.Sprintf("ExpiryDate:%v", m.ExpiryDate))
	vals = append(vals, fmt.Sprintf("Value:%v", m.Value))
	vals = append(vals, fmt.Sprintf("Currency:%#+v", m.Currency))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Currency asset type.
type Currency struct {
	Version            uint8
	TradingRestriction []byte
	ISOCode            string
	MonetaryAuthority  string
	Description        string
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
func (m *Currency) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Currency) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// ISOCode (string)
	if err := WriteFixedChar(buf, m.ISOCode, 3); err != nil {
		return nil, err
	}

	// MonetaryAuthority (string)
	if err := WriteVarChar(buf, m.MonetaryAuthority, 255); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 255); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// write populates the fields in Currency from the byte slice
func (m *Currency) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
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
		m.MonetaryAuthority, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// Description (string)
	{
		var err error
		m.Description, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m Currency) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("ISOCode:%#+v", m.ISOCode))
	vals = append(vals, fmt.Sprintf("MonetaryAuthority:%#+v", m.MonetaryAuthority))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// LoyaltyPoints asset type.
type LoyaltyPoints struct {
	Version             uint8
	TradingRestriction  []byte
	AgeRestriction      []byte
	OfferType           byte
	OfferName           string
	ValidFrom           uint64
	ExpirationTimestamp uint64
	Description         string
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
func (m *LoyaltyPoints) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *LoyaltyPoints) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// AgeRestriction ([]byte)
	if err := write(buf, pad(m.AgeRestriction, 5)); err != nil {
		return nil, err
	}

	// OfferType (byte)
	if err := write(buf, m.OfferType); err != nil {
		return nil, err
	}

	// OfferName (string)
	if err := WriteVarChar(buf, m.OfferName, 255); err != nil {
		return nil, err
	}

	// ValidFrom (uint64)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (uint64)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// write populates the fields in LoyaltyPoints from the byte slice
func (m *LoyaltyPoints) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	// AgeRestriction ([]byte)
	m.AgeRestriction = make([]byte, 5)
	if err := readLen(buf, m.AgeRestriction); err != nil {
		return 0, err
	}

	// OfferType (byte)
	if err := read(buf, &m.OfferType); err != nil {
		return 0, err
	}

	// OfferName (string)
	{
		var err error
		m.OfferName, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// ValidFrom (uint64)
	if err := read(buf, &m.ValidFrom); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (uint64)
	if err := read(buf, &m.ExpirationTimestamp); err != nil {
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
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#x", m.AgeRestriction))
	vals = append(vals, fmt.Sprintf("OfferType:%#+v", m.OfferType))
	vals = append(vals, fmt.Sprintf("OfferName:%#+v", m.OfferName))
	vals = append(vals, fmt.Sprintf("ValidFrom:%#+v", m.ValidFrom))
	vals = append(vals, fmt.Sprintf("ExpirationTimestamp:%#+v", m.ExpirationTimestamp))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Membership asset type.
type Membership struct {
	Version             uint8
	TradingRestriction  []byte
	AgeRestriction      []byte
	ValidFrom           uint64
	ExpirationTimestamp uint64
	ID                  string
	MembershipType      string
	Description         string
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
func (m *Membership) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *Membership) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// AgeRestriction ([]byte)
	if err := write(buf, pad(m.AgeRestriction, 5)); err != nil {
		return nil, err
	}

	// ValidFrom (uint64)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (uint64)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// ID (string)
	if err := WriteVarChar(buf, m.ID, 255); err != nil {
		return nil, err
	}

	// MembershipType (string)
	if err := WriteVarChar(buf, m.MembershipType, 255); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// write populates the fields in Membership from the byte slice
func (m *Membership) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	// AgeRestriction ([]byte)
	m.AgeRestriction = make([]byte, 5)
	if err := readLen(buf, m.AgeRestriction); err != nil {
		return 0, err
	}

	// ValidFrom (uint64)
	if err := read(buf, &m.ValidFrom); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (uint64)
	if err := read(buf, &m.ExpirationTimestamp); err != nil {
		return 0, err
	}

	// ID (string)
	{
		var err error
		m.ID, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// MembershipType (string)
	{
		var err error
		m.MembershipType, err = ReadVarChar(buf, 255)
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
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#x", m.AgeRestriction))
	vals = append(vals, fmt.Sprintf("ValidFrom:%#+v", m.ValidFrom))
	vals = append(vals, fmt.Sprintf("ExpirationTimestamp:%#+v", m.ExpirationTimestamp))
	vals = append(vals, fmt.Sprintf("ID:%#+v", m.ID))
	vals = append(vals, fmt.Sprintf("MembershipType:%#+v", m.MembershipType))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ShareCommon asset type.
type ShareCommon struct {
	Version            uint8
	TradingRestriction []byte
	TransferLockout    uint64
	Ticker             string
	ISIN               string
	Description        string
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
func (m *ShareCommon) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *ShareCommon) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// TransferLockout (uint64)
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
	if err := WriteVarChar(buf, m.Description, 113); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// write populates the fields in ShareCommon from the byte slice
func (m *ShareCommon) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	// TransferLockout (uint64)
	if err := read(buf, &m.TransferLockout); err != nil {
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
		m.Description, err = ReadVarChar(buf, 113)
		if err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (m ShareCommon) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("TransferLockout:%#+v", m.TransferLockout))
	vals = append(vals, fmt.Sprintf("Ticker:%#+v", m.Ticker))
	vals = append(vals, fmt.Sprintf("ISIN:%#+v", m.ISIN))
	vals = append(vals, fmt.Sprintf("Description:%#+v", m.Description))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// TicketAdmission asset type.
type TicketAdmission struct {
	Version             uint8
	TradingRestriction  []byte
	AgeRestriction      []byte
	AdmissionType       string
	Venue               string
	Class               string
	Area                string
	Seat                string
	StartTimeDate       uint64
	ValidFrom           uint64
	ExpirationTimestamp uint64
	Description         string
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
func (m *TicketAdmission) read(b []byte) (int, error) {
	data, err := m.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (m *TicketAdmission) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	if err := write(buf, m.Version); err != nil {
		return nil, err
	}

	// TradingRestriction ([]byte)
	if err := write(buf, pad(m.TradingRestriction, 5)); err != nil {
		return nil, err
	}

	// AgeRestriction ([]byte)
	if err := write(buf, pad(m.AgeRestriction, 5)); err != nil {
		return nil, err
	}

	// AdmissionType (string)
	if err := WriteFixedChar(buf, m.AdmissionType, 3); err != nil {
		return nil, err
	}

	// Venue (string)
	if err := WriteVarChar(buf, m.Venue, 255); err != nil {
		return nil, err
	}

	// Class (string)
	if err := WriteVarChar(buf, m.Class, 255); err != nil {
		return nil, err
	}

	// Area (string)
	if err := WriteVarChar(buf, m.Area, 255); err != nil {
		return nil, err
	}

	// Seat (string)
	if err := WriteVarChar(buf, m.Seat, 255); err != nil {
		return nil, err
	}

	// StartTimeDate (uint64)
	if err := write(buf, m.StartTimeDate); err != nil {
		return nil, err
	}

	// ValidFrom (uint64)
	if err := write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	// ExpirationTimestamp (uint64)
	if err := write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	// Description (string)
	if err := WriteVarChar(buf, m.Description, 16); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// write populates the fields in TicketAdmission from the byte slice
func (m *TicketAdmission) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	if err := read(buf, &m.Version); err != nil {
		return 0, err
	}

	// TradingRestriction ([]byte)
	m.TradingRestriction = make([]byte, 5)
	if err := readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	// AgeRestriction ([]byte)
	m.AgeRestriction = make([]byte, 5)
	if err := readLen(buf, m.AgeRestriction); err != nil {
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
		m.Venue, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// Class (string)
	{
		var err error
		m.Class, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// Area (string)
	{
		var err error
		m.Area, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// Seat (string)
	{
		var err error
		m.Seat, err = ReadVarChar(buf, 255)
		if err != nil {
			return 0, err
		}
	}

	// StartTimeDate (uint64)
	if err := read(buf, &m.StartTimeDate); err != nil {
		return 0, err
	}

	// ValidFrom (uint64)
	if err := read(buf, &m.ValidFrom); err != nil {
		return 0, err
	}

	// ExpirationTimestamp (uint64)
	if err := read(buf, &m.ExpirationTimestamp); err != nil {
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
	vals = append(vals, fmt.Sprintf("TradingRestriction:%#x", m.TradingRestriction))
	vals = append(vals, fmt.Sprintf("AgeRestriction:%#x", m.AgeRestriction))
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
