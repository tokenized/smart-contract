package protocol

import "bytes"

// The code in this file is auto-generated. Do not edit it by hand as it will
// be overwritten when code is regenerated.

const (
	// AssetTypeLen is the size in bytes of all asset type variants.
	AssetTypeLen = 152

	// CodeAssetTypeCoupon identifies data as a Coupon message.
	CodeAssetTypeCoupon = "COU"

	// CodeAssetTypeMovieTicket identifies data as a Movie Ticket message.
	CodeAssetTypeMovieTicket = "MOV"

	// CodeAssetTypeShareCommon identifies data as a Share - Common message.
	CodeAssetTypeShareCommon = "SHC"

	// CodeAssetTypeTicketAdmission identifies data as a Ticket (Admission) message.
	CodeAssetTypeTicketAdmission = "TIC"
)

// AssetTypeCoupon asset type.
type AssetTypeCoupon struct {
	BaseMessage

	Version            uint8
	TradingRestriction []byte
	RedeemingEntity    []byte
	ExpiryDate         uint64
	IssueDate          uint64
	Description        []byte
}

// NewAssetTypeCoupon returns a new AssetTypeCoupon.
func NewAssetTypeCoupon() *AssetTypeCoupon {
	return &AssetTypeCoupon{}
}

// Type returns the type identifer for this message.
func (m AssetTypeCoupon) Type() string {
	return CodeAssetTypeCoupon
}

// Len returns the byte size of this message.
func (m AssetTypeCoupon) Len() int64 {
	return AssetTypeLen
}

// Bytes returns the message in bytes.
func (m AssetTypeCoupon) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.TradingRestriction, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.RedeemingEntity, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExpiryDate); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.IssueDate); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Description, 93)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetTypeCoupon) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	if err := m.read(buf, &m.Version); err != nil {
		return 0, err
	}

	m.TradingRestriction = make([]byte, 3)
	if err := m.readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	m.TradingRestriction = bytes.Trim(m.TradingRestriction, "\x00")

	m.RedeemingEntity = make([]byte, 32)
	if err := m.readLen(buf, m.RedeemingEntity); err != nil {
		return 0, err
	}

	m.RedeemingEntity = bytes.Trim(m.RedeemingEntity, "\x00")

	if err := m.read(buf, &m.ExpiryDate); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.IssueDate); err != nil {
		return 0, err
	}

	m.Description = make([]byte, 93)
	if err := m.readLen(buf, m.Description); err != nil {
		return 0, err
	}

	m.Description = bytes.Trim(m.Description, "\x00")

	return int(m.Len()), nil
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetTypeCoupon) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// AssetTypeMovieTicket asset type.
type AssetTypeMovieTicket struct {
	BaseMessage

	Version             uint8
	TradingRestriction  []byte
	AgeRestriction      []byte
	Venue               []byte
	ValidFrom           uint64
	ExpirationTimestamp uint64
	Description         []byte
}

// NewAssetTypeMovieTicket returns a new AssetTypeMovieTicket.
func NewAssetTypeMovieTicket() *AssetTypeMovieTicket {
	return &AssetTypeMovieTicket{}
}

// Type returns the type identifer for this message.
func (m AssetTypeMovieTicket) Type() string {
	return CodeAssetTypeMovieTicket
}

// Len returns the byte size of this message.
func (m AssetTypeMovieTicket) Len() int64 {
	return AssetTypeLen
}

// Bytes returns the message in bytes.
func (m AssetTypeMovieTicket) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.TradingRestriction, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AgeRestriction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Venue, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Description, 88)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetTypeMovieTicket) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	if err := m.read(buf, &m.Version); err != nil {
		return 0, err
	}

	m.TradingRestriction = make([]byte, 3)
	if err := m.readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	m.TradingRestriction = bytes.Trim(m.TradingRestriction, "\x00")

	m.AgeRestriction = make([]byte, 5)
	if err := m.readLen(buf, m.AgeRestriction); err != nil {
		return 0, err
	}

	m.AgeRestriction = bytes.Trim(m.AgeRestriction, "\x00")

	m.Venue = make([]byte, 32)
	if err := m.readLen(buf, m.Venue); err != nil {
		return 0, err
	}

	m.Venue = bytes.Trim(m.Venue, "\x00")

	if err := m.read(buf, &m.ValidFrom); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.ExpirationTimestamp); err != nil {
		return 0, err
	}

	m.Description = make([]byte, 88)
	if err := m.readLen(buf, m.Description); err != nil {
		return 0, err
	}

	m.Description = bytes.Trim(m.Description, "\x00")

	return int(m.Len()), nil
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetTypeMovieTicket) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// AssetTypeShareCommon asset type.
type AssetTypeShareCommon struct {
	BaseMessage

	Version              uint8
	TradingRestriction   []byte
	DividendType         byte
	DividendVar          float32
	DividendFixed        float32
	DistributionInterval byte
	Guaranteed           byte
	Ticker               []byte
	ISIN                 []byte
	Description          []byte
}

// NewAssetTypeShareCommon returns a new AssetTypeShareCommon.
func NewAssetTypeShareCommon() *AssetTypeShareCommon {
	return &AssetTypeShareCommon{}
}

// Type returns the type identifer for this message.
func (m AssetTypeShareCommon) Type() string {
	return CodeAssetTypeShareCommon
}

// Len returns the byte size of this message.
func (m AssetTypeShareCommon) Len() int64 {
	return AssetTypeLen
}

// Bytes returns the message in bytes.
func (m AssetTypeShareCommon) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.TradingRestriction, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DividendType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DividendVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DividendFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DistributionInterval); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Guaranteed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Ticker, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ISIN, 12)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Description, 113)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetTypeShareCommon) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	if err := m.read(buf, &m.Version); err != nil {
		return 0, err
	}

	m.TradingRestriction = make([]byte, 3)
	if err := m.readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	m.TradingRestriction = bytes.Trim(m.TradingRestriction, "\x00")

	if err := m.read(buf, &m.DividendType); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.DividendVar); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.DividendFixed); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.DistributionInterval); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.Guaranteed); err != nil {
		return 0, err
	}

	m.Ticker = make([]byte, 5)
	if err := m.readLen(buf, m.Ticker); err != nil {
		return 0, err
	}

	m.Ticker = bytes.Trim(m.Ticker, "\x00")

	m.ISIN = make([]byte, 12)
	if err := m.readLen(buf, m.ISIN); err != nil {
		return 0, err
	}

	m.ISIN = bytes.Trim(m.ISIN, "\x00")

	m.Description = make([]byte, 113)
	if err := m.readLen(buf, m.Description); err != nil {
		return 0, err
	}

	m.Description = bytes.Trim(m.Description, "\x00")

	return int(m.Len()), nil
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetTypeShareCommon) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// AssetTypeTicketAdmission asset type.
type AssetTypeTicketAdmission struct {
	BaseMessage

	Version             uint8
	TradingRestriction  []byte
	AgeRestriction      []byte
	ValidFrom           uint64
	ExpirationTimestamp uint64
	Description         []byte
}

// NewAssetTypeTicketAdmission returns a new AssetTypeTicketAdmission.
func NewAssetTypeTicketAdmission() *AssetTypeTicketAdmission {
	return &AssetTypeTicketAdmission{}
}

// Type returns the type identifer for this message.
func (m AssetTypeTicketAdmission) Type() string {
	return CodeAssetTypeTicketAdmission
}

// Len returns the byte size of this message.
func (m AssetTypeTicketAdmission) Len() int64 {
	return AssetTypeLen
}

// Bytes returns the message in bytes.
func (m AssetTypeTicketAdmission) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.TradingRestriction, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AgeRestriction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ValidFrom); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExpirationTimestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Description, 120)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetTypeTicketAdmission) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	if err := m.read(buf, &m.Version); err != nil {
		return 0, err
	}

	m.TradingRestriction = make([]byte, 3)
	if err := m.readLen(buf, m.TradingRestriction); err != nil {
		return 0, err
	}

	m.TradingRestriction = bytes.Trim(m.TradingRestriction, "\x00")

	m.AgeRestriction = make([]byte, 5)
	if err := m.readLen(buf, m.AgeRestriction); err != nil {
		return 0, err
	}

	m.AgeRestriction = bytes.Trim(m.AgeRestriction, "\x00")

	if err := m.read(buf, &m.ValidFrom); err != nil {
		return 0, err
	}

	if err := m.read(buf, &m.ExpirationTimestamp); err != nil {
		return 0, err
	}

	m.Description = make([]byte, 120)
	if err := m.readLen(buf, m.Description); err != nil {
		return 0, err
	}

	m.Description = bytes.Trim(m.Description, "\x00")

	return int(m.Len()), nil
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetTypeTicketAdmission) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}
