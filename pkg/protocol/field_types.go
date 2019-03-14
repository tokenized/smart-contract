package protocol

import "bytes"

// Address Address represents a public address
type Address struct {
	PKH [20]byte `json:"pkh,omitempty"` // Public address where the token balance will be changed.
}

// NewAddress returns a new Address with defaults set.
func NewAddress() *Address {
	result := Address{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m Address) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// PKH ([20]byte)
	if err := write(buf, m.PKH); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *Address) Write(buf *bytes.Buffer) error {

	// PKH ([20]byte)
	if err := read(buf, &m.PKH); err != nil {
		return err
	}
	return nil
}

// Amendment An Amendment is used to describe the modification of a single
// field in a Contract or Asset, as defined in the ContractFormation and
// AssetCreation messages.
type Amendment struct {
	FieldIndex    uint8  `json:"field_index,omitempty"`    // Index of the field to be amended.
	Element       uint16 `json:"element,omitempty"`        // Specifies the element of the complex array type to be amended. This only applies to array types, and has no meaning for a simple type such as uint64, string, byte or byte[]. Specifying a value > 0 for a simple type will result in a Rejection.
	SubfieldIndex uint8  `json:"subfield_index,omitempty"` // Index of the subfield to be amended. This only applies to specific fields of an element in an array. This is used to specify which field of the array element the amendment applies to.
	Operation     uint8  `json:"operation,omitempty"`      // 0 = Modify. 1 = Add an element to the data to the array of elements. 2 = Delete the element listed in the Element field. The Add and Delete operations only apply to a particilar element of a complex array type. For example, it could be used to remove a particular VotingSystem from a Contract.
	Data          string `json:"data,omitempty"`           // New data for the amended subfield. Data type depends on the the type of the field being amended.
}

// NewAmendment returns a new Amendment with defaults set.
func NewAmendment() *Amendment {
	result := Amendment{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m Amendment) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// FieldIndex (uint8)
	if err := write(buf, m.FieldIndex); err != nil {
		return nil, err
	}

	// Element (uint16)
	if err := write(buf, m.Element); err != nil {
		return nil, err
	}

	// SubfieldIndex (uint8)
	if err := write(buf, m.SubfieldIndex); err != nil {
		return nil, err
	}

	// Operation (uint8)
	if err := write(buf, m.Operation); err != nil {
		return nil, err
	}

	// Data (string)
	if err := WriteVarChar(buf, m.Data, 32); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *Amendment) Write(buf *bytes.Buffer) error {

	// FieldIndex (uint8)
	if err := read(buf, &m.FieldIndex); err != nil {
		return err
	}

	// Element (uint16)
	if err := read(buf, &m.Element); err != nil {
		return err
	}

	// SubfieldIndex (uint8)
	if err := read(buf, &m.SubfieldIndex); err != nil {
		return err
	}

	// Operation (uint8)
	if err := read(buf, &m.Operation); err != nil {
		return err
	}

	// Data (string)
	{
		var err error
		m.Data, err = ReadVarChar(buf, 32)
		if err != nil {
			return err
		}
	}
	return nil
}

// AssetSettlement AssetSettlement is the data required to settle an asset
// transfer.
type AssetSettlement struct {
	AssetType   string          `json:"asset_type,omitempty"`  // eg. Share, Bond, Ticket. All characters must be capitalised.
	AssetID     string          `json:"asset_id,omitempty"`    // Randomly generated base58 string.  Each Asset ID should be unique.  However, a Asset ID is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type can be the leading bytes - a convention - to make it easy to identify that it is a token by humans.
	Settlements []QuantityIndex `json:"settlements,omitempty"` // Each element contains the resulting token balance of Asset X for the output Address, which is referred to by the index.
}

// NewAssetSettlement returns a new AssetSettlement with defaults set.
func NewAssetSettlement() *AssetSettlement {
	result := AssetSettlement{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m AssetSettlement) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// AssetType (string)
	if err := WriteFixedChar(buf, m.AssetType, 3); err != nil {
		return nil, err
	}

	// AssetID (string)
	if err := WriteFixedChar(buf, m.AssetID, 32); err != nil {
		return nil, err
	}

	// Settlements ([]QuantityIndex)
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
	return buf.Bytes(), nil
}

func (m *AssetSettlement) Write(buf *bytes.Buffer) error {

	// AssetType (string)
	{
		var err error
		m.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// AssetID (string)
	{
		var err error
		m.AssetID, err = ReadFixedChar(buf, 32)
		if err != nil {
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

// AssetTransfer AssetTransfer is the data required to transfer an asset.
type AssetTransfer struct {
	AssetType      string          `json:"asset_type,omitempty"`      // eg. Share, Bond, Ticket. All characters must be capitalised.
	AssetID        string          `json:"asset_id,omitempty"`        // Randomly generated base58 string.  Each Asset ID should be unique.  However, a Asset ID is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type can be the leading bytes - a convention - to make it easy to identify that it is a token by humans.
	AssetSenders   []QuantityIndex `json:"asset_senders,omitempty"`   // Each element has the value of tokens to be spent from the input address, which is referred to by the index.
	AssetReceivers []TokenReceiver `json:"asset_receivers,omitempty"` // Each element has the value of tokens to be received by the output address, which is referred to by the index.
}

// NewAssetTransfer returns a new AssetTransfer with defaults set.
func NewAssetTransfer() *AssetTransfer {
	result := AssetTransfer{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m AssetTransfer) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// AssetType (string)
	if err := WriteFixedChar(buf, m.AssetType, 3); err != nil {
		return nil, err
	}

	// AssetID (string)
	if err := WriteFixedChar(buf, m.AssetID, 32); err != nil {
		return nil, err
	}

	// AssetSenders ([]QuantityIndex)
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

	// AssetReceivers ([]TokenReceiver)
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
	return buf.Bytes(), nil
}

func (m *AssetTransfer) Write(buf *bytes.Buffer) error {

	// AssetType (string)
	{
		var err error
		m.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// AssetID (string)
	{
		var err error
		m.AssetID, err = ReadFixedChar(buf, 32)
		if err != nil {
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

// Entity Entity represents the details of a legal Entity, such as a public
// or private company, government agency, or and individual.
type Entity struct {
	Name                       string        `json:"name,omitempty"`                          // Length 1-255 bytes (0 is not valid). Issuing entity (company, organization, individual).  Can be any unique identifying string, including human readable names for branding/vanity purposes.
	Type                       byte          `json:"type,omitempty"`                          // P - Public Company Limited by Shares, C - Private Company Limited by Shares, I - Individual, L - Limited Partnership, U -Unlimited Partnership, T - Sole Proprietorship, S - Statutory Company, O - Non-Profit Organization, N - Nation State, G - Government Agency, U - Unit Trust, D - Discretionary Trust.  Found in 'Entities' (Specification/Resources).
	Address                    bool          `json:"address,omitempty"`                       // Registered/Physical/mailing address(HQ). Y-1/N-0, N means there is no issuer address.
	UnitNumber                 string        `json:"unit_number,omitempty"`                   // Issuer/Entity/Contracting Party X Address Details (eg. HQ)
	BuildingNumber             string        `json:"building_number,omitempty"`               //
	Street                     string        `json:"street,omitempty"`                        //
	SuburbCity                 string        `json:"suburb_city,omitempty"`                   //
	TerritoryStateProvinceCode string        `json:"territory_state_province_code,omitempty"` //
	CountryCode                string        `json:"country_code,omitempty"`                  //
	PostalZIPCode              string        `json:"postal_zip_code,omitempty"`               //
	EmailAddress               string        `json:"email_address,omitempty"`                 // Length 0-255 bytes. Address for text-based communication: eg. email address, Bitcoin address
	PhoneNumber                string        `json:"phone_number,omitempty"`                  // Length 0-50 bytes. 0 is valid. Phone Number for Entity.
	KeyRoles                   []KeyRole     `json:"key_roles,omitempty"`                     // A list of Key Roles.
	NotableRoles               []NotableRole `json:"notable_roles,omitempty"`                 // A list of Notable Roles.
}

// NewEntity returns a new Entity with defaults set.
func NewEntity() *Entity {
	result := Entity{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m Entity) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	if err := WriteVarChar(buf, m.Name, 8); err != nil {
		return nil, err
	}

	// Type (byte)
	if err := write(buf, m.Type); err != nil {
		return nil, err
	}

	// Address (bool)
	if err := write(buf, m.Address); err != nil {
		return nil, err
	}

	// UnitNumber (string)
	if err := WriteVarChar(buf, m.UnitNumber, 8); err != nil {
		return nil, err
	}

	// BuildingNumber (string)
	if err := WriteVarChar(buf, m.BuildingNumber, 8); err != nil {
		return nil, err
	}

	// Street (string)
	if err := WriteVarChar(buf, m.Street, 16); err != nil {
		return nil, err
	}

	// SuburbCity (string)
	if err := WriteVarChar(buf, m.SuburbCity, 8); err != nil {
		return nil, err
	}

	// TerritoryStateProvinceCode (string)
	if err := WriteFixedChar(buf, m.TerritoryStateProvinceCode, 5); err != nil {
		return nil, err
	}

	// CountryCode (string)
	if err := WriteFixedChar(buf, m.CountryCode, 3); err != nil {
		return nil, err
	}

	// PostalZIPCode (string)
	if err := WriteFixedChar(buf, m.PostalZIPCode, 12); err != nil {
		return nil, err
	}

	// EmailAddress (string)
	if err := WriteVarChar(buf, m.EmailAddress, 8); err != nil {
		return nil, err
	}

	// PhoneNumber (string)
	if err := WriteVarChar(buf, m.PhoneNumber, 8); err != nil {
		return nil, err
	}

	// KeyRoles ([]KeyRole)
	if err := WriteVariableSize(buf, uint64(len(m.KeyRoles)), 0, 8); err != nil {
		return nil, err
	}
	for _, value := range m.KeyRoles {
		b, err := value.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// NotableRoles ([]NotableRole)
	if err := WriteVariableSize(buf, uint64(len(m.NotableRoles)), 0, 8); err != nil {
		return nil, err
	}
	for _, value := range m.NotableRoles {
		b, err := value.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
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
	if err := read(buf, &m.Type); err != nil {
		return err
	}

	// Address (bool)
	if err := read(buf, &m.Address); err != nil {
		return err
	}

	// UnitNumber (string)
	{
		var err error
		m.UnitNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// BuildingNumber (string)
	{
		var err error
		m.BuildingNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// Street (string)
	{
		var err error
		m.Street, err = ReadVarChar(buf, 16)
		if err != nil {
			return err
		}
	}

	// SuburbCity (string)
	{
		var err error
		m.SuburbCity, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}

	// TerritoryStateProvinceCode (string)
	{
		var err error
		m.TerritoryStateProvinceCode, err = ReadFixedChar(buf, 5)
		if err != nil {
			return err
		}
	}

	// CountryCode (string)
	{
		var err error
		m.CountryCode, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}

	// PostalZIPCode (string)
	{
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

	// KeyRoles ([]KeyRole)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.KeyRoles = make([]KeyRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue KeyRole
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.KeyRoles = append(m.KeyRoles, newValue)
		}
	}

	// NotableRoles ([]NotableRole)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return err
		}
		m.NotableRoles = make([]NotableRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue NotableRole
			if err := newValue.Write(buf); err != nil {
				return err
			}

			m.NotableRoles = append(m.NotableRoles, newValue)
		}
	}
	return nil
}

// Header Header contains common details required by every Tokenized
// message.
type Header struct {
}

// NewHeader returns a new Header with defaults set.
func NewHeader() *Header {
	result := Header{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m Header) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	return buf.Bytes(), nil
}

func (m *Header) Write(buf *bytes.Buffer) error {
	return nil
}

// KeyRole KeyRole is used to refer to a key role in an Entity.
type KeyRole struct {
	Type uint8  `json:"type,omitempty"` // Chairman, Director. Found in 'Roles' in Specification/Resources
	Name string `json:"name,omitempty"` // Length 0-255 bytes. 0 is valid. Name (eg. John Alexander Smith)
}

// NewKeyRole returns a new KeyRole with defaults set.
func NewKeyRole(roleType uint8, name string) *KeyRole {
	result := KeyRole{}
	result.Type = roleType
	result.Name = name
	return &result
}

// Serialize returns the byte representation of the message.
func (m KeyRole) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Type (uint8)
	if err := write(buf, m.Type); err != nil {
		return nil, err
	}

	// Name (string)
	if err := WriteVarChar(buf, m.Name, 8); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *KeyRole) Write(buf *bytes.Buffer) error {

	// Type (uint8)
	if err := read(buf, &m.Type); err != nil {
		return err
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

// NotableRole NotableRole is used to refer to a role of note in an Entity.
type NotableRole struct {
	Type uint8  `json:"type,omitempty"` // Found in 'Roles' in Specification/Resources
	Name string `json:"name,omitempty"` // Length 0-255 bytes. 0 is valid. Name (eg. John Alexander Smith)
}

// NewNotableRole returns a new NotableRole with defaults set.
func NewNotableRole(roleType uint8, name string) *NotableRole {
	result := NotableRole{}
	result.Type = roleType
	result.Name = name
	return &result
}

// Serialize returns the byte representation of the message.
func (m NotableRole) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Type (uint8)
	if err := write(buf, m.Type); err != nil {
		return nil, err
	}

	// Name (string)
	if err := WriteVarChar(buf, m.Name, 8); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *NotableRole) Write(buf *bytes.Buffer) error {

	// Type (uint8)
	if err := read(buf, &m.Type); err != nil {
		return err
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

// QuantityIndex A QuantityIndex contains a quantity, and an index. The
// quantity could be used to describe a number of tokens, or a value. The
// index is used to refer to an input index position.
type QuantityIndex struct {
	Index    uint16 `json:"index,omitempty"`    // The index of the input sending the tokens
	Quantity uint64 `json:"quantity,omitempty"` // Number of tokens being sent
}

// NewQuantityIndex returns a new QuantityIndex with defaults set.
func NewQuantityIndex() *QuantityIndex {
	result := QuantityIndex{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m QuantityIndex) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Index (uint16)
	if err := write(buf, m.Index); err != nil {
		return nil, err
	}

	// Quantity (uint64)
	if err := write(buf, m.Quantity); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *QuantityIndex) Write(buf *bytes.Buffer) error {

	// Index (uint16)
	if err := read(buf, &m.Index); err != nil {
		return err
	}

	// Quantity (uint64)
	if err := read(buf, &m.Quantity); err != nil {
		return err
	}
	return nil
}

// Registry A Registry defines the details of a public Registry.
type Registry struct {
	Name      string `json:"name,omitempty"`       // Length 0-255 bytes. 0 is valid. Registry X Name (eg. Coinbase, Tokenized, etc.)
	URL       string `json:"url,omitempty"`        // Length 0-255 bytes. 0 is valid. If applicable: URL for REST/RPC Endpoint
	PublicKey string `json:"public_key,omitempty"` // Length 0-255 bytes. 0 is not valid. Registry Public Key (eg. Bitcoin Public key), used to confirm digital signed proofs for transfers.  Can also be the same public address that controls a Tokenized Registry.
}

// NewRegistry returns a new Registry with defaults set.
func NewRegistry() *Registry {
	result := Registry{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m Registry) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	if err := WriteVarChar(buf, m.Name, 8); err != nil {
		return nil, err
	}

	// URL (string)
	if err := WriteVarChar(buf, m.URL, 8); err != nil {
		return nil, err
	}

	// PublicKey (string)
	if err := WriteVarChar(buf, m.PublicKey, 8); err != nil {
		return nil, err
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

	// PublicKey (string)
	{
		var err error
		m.PublicKey, err = ReadVarChar(buf, 8)
		if err != nil {
			return err
		}
	}
	return nil
}

// TargetAddress A TargetAddress defines a public address and quantity.
type TargetAddress struct {
	Address  Address `json:"address,omitempty"`  // Public address where the token balance will be changed.
	Quantity uint64  `json:"quantity,omitempty"` // Qty of tokens to be frozen, thawed, confiscated or reconciled. For Contract-wide freezes 0 will be used.
}

// NewTargetAddress returns a new TargetAddress with defaults set.
func NewTargetAddress() *TargetAddress {
	result := TargetAddress{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m TargetAddress) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Address (Address)
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
	if err := write(buf, m.Quantity); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *TargetAddress) Write(buf *bytes.Buffer) error {

	// Address (Address)
	if err := m.Address.Write(buf); err != nil {
		return err
	}

	// Quantity (uint64)
	if err := read(buf, &m.Quantity); err != nil {
		return err
	}
	return nil
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

// NewTokenReceiver returns a new TokenReceiver with defaults set.
func NewTokenReceiver() *TokenReceiver {
	result := TokenReceiver{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m TokenReceiver) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Index (uint16)
	if err := write(buf, m.Index); err != nil {
		return nil, err
	}

	// Quantity (uint64)
	if err := write(buf, m.Quantity); err != nil {
		return nil, err
	}

	// RegistrySigAlgorithm (uint8)
	if err := write(buf, m.RegistrySigAlgorithm); err != nil {
		return nil, err
	}

	// RegistryConfirmationSigToken (string)
	if err := WriteVarChar(buf, m.RegistryConfirmationSigToken, 8); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *TokenReceiver) Write(buf *bytes.Buffer) error {

	// Index (uint16)
	if err := read(buf, &m.Index); err != nil {
		return err
	}

	// Quantity (uint64)
	if err := read(buf, &m.Quantity); err != nil {
		return err
	}

	// RegistrySigAlgorithm (uint8)
	if err := read(buf, &m.RegistrySigAlgorithm); err != nil {
		return err
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

// VotingSystem A VotingSystem defines all details of a Voting System.
type VotingSystem struct {
	Name                        string  `json:"name,omitempty"`                          // eg. Special Resolutions, Ordinary Resolutions, Fundamental Matters, General Matters, Directors' Vote, Poll, etc.
	System                      [8]byte `json:"system,omitempty"`                        // Specifies which subfield is subject to this vote system's control.
	Method                      byte    `json:"method,omitempty"`                        // R - Relative Threshold, A - Absolute Threshold, P - Plurality,  (Relative Threshold means the number of counted votes must exceed the threshold % of total ballots cast.  Abstentations/spoiled votes do not detract from the liklihood of a vote passing as they are not included in the denominator.  Absolute Threshold requires the number of ballots counted to exceed the threshold value when compared to the total outstanding tokens.  Abstentations/spoiled votes detract from the liklihood of the vote passing.  For example, in an absolute threshold vote, if the threshold was 50% and 51% of the total outstanding tokens did not vote, then the vote cannot pass.  50% of all tokens would have had to vote for one vote option for the vote to be successful.
	Logic                       byte    `json:"logic,omitempty"`                         // 0 - Standard Scoring (+1 * # of tokens owned), 1 - Weighted Scoring (1st choice * Vote Max * # of tokens held, 2nd choice * Vote Max-1 * # of tokens held,..etc.)
	ThresholdPercentage         uint8   `json:"threshold_percentage,omitempty"`          // 1-100 is valid for relative threshold and absolute threshold. (eg. 75 means 75% and greater). 0 & >=101 is invalid and will be rejected by the smart contract.  Only applicable to Relative and Absolute Threshold vote methods.  The Plurality vote method requires no threshold value (NULL), as the successful vote option is simply selected on the basis of highest ballots cast for it.
	VoteMultiplierPermitted     byte    `json:"vote_multiplier_permitted,omitempty"`     // Y - Yes, N - No. Where an asset has a vote multiplier, Y must be selected here for the vote multiplier to count, otherwise votes are simply treated as 1x per token.
	InitiativeThreshold         float32 `json:"initiative_threshold,omitempty"`          // Token Owners must pay the threshold amount to broadcast a valid Initiative.  If the Initiative action is valid, the smart contract will start a vote. 0 is valid.
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency,omitempty"` // Currency.  Always paid in BSV or a currency token (CUR) at current market valuations in the currency listed. NULL is valid.
}

// NewVotingSystem returns a new VotingSystem with defaults set.
func NewVotingSystem() *VotingSystem {
	result := VotingSystem{}
	return &result
}

// Serialize returns the byte representation of the message.
func (m VotingSystem) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Name (string)
	if err := WriteVarChar(buf, m.Name, 8); err != nil {
		return nil, err
	}

	// System ([8]byte)
	if err := write(buf, m.System); err != nil {
		return nil, err
	}

	// Method (byte)
	if err := write(buf, m.Method); err != nil {
		return nil, err
	}

	// Logic (byte)
	if err := write(buf, m.Logic); err != nil {
		return nil, err
	}

	// ThresholdPercentage (uint8)
	if err := write(buf, m.ThresholdPercentage); err != nil {
		return nil, err
	}

	// VoteMultiplierPermitted (byte)
	if err := write(buf, m.VoteMultiplierPermitted); err != nil {
		return nil, err
	}

	// InitiativeThreshold (float32)
	if err := write(buf, m.InitiativeThreshold); err != nil {
		return nil, err
	}

	// InitiativeThresholdCurrency (string)
	if err := WriteFixedChar(buf, m.InitiativeThresholdCurrency, 3); err != nil {
		return nil, err
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

	// System ([8]byte)
	if err := read(buf, &m.System); err != nil {
		return err
	}

	// Method (byte)
	if err := read(buf, &m.Method); err != nil {
		return err
	}

	// Logic (byte)
	if err := read(buf, &m.Logic); err != nil {
		return err
	}

	// ThresholdPercentage (uint8)
	if err := read(buf, &m.ThresholdPercentage); err != nil {
		return err
	}

	// VoteMultiplierPermitted (byte)
	if err := read(buf, &m.VoteMultiplierPermitted); err != nil {
		return err
	}

	// InitiativeThreshold (float32)
	if err := read(buf, &m.InitiativeThreshold); err != nil {
		return err
	}

	// InitiativeThresholdCurrency (string)
	{
		var err error
		m.InitiativeThresholdCurrency, err = ReadFixedChar(buf, 3)
		if err != nil {
			return err
		}
	}
	return nil
}
