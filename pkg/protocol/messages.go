package protocol

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// CodeRevertedTx identifies data as a RevertedTx message.
	CodeRevertedTx = 0004

	// CodeOffer identifies data as a Offer message.
	CodeOffer = 1001

	// CodeSignatureRequest identifies data as a SignatureRequest message.
	CodeSignatureRequest = 1002

	// CodeSettlementRequest identifies data as a SettlementRequest message.
	CodeSettlementRequest = 1003

	// CodePublicMessage identifies data as a PublicMessage message.
	CodePublicMessage = 2

	// CodePrivateMessage identifies data as a PrivateMessage message.
	CodePrivateMessage = 3
)

// MessagePayload is the interface for payloads within message actions.
type MessagePayload interface {
	Type() uint16
	Serialize() ([]byte, error)
	Write(b []byte) (int, error)
	Validate() error
}

// MessageTypeMapping holds a mapping of message codes to message types.
func MessageTypeMapping(code uint16) MessagePayload {
	switch code {
	case CodeRevertedTx:
		result := RevertedTx{}
		return &result
	case CodeOffer:
		result := Offer{}
		return &result
	case CodeSignatureRequest:
		result := SignatureRequest{}
		return &result
	case CodeSettlementRequest:
		result := SettlementRequest{}
		return &result
	case CodePublicMessage:
		result := PublicMessage{}
		return &result
	case CodePrivateMessage:
		result := PrivateMessage{}
		return &result
	default:
		return nil
	}
}

// RevertedTx A message that contains a bitcoin transaction that was
// accepted by the network or an agent and then invalidated by a reorg, or
// zero conf double spend. This serves as on chain evidence of the sending
// party's signatures and approval for the given transaction.
type RevertedTx struct {
	Version     uint8     `json:"version,omitempty"`     // Payload Version
	Timestamp   Timestamp `json:"timestamp,omitempty"`   // Timestamp in nanoseconds for when the message sender creates the transaction.
	Transaction []byte    `json:"transaction,omitempty"` // Serialized bitcoin transaction that was reverted/invalidated after being accepted.
}

// Type returns the type identifer for this message.
func (action RevertedTx) Type() uint16 {
	return CodeRevertedTx
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *RevertedTx) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *RevertedTx) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Transaction ([]byte)
	// fmt.Printf("Serializing Transaction\n")
	{
		if err := WriteVarBin(buf, action.Transaction, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in RevertedTx from the byte slice
func (action *RevertedTx) Write(b []byte) (int, error) {
	// fmt.Printf("Reading RevertedTx : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// Transaction ([]byte)
	{
		var err error
		action.Transaction, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Transaction : %d bytes remaining\n%+v\n", buf.Len(), action.Transaction)

	// fmt.Printf("Read RevertedTx : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *RevertedTx) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// Transaction ([]byte)
	{
		if len(m.Transaction) > (2<<32)-1 {
			return fmt.Errorf("varbin field Transaction too long %d/%d", len(m.Transaction), (2<<32)-1)
		}
	}

	return nil
}

func (action RevertedTx) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("Transaction:%#x", action.Transaction))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Offer A message that contains all of the details required for an
// agreement to be formed. Sent to an address(es). The Offer should have
// all, or nearly all, of the details required for the receiving party to
// accept the offer. The Offer shall be in the form of a partially formed
// Bitcoin transaction with all of the relevent details (offer,
// consideration, offeror's payment/receipt details, etc.). The Offer
// message is different to a Signature Request message in that it is
// missing the offeree's payment/receipt details (eg. UTXOs). If the Offer
// message is well received by the offeree, then the offeree can add their
// relevent details (eg. inputs/outputs) and sign the transaction. If an
// additional signature is required from the offeror at this point, then
// the partially-signed transaction can be sent to the offeror by way of a
// Signature Request message.
type Offer struct {
	Version   uint8     `json:"version,omitempty"`   // Payload Version
	Timestamp Timestamp `json:"timestamp,omitempty"` // Timestamp in nanoseconds for when the message sender created the offer.
	Payload   []byte    `json:"payload,omitempty"`   // Serialized Tokenized OP_RETURN message. The message needs data added by another party upon acceptance of offer.
}

// Type returns the type identifer for this message.
func (action Offer) Type() uint16 {
	return CodeOffer
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Offer) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *Offer) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Payload ([]byte)
	// fmt.Printf("Serializing Payload\n")
	{
		if err := WriteVarBin(buf, action.Payload, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in Offer from the byte slice
func (action *Offer) Write(b []byte) (int, error) {
	// fmt.Printf("Reading Offer : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// Payload ([]byte)
	{
		var err error
		action.Payload, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Payload : %d bytes remaining\n%+v\n", buf.Len(), action.Payload)

	// fmt.Printf("Read Offer : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *Offer) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// Payload ([]byte)
	{
		if len(m.Payload) > (2<<32)-1 {
			return fmt.Errorf("varbin field Payload too long %d/%d", len(m.Payload), (2<<32)-1)
		}
	}

	return nil
}

func (action Offer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("Payload:%#x", action.Payload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// SignatureRequest Partially-signed transactions (Tokenized actions,
// Bitcoin, Multisig, Threshold Signatures, etc.) can be passed around
// on-chain to the parties (including Smart Contracts) that still have to
// sign the transaction.
type SignatureRequest struct {
	Version   uint8     `json:"version,omitempty"`   // Payload Version
	Timestamp Timestamp `json:"timestamp,omitempty"` // Timestamp in nanoseconds for when the message sender creates the transaction.
	Payload   []byte    `json:"payload,omitempty"`   // Full serialized bitcoin tx with multiple inputs from different wallets/users.
}

// Type returns the type identifer for this message.
func (action SignatureRequest) Type() uint16 {
	return CodeSignatureRequest
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *SignatureRequest) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *SignatureRequest) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Payload ([]byte)
	// fmt.Printf("Serializing Payload\n")
	{
		if err := WriteVarBin(buf, action.Payload, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in SignatureRequest from the byte slice
func (action *SignatureRequest) Write(b []byte) (int, error) {
	// fmt.Printf("Reading SignatureRequest : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// Payload ([]byte)
	{
		var err error
		action.Payload, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Payload : %d bytes remaining\n%+v\n", buf.Len(), action.Payload)

	// fmt.Printf("Read SignatureRequest : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *SignatureRequest) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// Payload ([]byte)
	{
		if len(m.Payload) > (2<<32)-1 {
			return fmt.Errorf("varbin field Payload too long %d/%d", len(m.Payload), (2<<32)-1)
		}
	}

	return nil
}

func (action SignatureRequest) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("Payload:%#x", action.Payload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// SettlementRequest A message that contains a multi-contract settlement
// that needs settlement data added by another contract. Sent to another
// contract to request data be added.
type SettlementRequest struct {
	Version      uint8           `json:"version,omitempty"`        // Payload Version
	Timestamp    Timestamp       `json:"timestamp,omitempty"`      // Timestamp in nanoseconds for when the message sender creates the transaction.
	TransferTxId TxId            `json:"transfer_tx_id,omitempty"` // Tx Id of the transfer request transaction that triggered this message.
	ContractFees []TargetAddress `json:"contract_fees,omitempty"`  // Contract fees (in bitcoin) and addresses(PKHs) where fees should be paid. Added by each contract as settlement data is added.
	Settlement   []byte          `json:"settlement,omitempty"`     // Serialized settlement OP_RETURN that needs data added by another contract.
}

// Type returns the type identifer for this message.
func (action SettlementRequest) Type() uint16 {
	return CodeSettlementRequest
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *SettlementRequest) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *SettlementRequest) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// TransferTxId (TxId)
	// fmt.Printf("Serializing TransferTxId\n")
	{
		{
			b, err := action.TransferTxId.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// ContractFees ([]TargetAddress)
	// fmt.Printf("Serializing ContractFees\n")
	{
		if err := WriteVariableSize(buf, uint64(len(action.ContractFees)), 8, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ContractFees {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Settlement ([]byte)
	// fmt.Printf("Serializing Settlement\n")
	{
		if err := WriteVarBin(buf, action.Settlement, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in SettlementRequest from the byte slice
func (action *SettlementRequest) Write(b []byte) (int, error) {
	// fmt.Printf("Reading SettlementRequest : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// TransferTxId (TxId)
	{
		if err := action.TransferTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read TransferTxId : %d bytes remaining\n%+v\n", buf.Len(), action.TransferTxId)

	// ContractFees ([]TargetAddress)
	{
		size, err := ReadVariableSize(buf, 8, 8)
		if err != nil {
			return 0, err
		}
		action.ContractFees = make([]TargetAddress, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue TargetAddress
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ContractFees = append(action.ContractFees, newValue)
		}
	}

	// fmt.Printf("Read ContractFees : %d bytes remaining\n%+v\n", buf.Len(), action.ContractFees)

	// Settlement ([]byte)
	{
		var err error
		action.Settlement, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Settlement : %d bytes remaining\n%+v\n", buf.Len(), action.Settlement)

	// fmt.Printf("Read SettlementRequest : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *SettlementRequest) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// TransferTxId (TxId)
	{
		if err := m.TransferTxId.Validate(); err != nil {
			return fmt.Errorf("field TransferTxId is invalid : %s", err)
		}

	}

	// ContractFees ([]TargetAddress)
	{
		if len(m.ContractFees) > (2<<8)-1 {
			return fmt.Errorf("list field ContractFees has too many items %d/%d", len(m.ContractFees), (2<<8)-1)
		}

		for i, value := range m.ContractFees {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field ContractFees[%d] is invalid : %s", i, err)
			}
		}
	}

	// Settlement ([]byte)
	{
		if len(m.Settlement) > (2<<32)-1 {
			return fmt.Errorf("varbin field Settlement too long %d/%d", len(m.Settlement), (2<<32)-1)
		}
	}

	return nil
}

func (action SettlementRequest) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("TransferTxId:%#+v", action.TransferTxId))
	vals = append(vals, fmt.Sprintf("ContractFees:%#+v", action.ContractFees))
	vals = append(vals, fmt.Sprintf("Settlement:%#x", action.Settlement))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// PublicMessage Generic public message or public announcement. Sent to an
// address(es). Can be used for an official issuer announcement.
type PublicMessage struct {
	Version       uint8     `json:"version,omitempty"`        // Payload Version
	Timestamp     Timestamp `json:"timestamp,omitempty"`      // Timestamp in nanoseconds for when the message sender creates the transaction.
	PublicMessage string    `json:"public_message,omitempty"` // Tokenized Ltd. announces product launch.
}

// Type returns the type identifer for this message.
func (action PublicMessage) Type() uint16 {
	return CodePublicMessage
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *PublicMessage) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *PublicMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// PublicMessage (string)
	// fmt.Printf("Serializing PublicMessage\n")
	{
		if err := WriteVarChar(buf, action.PublicMessage, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in PublicMessage from the byte slice
func (action *PublicMessage) Write(b []byte) (int, error) {
	// fmt.Printf("Reading PublicMessage : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// PublicMessage (string)
	{
		var err error
		action.PublicMessage, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read PublicMessage : %d bytes remaining\n%+v\n", buf.Len(), action.PublicMessage)

	// fmt.Printf("Read PublicMessage : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *PublicMessage) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// PublicMessage (string)
	{
		if len(m.PublicMessage) > (2<<32)-1 {
			return fmt.Errorf("varchar field PublicMessage too long %d/%d", len(m.PublicMessage), (2<<32)-1)
		}
	}

	return nil
}

func (action PublicMessage) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("PublicMessage:%#+v", action.PublicMessage))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// PrivateMessage Generic private message. Sent to another address(es).
// Encryption is to be used.
type PrivateMessage struct {
	Version        uint8     `json:"version,omitempty"`         // Payload Version
	Timestamp      Timestamp `json:"timestamp,omitempty"`       // Timestamp in nanoseconds for when the message sender creates the transaction.
	PrivateMessage []byte    `json:"private_message,omitempty"` // Tokenized Ltd announces product launch.
}

// Type returns the type identifer for this message.
func (action PrivateMessage) Type() uint16 {
	return CodePrivateMessage
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *PrivateMessage) Read(b []byte) (int, error) {
	data, err := action.Serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Serialize returns the full OP_RETURN payload bytes.
func (action *PrivateMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Version (uint8)
	// fmt.Printf("Serializing Version\n")
	{
		if err := write(buf, action.Version); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		{
			b, err := action.Timestamp.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// PrivateMessage ([]byte)
	// fmt.Printf("Serializing PrivateMessage\n")
	{
		if err := WriteVarBin(buf, action.PrivateMessage, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Write populates the fields in PrivateMessage from the byte slice
func (action *PrivateMessage) Write(b []byte) (int, error) {
	// fmt.Printf("Reading PrivateMessage : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	{
		if err := read(buf, &action.Version); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// PrivateMessage ([]byte)
	{
		var err error
		action.PrivateMessage, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read PrivateMessage : %d bytes remaining\n%+v\n", buf.Len(), action.PrivateMessage)

	// fmt.Printf("Read PrivateMessage : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

func (m *PrivateMessage) Validate() error {

	// Version (uint8)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	// PrivateMessage ([]byte)
	{
		if len(m.PrivateMessage) > (2<<32)-1 {
			return fmt.Errorf("varbin field PrivateMessage too long %d/%d", len(m.PrivateMessage), (2<<32)-1)
		}
	}

	return nil
}

func (action PrivateMessage) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("PrivateMessage:%#x", action.PrivateMessage))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
