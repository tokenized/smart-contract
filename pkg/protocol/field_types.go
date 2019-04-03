package protocol

import (
	"bytes"
	"fmt"
)

// Administrator Administrator is used to refer to a Administration role in
// an Entity.
type Administrator struct {
	Type uint8  `json:"type,omitempty"` // Chairman, Director, Managing Partner, etc.. Found in 'Roles' in Specification/Resources
	Name string `json:"name,omitempty"` // Length 0-255 bytes. 0 is valid. Name (eg. John Alexander Smith)
}

// Serialize returns the byte representation of the message.
func (m Administrator) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Type (uint8)
	{
		if err := write(buf, m.Type); err != nil {
			return nil, err
		}
	}

	// Name (string)
	{
		if err := WriteVarChar(buf, m.Name, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *Administrator) Write(buf *bytes.Buffer) error {

	// Type (uint8)
	{
		if err := read(buf, &m.Type); err != nil {
			return err
		}
	}

	// Name (string)
	{
		var err error
		m.Name, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Administrator) Validate() error {

	// Type (uint8)
	{
		roles, err := GetRoles()
		if err != nil {
			return err
		}
		_, exists := roles[m.Type]
		if !exists {
			return fmt.Errorf("Invalid role value : %d", m.Type)
		}
	}

	// Name (string)
	{
		if len(m.Name) > (2<<8)-1 {
			return fmt.Errorf("varchar field Name too long %d/%d", len(m.Name), (2<<8)-1)
		}
	}

	return nil
}

func (m *Administrator) Equal(other Administrator) bool {

	// Type (uint8)
	if m.Type != other.Type {
		return false
	}

	// Name (string)
	if m.Name != other.Name {
		return false
	}
	return true
}

// AgeRestriction Age restriction is used to specify required ages for
// asset ownership.
type AgeRestriction struct {
	Lower uint8 `json:"lower,omitempty"` // The lowest age valid to own asset. Zero for no restriction.
	Upper uint8 `json:"upper,omitempty"` // The highest age valid to own asset. Zero for no restriction.
}

// Serialize returns the byte representation of the message.
func (m AgeRestriction) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Lower (uint8)
	{
		if err := write(buf, m.Lower); err != nil {
			return nil, err
		}
	}

	// Upper (uint8)
	{
		if err := write(buf, m.Upper); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *AgeRestriction) Write(buf *bytes.Buffer) error {

	// Lower (uint8)
	{
		if err := read(buf, &m.Lower); err != nil {
			return err
		}
	}

	// Upper (uint8)
	{
		if err := read(buf, &m.Upper); err != nil {
			return err
		}
	}

	return nil
}

func (m *AgeRestriction) Validate() error {

	// Lower (uint8)
	{
	}

	// Upper (uint8)
	{
	}

	return nil
}

func (m *AgeRestriction) Equal(other AgeRestriction) bool {

	// Lower (uint8)
	if m.Lower != other.Lower {
		return false
	}

	// Upper (uint8)
	if m.Upper != other.Upper {
		return false
	}
	return true
}

// Amendment An Amendment is used to describe the modification of a single
// field in a Contract or Asset, as defined in the ContractFormation and
// AssetCreation messages.
type Amendment struct {
	FieldIndex      uint8  `json:"field_index,omitempty"`      // Index of the field to be amended.
	Element         uint16 `json:"element,omitempty"`          // Specifies the element of the complex array type to be amended. This only applies to array types, and has no meaning for a simple type such as uint64, string, byte or byte[]. Specifying a value > 0 for a simple type will result in a Rejection.
	SubfieldIndex   uint8  `json:"subfield_index,omitempty"`   // Index of the subfield to be amended. This only applies to specific fields containing complex types with subfields. This is used to specify which field of the object the amendment applies to.
	SubfieldElement uint16 `json:"subfield_element,omitempty"` // Specifies the element of the complex array type to be amended. This only applies to array types, and has no meaning for a simple type such as uint64, string, byte or byte[]. Specifying a value > 0 for a simple type will result in a Rejection.
	Operation       uint8  `json:"operation,omitempty"`        // 0 = Modify. 1 = Add an element to the data to the array of elements. 2 = Delete the element listed in the Element field. The Add and Delete operations only apply to a particilar element of a complex array type. For example, it could be used to remove a particular VotingSystem from a Contract.
	Data            []byte `json:"data,omitempty"`             // New data for the amended subfield. Data type depends on the the type of the field being amended. The value should be serialize as defined by the protocol.
}

// Serialize returns the byte representation of the message.
func (m Amendment) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// FieldIndex (uint8)
	{
		if err := write(buf, m.FieldIndex); err != nil {
			return nil, err
		}
	}

	// Element (uint16)
	{
		if err := write(buf, m.Element); err != nil {
			return nil, err
		}
	}

	// SubfieldIndex (uint8)
	{
		if err := write(buf, m.SubfieldIndex); err != nil {
			return nil, err
		}
	}

	// SubfieldElement (uint16)
	{
		if err := write(buf, m.SubfieldElement); err != nil {
			return nil, err
		}
	}

	// Operation (uint8)
	{
		if err := write(buf, m.Operation); err != nil {
			return nil, err
		}
	}

	// Data ([]byte)
	{
		if err := WriteVarBin(buf, m.Data, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *Amendment) Write(buf *bytes.Buffer) error {

	// FieldIndex (uint8)
	{
		if err := read(buf, &m.FieldIndex); err != nil {
			return err
		}
	}

	// Element (uint16)
	{
		if err := read(buf, &m.Element); err != nil {
			return err
		}
	}

	// SubfieldIndex (uint8)
	{
		if err := read(buf, &m.SubfieldIndex); err != nil {
			return err
		}
	}

	// SubfieldElement (uint16)
	{
		if err := read(buf, &m.SubfieldElement); err != nil {
			return err
		}
	}

	// Operation (uint8)
	{
		if err := read(buf, &m.Operation); err != nil {
			return err
		}
	}

	// Data ([]byte)
	{
		var err error
		m.Data, err = ReadVarBin(buf, 32)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Amendment) Validate() error {

	// FieldIndex (uint8)
	{
	}

	// Element (uint16)
	{
	}

	// SubfieldIndex (uint8)
	{
	}

	// SubfieldElement (uint16)
	{
	}

	// Operation (uint8)
	{
	}

	// Data ([]byte)
	{
		if len(m.Data) > (2<<32)-1 {
			return fmt.Errorf("varbin field Data too long %d/%d", len(m.Data), (2<<32)-1)
		}
	}

	return nil
}

func (m *Amendment) Equal(other Amendment) bool {

	// FieldIndex (uint8)
	if m.FieldIndex != other.FieldIndex {
		return false
	}

	// Element (uint16)
	if m.Element != other.Element {
		return false
	}

	// SubfieldIndex (uint8)
	if m.SubfieldIndex != other.SubfieldIndex {
		return false
	}

	// SubfieldElement (uint16)
	if m.SubfieldElement != other.SubfieldElement {
		return false
	}

	// Operation (uint8)
	if m.Operation != other.Operation {
		return false
	}

	// Data ([]byte)
	if !bytes.Equal(m.Data, other.Data) {
		return false
	}
	return true
}

// AssetSettlement AssetSettlement is the data required to settle an asset
// transfer.
type AssetSettlement struct {
	ContractIndex uint16          `json:"contract_index,omitempty"` // Index of input containing the contract's address for this offset
	AssetType     string          `json:"asset_type,omitempty"`     // eg. Share, Bond, Ticket. All characters must be capitalised.
	AssetCode     AssetCode       `json:"asset_code,omitempty"`     // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Settlements   []QuantityIndex `json:"settlements,omitempty"`    // Each element contains the resulting token balance of Asset X for the output Address, which is referred to by the index.
}

// Serialize returns the byte representation of the message.
func (m AssetSettlement) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// ContractIndex (uint16)
	{
		if err := write(buf, m.ContractIndex); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	{
		if err := WriteFixedChar(buf, m.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := m.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Settlements ([]QuantityIndex)
	{
		if err := WriteVariableSize(buf, uint64(len(m.Settlements)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range m.Settlements {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func (m *AssetSettlement) Write(buf *bytes.Buffer) error {

	// ContractIndex (uint16)
	{
		if err := read(buf, &m.ContractIndex); err != nil {
			return err
		}
	}

	// AssetType (string)
	{
		var err error
		m.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := m.AssetCode.Write(buf); err != nil {
			return err
		}
	}

	// Settlements ([]QuantityIndex)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.Settlements = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.Settlements = append(m.Settlements, newValue)
		}
	}

	return nil
}

func (m *AssetSettlement) Validate() error {

	// ContractIndex (uint16)
	{
	}

	// AssetType (string)
	{
		if len(m.AssetType) > 3 {
			return fmt.Errorf("fixedchar field AssetType too long %d/%d", len(m.AssetType), 3)
		}
	}

	// AssetCode (AssetCode)
	{
		if err := m.AssetCode.Validate(); err != nil {
			return fmt.Errorf("field AssetCode is invalid : %s", err)
		}
	}

	// Settlements ([]QuantityIndex)
	{
		if len(m.Settlements) > (2<<0)-1 {
			return fmt.Errorf("list field Settlements has too many items %d/%d", len(m.Settlements), (2<<0)-1)
		}

		for i, value := range m.Settlements {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Settlements[%d] is invalid : %s", i, err)
			}
		}
	}

	return nil
}

func (m *AssetSettlement) Equal(other AssetSettlement) bool {

	// ContractIndex (uint16)
	if m.ContractIndex != other.ContractIndex {
		return false
	}

	// AssetType (string)
	if m.AssetType != other.AssetType {
		return false
	}

	// AssetCode (AssetCode)
	if !m.AssetCode.Equal(other.AssetCode) {
		return false
	}

	// Settlements ([]QuantityIndex)
	if len(m.Settlements) != 0 || len(other.Settlements) != 0 {
		if len(m.Settlements) != len(other.Settlements) {
			return false
		}

		for i, value := range m.Settlements {
			if !value.Equal(other.Settlements[i]) {
				return false
			}
		}
	}
	return true
}

// AssetTransfer AssetTransfer is the data required to transfer an asset.
type AssetTransfer struct {
	ContractIndex  uint16          `json:"contract_index,omitempty"`  // Index of output containing the contract's address for this offset
	AssetType      string          `json:"asset_type,omitempty"`      // eg. Share, Bond, Ticket. All characters must be capitalised.
	AssetCode      AssetCode       `json:"asset_code,omitempty"`      // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetSenders   []QuantityIndex `json:"asset_senders,omitempty"`   // Each element has the value of tokens to be spent from the input address, which is referred to by the index.
	AssetReceivers []TokenReceiver `json:"asset_receivers,omitempty"` // Each element has the value of tokens to be received by the output address, which is referred to by the index.
}

// Serialize returns the byte representation of the message.
func (m AssetTransfer) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// ContractIndex (uint16)
	{
		if err := write(buf, m.ContractIndex); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	{
		if err := WriteFixedChar(buf, m.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := m.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetSenders ([]QuantityIndex)
	{
		if err := WriteVariableSize(buf, uint64(len(m.AssetSenders)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range m.AssetSenders {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// AssetReceivers ([]TokenReceiver)
	{
		if err := WriteVariableSize(buf, uint64(len(m.AssetReceivers)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range m.AssetReceivers {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func (m *AssetTransfer) Write(buf *bytes.Buffer) error {

	// ContractIndex (uint16)
	{
		if err := read(buf, &m.ContractIndex); err != nil {
			return err
		}
	}

	// AssetType (string)
	{
		var err error
		m.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := m.AssetCode.Write(buf); err != nil {
			return err
		}
	}

	// AssetSenders ([]QuantityIndex)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.AssetSenders = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.AssetSenders = append(m.AssetSenders, newValue)
		}
	}

	// AssetReceivers ([]TokenReceiver)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.AssetReceivers = make([]TokenReceiver, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue TokenReceiver
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.AssetReceivers = append(m.AssetReceivers, newValue)
		}
	}

	return nil
}

func (m *AssetTransfer) Validate() error {

	// ContractIndex (uint16)
	{
	}

	// AssetType (string)
	{
		if len(m.AssetType) > 3 {
			return fmt.Errorf("fixedchar field AssetType too long %d/%d", len(m.AssetType), 3)
		}
	}

	// AssetCode (AssetCode)
	{
		if err := m.AssetCode.Validate(); err != nil {
			return fmt.Errorf("field AssetCode is invalid : %s", err)
		}
	}

	// AssetSenders ([]QuantityIndex)
	{
		if len(m.AssetSenders) > (2<<0)-1 {
			return fmt.Errorf("list field AssetSenders has too many items %d/%d", len(m.AssetSenders), (2<<0)-1)
		}

		for i, value := range m.AssetSenders {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field AssetSenders[%d] is invalid : %s", i, err)
			}
		}
	}

	// AssetReceivers ([]TokenReceiver)
	{
		if len(m.AssetReceivers) > (2<<0)-1 {
			return fmt.Errorf("list field AssetReceivers has too many items %d/%d", len(m.AssetReceivers), (2<<0)-1)
		}

		for i, value := range m.AssetReceivers {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field AssetReceivers[%d] is invalid : %s", i, err)
			}
		}
	}

	return nil
}

func (m *AssetTransfer) Equal(other AssetTransfer) bool {

	// ContractIndex (uint16)
	if m.ContractIndex != other.ContractIndex {
		return false
	}

	// AssetType (string)
	if m.AssetType != other.AssetType {
		return false
	}

	// AssetCode (AssetCode)
	if !m.AssetCode.Equal(other.AssetCode) {
		return false
	}

	// AssetSenders ([]QuantityIndex)
	if len(m.AssetSenders) != 0 || len(other.AssetSenders) != 0 {
		if len(m.AssetSenders) != len(other.AssetSenders) {
			return false
		}

		for i, value := range m.AssetSenders {
			if !value.Equal(other.AssetSenders[i]) {
				return false
			}
		}
	}

	// AssetReceivers ([]TokenReceiver)
	if len(m.AssetReceivers) != 0 || len(other.AssetReceivers) != 0 {
		if len(m.AssetReceivers) != len(other.AssetReceivers) {
			return false
		}

		for i, value := range m.AssetReceivers {
			if !value.Equal(other.AssetReceivers[i]) {
				return false
			}
		}
	}
	return true
}

// Entity Entity represents the details of a legal Entity, such as a public
// or private company, government agency, or and individual.
type Entity struct {
	Name                       string          `json:"name,omitempty"`                          // Length 1-255 bytes (0 is not valid). Issuing entity (company, organization, individual).  Can be any unique identifying string, including human readable names for branding/vanity purposes.
	Type                       byte            `json:"type,omitempty"`                          // P - Public Company Limited by Shares, C - Private Company Limited by Shares, I - Individual, L - Limited Partnership, U -Unlimited Partnership, T - Sole Proprietorship, S - Statutory Company, O - Non-Profit Organization, N - Nation State, G - Government Agency, U - Unit Trust, D - Discretionary Trust.  Found in 'Entities' (Specification/Resources).
	LEI                        string          `json:"lei,omitempty"`                           // Null is valid. A Legal Entity Identifier (or LEI) is an international identifier made up of a 20-character identifier that identifies distinct legal entities that engage in financial transactions. It is defined by ISO 17442.[1] Natural persons are not required to have an LEI; they’re eligible to have one issued, however, but only if they act in an independent business capacity.[2] The LEI is a global standard, designed to be non-proprietary data that is freely accessible to all.[3] As of December 2018, over 1,300,000 legal entities from more than 200 countries have now been issued with LEIs.
	AddressIncluded            bool            `json:"address_included,omitempty"`              // Registered/Physical/mailing address(HQ). Y-1/N-0, N means there is no issuer address.
	UnitNumber                 string          `json:"unit_number,omitempty"`                   // Issuer/Entity/Contracting Party X Address Details (eg. HQ)
	BuildingNumber             string          `json:"building_number,omitempty"`               //
	Street                     string          `json:"street,omitempty"`                        //
	SuburbCity                 string          `json:"suburb_city,omitempty"`                   //
	TerritoryStateProvinceCode string          `json:"territory_state_province_code,omitempty"` //
	CountryCode                string          `json:"country_code,omitempty"`                  //
	PostalZIPCode              string          `json:"postal_zip_code,omitempty"`               //
	EmailAddress               string          `json:"email_address,omitempty"`                 // Length 0-255 bytes. Address for text-based communication: eg. email address, Bitcoin address
	PhoneNumber                string          `json:"phone_number,omitempty"`                  // Length 0-50 bytes. 0 is valid. Phone Number for Entity.
	Administration             []Administrator `json:"administration,omitempty"`                // A list of people that are in Administrative Roles for the Entity.  eg. Chair, Director, Managing Partner, etc.
	Management                 []Manager       `json:"management,omitempty"`                    // A list of people in Management Roles for the Entity. e.g CEO, COO, CTO, CFO, Secretary, Executive, etc.
}

// Serialize returns the byte representation of the message.
func (m Entity) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	{
		if err := WriteVarChar(buf, m.Name, 8); err != nil {
			return nil, err
		}
	}

	// Type (byte)
	{
		if err := write(buf, m.Type); err != nil {
			return nil, err
		}
	}

	// LEI (string)
	{
		if err := WriteFixedChar(buf, m.LEI, 20); err != nil {
			return nil, err
		}
	}

	// AddressIncluded (bool)
	{
		if err := write(buf, m.AddressIncluded); err != nil {
			return nil, err
		}
	}

	// UnitNumber (string)
	if m.AddressIncluded {
		if err := WriteVarChar(buf, m.UnitNumber, 8); err != nil {
			return nil, err
		}
	}

	// BuildingNumber (string)
	if m.AddressIncluded {
		if err := WriteVarChar(buf, m.BuildingNumber, 8); err != nil {
			return nil, err
		}
	}

	// Street (string)
	if m.AddressIncluded {
		if err := WriteVarChar(buf, m.Street, 16); err != nil {
			return nil, err
		}
	}

	// SuburbCity (string)
	if m.AddressIncluded {
		if err := WriteVarChar(buf, m.SuburbCity, 8); err != nil {
			return nil, err
		}
	}

	// TerritoryStateProvinceCode (string)
	if m.AddressIncluded {
		if err := WriteFixedChar(buf, m.TerritoryStateProvinceCode, 5); err != nil {
			return nil, err
		}
	}

	// CountryCode (string)
	if m.AddressIncluded {
		if err := WriteFixedChar(buf, m.CountryCode, 3); err != nil {
			return nil, err
		}
	}

	// PostalZIPCode (string)
	if m.AddressIncluded {
		if err := WriteFixedChar(buf, m.PostalZIPCode, 12); err != nil {
			return nil, err
		}
	}

	// EmailAddress (string)
	{
		if err := WriteVarChar(buf, m.EmailAddress, 8); err != nil {
			return nil, err
		}
	}

	// PhoneNumber (string)
	{
		if err := WriteVarChar(buf, m.PhoneNumber, 8); err != nil {
			return nil, err
		}
	}

	// Administration ([]Administrator)
	{
		if err := WriteVariableSize(buf, uint64(len(m.Administration)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range m.Administration {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Management ([]Manager)
	{
		if err := WriteVariableSize(buf, uint64(len(m.Management)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range m.Management {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

func (m *Entity) Write(buf *bytes.Buffer) error {

	// Name (string)
	{
		var err error
		m.Name, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// Type (byte)
	{
		if err := read(buf, &m.Type); err != nil {
			return err
		}
	}

	// LEI (string)
	{
		var err error
		m.LEI, err = ReadFixedChar(buf, 20)
		if err != nil {
			return err
		}
	}

	// AddressIncluded (bool)
	{
		if err := read(buf, &m.AddressIncluded); err != nil {
			return err
		}
	}

	// UnitNumber (string)
	if m.AddressIncluded {
		var err error
		m.UnitNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// BuildingNumber (string)
	if m.AddressIncluded {
		var err error
		m.BuildingNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// Street (string)
	if m.AddressIncluded {
		var err error
		m.Street, err = ReadVarChar(buf, 16)
		if err != nil {
			return err
		}
	}

	// SuburbCity (string)
	if m.AddressIncluded {
		var err error
		m.SuburbCity, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// TerritoryStateProvinceCode (string)
	if m.AddressIncluded {
		var err error
		m.TerritoryStateProvinceCode, err = ReadFixedChar(buf, 5)
		if err != nil {
			return err
		}
	}

	// CountryCode (string)
	if m.AddressIncluded {
		var err error
		m.CountryCode, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// PostalZIPCode (string)
	if m.AddressIncluded {
		var err error
		m.PostalZIPCode, err = ReadFixedChar(buf, 12)
		if err != nil {
			return err
		}
	}

	// EmailAddress (string)
	{
		var err error
		m.EmailAddress, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// PhoneNumber (string)
	{
		var err error
		m.PhoneNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// Administration ([]Administrator)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.Administration = make([]Administrator, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Administrator
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.Administration = append(m.Administration, newValue)
		}
	}

	// Management ([]Manager)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.Management = make([]Manager, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Manager
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.Management = append(m.Management, newValue)
		}
	}

	return nil
}

func (m *Entity) Validate() error {

	// Name (string)
	{
		if len(m.Name) > (2<<8)-1 {
			return fmt.Errorf("varchar field Name too long %d/%d", len(m.Name), (2<<8)-1)
		}
	}

	// Type (byte)
	{
	}

	// LEI (string)
	{
		if len(m.LEI) > 20 {
			return fmt.Errorf("fixedchar field LEI too long %d/%d", len(m.LEI), 20)
		}
	}

	// AddressIncluded (bool)
	{
	}

	// UnitNumber (string)
	if m.AddressIncluded {
		if len(m.UnitNumber) > (2<<8)-1 {
			return fmt.Errorf("varchar field UnitNumber too long %d/%d", len(m.UnitNumber), (2<<8)-1)
		}
	}

	// BuildingNumber (string)
	if m.AddressIncluded {
		if len(m.BuildingNumber) > (2<<8)-1 {
			return fmt.Errorf("varchar field BuildingNumber too long %d/%d", len(m.BuildingNumber), (2<<8)-1)
		}
	}

	// Street (string)
	if m.AddressIncluded {
		if len(m.Street) > (2<<16)-1 {
			return fmt.Errorf("varchar field Street too long %d/%d", len(m.Street), (2<<16)-1)
		}
	}

	// SuburbCity (string)
	if m.AddressIncluded {
		if len(m.SuburbCity) > (2<<8)-1 {
			return fmt.Errorf("varchar field SuburbCity too long %d/%d", len(m.SuburbCity), (2<<8)-1)
		}
	}

	// TerritoryStateProvinceCode (string)
	if m.AddressIncluded {
		if len(m.TerritoryStateProvinceCode) > 5 {
			return fmt.Errorf("fixedchar field TerritoryStateProvinceCode too long %d/%d", len(m.TerritoryStateProvinceCode), 5)
		}
	}

	// CountryCode (string)
	if m.AddressIncluded {
		if len(m.CountryCode) > 3 {
			return fmt.Errorf("fixedchar field CountryCode too long %d/%d", len(m.CountryCode), 3)
		}
	}

	// PostalZIPCode (string)
	if m.AddressIncluded {
		if len(m.PostalZIPCode) > 12 {
			return fmt.Errorf("fixedchar field PostalZIPCode too long %d/%d", len(m.PostalZIPCode), 12)
		}
	}

	// EmailAddress (string)
	{
		if len(m.EmailAddress) > (2<<8)-1 {
			return fmt.Errorf("varchar field EmailAddress too long %d/%d", len(m.EmailAddress), (2<<8)-1)
		}
	}

	// PhoneNumber (string)
	{
		if len(m.PhoneNumber) > (2<<8)-1 {
			return fmt.Errorf("varchar field PhoneNumber too long %d/%d", len(m.PhoneNumber), (2<<8)-1)
		}
	}

	// Administration ([]Administrator)
	{
		if len(m.Administration) > (2<<0)-1 {
			return fmt.Errorf("list field Administration has too many items %d/%d", len(m.Administration), (2<<0)-1)
		}

		for i, value := range m.Administration {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Administration[%d] is invalid : %s", i, err)
			}
		}
	}

	// Management ([]Manager)
	{
		if len(m.Management) > (2<<0)-1 {
			return fmt.Errorf("list field Management has too many items %d/%d", len(m.Management), (2<<0)-1)
		}

		for i, value := range m.Management {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Management[%d] is invalid : %s", i, err)
			}
		}
	}

	return nil
}

func (m *Entity) Equal(other Entity) bool {

	// Name (string)
	if m.Name != other.Name {
		return false
	}

	// Type (byte)
	if m.Type != other.Type {
		return false
	}

	// LEI (string)
	if m.LEI != other.LEI {
		return false
	}

	// AddressIncluded (bool)
	if m.AddressIncluded != other.AddressIncluded {
		return false
	}

	// UnitNumber (string)
	if m.UnitNumber != other.UnitNumber {
		return false
	}

	// BuildingNumber (string)
	if m.BuildingNumber != other.BuildingNumber {
		return false
	}

	// Street (string)
	if m.Street != other.Street {
		return false
	}

	// SuburbCity (string)
	if m.SuburbCity != other.SuburbCity {
		return false
	}

	// TerritoryStateProvinceCode (string)
	if m.TerritoryStateProvinceCode != other.TerritoryStateProvinceCode {
		return false
	}

	// CountryCode (string)
	if m.CountryCode != other.CountryCode {
		return false
	}

	// PostalZIPCode (string)
	if m.PostalZIPCode != other.PostalZIPCode {
		return false
	}

	// EmailAddress (string)
	if m.EmailAddress != other.EmailAddress {
		return false
	}

	// PhoneNumber (string)
	if m.PhoneNumber != other.PhoneNumber {
		return false
	}

	// Administration ([]Administrator)
	if len(m.Administration) != 0 || len(other.Administration) != 0 {
		if len(m.Administration) != len(other.Administration) {
			return false
		}

		for i, value := range m.Administration {
			if !value.Equal(other.Administration[i]) {
				return false
			}
		}
	}

	// Management ([]Manager)
	if len(m.Management) != 0 || len(other.Management) != 0 {
		if len(m.Management) != len(other.Management) {
			return false
		}

		for i, value := range m.Management {
			if !value.Equal(other.Management[i]) {
				return false
			}
		}
	}
	return true
}

// Header Header contains common details required by every Tokenized
// message.
type Header struct {
}

// Serialize returns the byte representation of the message.
func (m Header) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	return buf.Bytes(), nil
}

func (m *Header) Write(buf *bytes.Buffer) error {
	return nil
}

func (m *Header) Validate() error {
	return nil
}

func (m *Header) Equal(other Header) bool {
	return true
}

// Manager Manager is used to refer to a role that is responsible for the
// Management of an Entity.
type Manager struct {
	Type uint8  `json:"type,omitempty"` // CEO, COO, CFO, etc. Found in 'Roles' in Specification/Resources
	Name string `json:"name,omitempty"` // Length 0-255 bytes. 0 is valid. Name (eg. John Alexander Smith)
}

// Serialize returns the byte representation of the message.
func (m Manager) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Type (uint8)
	{
		if err := write(buf, m.Type); err != nil {
			return nil, err
		}
	}

	// Name (string)
	{
		if err := WriteVarChar(buf, m.Name, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *Manager) Write(buf *bytes.Buffer) error {

	// Type (uint8)
	{
		if err := read(buf, &m.Type); err != nil {
			return err
		}
	}

	// Name (string)
	{
		var err error
		m.Name, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Validate() error {

	// Type (uint8)
	{
		roles, err := GetRoles()
		if err != nil {
			return err
		}
		_, exists := roles[m.Type]
		if !exists {
			return fmt.Errorf("Invalid role value : %d", m.Type)
		}
	}

	// Name (string)
	{
		if len(m.Name) > (2<<8)-1 {
			return fmt.Errorf("varchar field Name too long %d/%d", len(m.Name), (2<<8)-1)
		}
	}

	return nil
}

func (m *Manager) Equal(other Manager) bool {

	// Type (uint8)
	if m.Type != other.Type {
		return false
	}

	// Name (string)
	if m.Name != other.Name {
		return false
	}
	return true
}

// QuantityIndex A QuantityIndex contains a quantity, and an index. The
// quantity could be used to describe a number of tokens, or a value. The
// index is used to refer to an input index position.
type QuantityIndex struct {
	Index    uint16 `json:"index,omitempty"`    // The index of the input sending the tokens
	Quantity uint64 `json:"quantity,omitempty"` // Number of tokens being sent
}

// Serialize returns the byte representation of the message.
func (m QuantityIndex) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Index (uint16)
	{
		if err := write(buf, m.Index); err != nil {
			return nil, err
		}
	}

	// Quantity (uint64)
	{
		if err := write(buf, m.Quantity); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *QuantityIndex) Write(buf *bytes.Buffer) error {

	// Index (uint16)
	{
		if err := read(buf, &m.Index); err != nil {
			return err
		}
	}

	// Quantity (uint64)
	{
		if err := read(buf, &m.Quantity); err != nil {
			return err
		}
	}

	return nil
}

func (m *QuantityIndex) Validate() error {

	// Index (uint16)
	{
	}

	// Quantity (uint64)
	{
	}

	return nil
}

func (m *QuantityIndex) Equal(other QuantityIndex) bool {

	// Index (uint16)
	if m.Index != other.Index {
		return false
	}

	// Quantity (uint64)
	if m.Quantity != other.Quantity {
		return false
	}
	return true
}

// Registry A Registry defines the details of a public Registry.
type Registry struct {
	Name      string `json:"name,omitempty"`       // Length 0-255 bytes. 0 is valid. Registry X Name (eg. Coinbase, Tokenized, etc.)
	URL       string `json:"url,omitempty"`        // Length 0-255 bytes. 0 is valid. If applicable: URL for REST/RPC Endpoint
	PublicKey []byte `json:"public_key,omitempty"` // Length 0-255 bytes. 0 is not valid. Registry Public Key (eg. Bitcoin Public key), used to confirm digital signed proofs for transfers.  Can also be the same public address that controls a Tokenized Registry.
}

// Serialize returns the byte representation of the message.
func (m Registry) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	{
		if err := WriteVarChar(buf, m.Name, 8); err != nil {
			return nil, err
		}
	}

	// URL (string)
	{
		if err := WriteVarChar(buf, m.URL, 8); err != nil {
			return nil, err
		}
	}

	// PublicKey ([]byte)
	{
		if err := WriteVarBin(buf, m.PublicKey, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *Registry) Write(buf *bytes.Buffer) error {

	// Name (string)
	{
		var err error
		m.Name, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// URL (string)
	{
		var err error
		m.URL, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// PublicKey ([]byte)
	{
		var err error
		m.PublicKey, err = ReadVarBin(buf, 8)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Registry) Validate() error {

	// Name (string)
	{
		if len(m.Name) > (2<<8)-1 {
			return fmt.Errorf("varchar field Name too long %d/%d", len(m.Name), (2<<8)-1)
		}
	}

	// URL (string)
	{
		if len(m.URL) > (2<<8)-1 {
			return fmt.Errorf("varchar field URL too long %d/%d", len(m.URL), (2<<8)-1)
		}
	}

	// PublicKey ([]byte)
	{
		if len(m.PublicKey) > (2<<8)-1 {
			return fmt.Errorf("varbin field PublicKey too long %d/%d", len(m.PublicKey), (2<<8)-1)
		}
	}

	return nil
}

func (m *Registry) Equal(other Registry) bool {

	// Name (string)
	if m.Name != other.Name {
		return false
	}

	// URL (string)
	if m.URL != other.URL {
		return false
	}

	// PublicKey ([]byte)
	if !bytes.Equal(m.PublicKey, other.PublicKey) {
		return false
	}
	return true
}

// TargetAddress A TargetAddress defines a public address and quantity.
type TargetAddress struct {
	Address  PublicKeyHash `json:"address,omitempty"`  // Public address where the token balance will be changed.
	Quantity uint64        `json:"quantity,omitempty"` // Qty of tokens to be frozen, thawed, confiscated or reconciled. For Contract-wide freezes 0 will be used.
}

// Serialize returns the byte representation of the message.
func (m TargetAddress) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Address (PublicKeyHash)
	{
		b, err := m.Address.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Quantity (uint64)
	{
		if err := write(buf, m.Quantity); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *TargetAddress) Write(buf *bytes.Buffer) error {

	// Address (PublicKeyHash)
	{
		if err := m.Address.Write(buf); err != nil {
			return err
		}
	}

	// Quantity (uint64)
	{
		if err := read(buf, &m.Quantity); err != nil {
			return err
		}
	}

	return nil
}

func (m *TargetAddress) Validate() error {

	// Address (PublicKeyHash)
	{
		if err := m.Address.Validate(); err != nil {
			return fmt.Errorf("field Address is invalid : %s", err)
		}
	}

	// Quantity (uint64)
	{
	}

	return nil
}

func (m *TargetAddress) Equal(other TargetAddress) bool {

	// Address (PublicKeyHash)
	if !m.Address.Equal(other.Address) {
		return false
	}

	// Quantity (uint64)
	if m.Quantity != other.Quantity {
		return false
	}
	return true
}

// TokenReceiver A TokenReceiver is contains a quantity, index, and
// registry token deails. The quantity could be used to describe a number
// of tokens, or a value. The index is used to refer to an input index
// position. The registry token details include the type of algorithm, and
// the token.
type TokenReceiver struct {
	Index                        uint16 `json:"index,omitempty"`                           // The index of the output receiving the tokens
	Quantity                     uint64 `json:"quantity,omitempty"`                        // Number of tokens to be received by address at Output X
	RegistrySigAlgorithm         uint8  `json:"registry_sig_algorithm,omitempty"`          // 0 = No Registry-signed Message, 1 = ECDSA+secp256k1
	RegistryConfirmationSigToken string `json:"registry_confirmation_sig_token,omitempty"` // Length 0-255 bytes. IF restricted to a registry (whitelist) or has transfer restrictions (age, location, investor status): ECDSA+secp256k1 (or the like) signed message provided by an approved/trusted registry through an API signature of [Contract Address + Asset Code + Public Address + Blockhash of the Latest Block + Block Height + Confirmed/Rejected Bool]. If no transfer restrictions(trade restriction/age restriction fields in the Asset Type payload. or restricted to a whitelist by the Contract Auth Flags, it is a NULL field.
}

// Serialize returns the byte representation of the message.
func (m TokenReceiver) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Index (uint16)
	{
		if err := write(buf, m.Index); err != nil {
			return nil, err
		}
	}

	// Quantity (uint64)
	{
		if err := write(buf, m.Quantity); err != nil {
			return nil, err
		}
	}

	// RegistrySigAlgorithm (uint8)
	{
		if err := write(buf, m.RegistrySigAlgorithm); err != nil {
			return nil, err
		}
	}

	// RegistryConfirmationSigToken (string)
	{
		if err := WriteVarChar(buf, m.RegistryConfirmationSigToken, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *TokenReceiver) Write(buf *bytes.Buffer) error {

	// Index (uint16)
	{
		if err := read(buf, &m.Index); err != nil {
			return err
		}
	}

	// Quantity (uint64)
	{
		if err := read(buf, &m.Quantity); err != nil {
			return err
		}
	}

	// RegistrySigAlgorithm (uint8)
	{
		if err := read(buf, &m.RegistrySigAlgorithm); err != nil {
			return err
		}
	}

	// RegistryConfirmationSigToken (string)
	{
		var err error
		m.RegistryConfirmationSigToken, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *TokenReceiver) Validate() error {

	// Index (uint16)
	{
	}

	// Quantity (uint64)
	{
	}

	// RegistrySigAlgorithm (uint8)
	{
	}

	// RegistryConfirmationSigToken (string)
	{
		if len(m.RegistryConfirmationSigToken) > (2<<8)-1 {
			return fmt.Errorf("varchar field RegistryConfirmationSigToken too long %d/%d", len(m.RegistryConfirmationSigToken), (2<<8)-1)
		}
	}

	return nil
}

func (m *TokenReceiver) Equal(other TokenReceiver) bool {

	// Index (uint16)
	if m.Index != other.Index {
		return false
	}

	// Quantity (uint64)
	if m.Quantity != other.Quantity {
		return false
	}

	// RegistrySigAlgorithm (uint8)
	if m.RegistrySigAlgorithm != other.RegistrySigAlgorithm {
		return false
	}

	// RegistryConfirmationSigToken (string)
	if m.RegistryConfirmationSigToken != other.RegistryConfirmationSigToken {
		return false
	}
	return true
}

// VotingSystem A VotingSystem defines all details of a Voting System.
type VotingSystem struct {
	Name                    string `json:"name,omitempty"`                      // eg. Special Resolutions, Ordinary Resolutions, Fundamental Matters, General Matters, Directors' Vote, Poll, etc.
	VoteType                byte   `json:"vote_type,omitempty"`                 // R - Relative Threshold, A - Absolute Threshold, P - Plurality,  (Relative Threshold means the number of counted votes must exceed the threshold % of total ballots cast.  Abstentations/spoiled votes do not detract from the liklihood of a vote passing as they are not included in the denominator.  Absolute Threshold requires the number of ballots counted to exceed the threshold value when compared to the total outstanding tokens.  Abstentations/spoiled votes detract from the liklihood of the vote passing.  For example, in an absolute threshold vote, if the threshold was 50% and 51% of the total outstanding tokens did not vote, then the vote cannot pass.  50% of all tokens would have had to vote for one vote option for the vote to be successful.
	TallyLogic              byte   `json:"tally_logic,omitempty"`               // 0 - Standard Scoring (+1 * # of tokens owned), 1 - Weighted Scoring (1st choice * Vote Max * # of tokens held, 2nd choice * Vote Max-1 * # of tokens held,..etc.)
	ThresholdPercentage     uint8  `json:"threshold_percentage,omitempty"`      // This field is ignored when VoteType is P (Plurality). Otherwise it is the percentage of ballots required for a proposal to pass. Valid values are greater than 0 and less than 100. 75 means 75% and greater.  Only applicable to Relative and Absolute Threshold vote methods.  The Plurality vote method requires no threshold value (NULL), as the successful vote option is simply selected on the basis of highest ballots cast for it.
	VoteMultiplierPermitted bool   `json:"vote_multiplier_permitted,omitempty"` // Where an asset has a vote multiplier, true must be set here for the vote multiplier to count, otherwise votes are simply treated as 1x per token.
	HolderProposalFee       uint64 `json:"holder_proposal_fee,omitempty"`       // Token Owners must pay the fee amount to broadcast a valid Proposal.  If the Proposal action is valid, the smart contract will start a vote. 0 is valid.
}

// Serialize returns the byte representation of the message.
func (m VotingSystem) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	{
		if err := WriteVarChar(buf, m.Name, 8); err != nil {
			return nil, err
		}
	}

	// VoteType (byte)
	{
		if err := write(buf, m.VoteType); err != nil {
			return nil, err
		}
	}

	// TallyLogic (byte)
	{
		if err := write(buf, m.TallyLogic); err != nil {
			return nil, err
		}
	}

	// ThresholdPercentage (uint8)
	{
		if err := write(buf, m.ThresholdPercentage); err != nil {
			return nil, err
		}
	}

	// VoteMultiplierPermitted (bool)
	{
		if err := write(buf, m.VoteMultiplierPermitted); err != nil {
			return nil, err
		}
	}

	// HolderProposalFee (uint64)
	{
		if err := write(buf, m.HolderProposalFee); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (m *VotingSystem) Write(buf *bytes.Buffer) error {

	// Name (string)
	{
		var err error
		m.Name, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// VoteType (byte)
	{
		if err := read(buf, &m.VoteType); err != nil {
			return err
		}
	}

	// TallyLogic (byte)
	{
		if err := read(buf, &m.TallyLogic); err != nil {
			return err
		}
	}

	// ThresholdPercentage (uint8)
	{
		if err := read(buf, &m.ThresholdPercentage); err != nil {
			return err
		}
	}

	// VoteMultiplierPermitted (bool)
	{
		if err := read(buf, &m.VoteMultiplierPermitted); err != nil {
			return err
		}
	}

	// HolderProposalFee (uint64)
	{
		if err := read(buf, &m.HolderProposalFee); err != nil {
			return err
		}
	}

	return nil
}

func (m *VotingSystem) Validate() error {

	// Name (string)
	{
		if len(m.Name) > (2<<8)-1 {
			return fmt.Errorf("varchar field Name too long %d/%d", len(m.Name), (2<<8)-1)
		}
	}

	// VoteType (byte)
	{
	}

	// TallyLogic (byte)
	{
	}

	// ThresholdPercentage (uint8)
	{
	}

	// VoteMultiplierPermitted (bool)
	{
	}

	// HolderProposalFee (uint64)
	{
	}

	return nil
}

func (m *VotingSystem) Equal(other VotingSystem) bool {

	// Name (string)
	if m.Name != other.Name {
		return false
	}

	// VoteType (byte)
	if m.VoteType != other.VoteType {
		return false
	}

	// TallyLogic (byte)
	if m.TallyLogic != other.TallyLogic {
		return false
	}

	// ThresholdPercentage (uint8)
	if m.ThresholdPercentage != other.ThresholdPercentage {
		return false
	}

	// VoteMultiplierPermitted (bool)
	if m.VoteMultiplierPermitted != other.VoteMultiplierPermitted {
		return false
	}

	// HolderProposalFee (uint64)
	if m.HolderProposalFee != other.HolderProposalFee {
		return false
	}
	return true
}
