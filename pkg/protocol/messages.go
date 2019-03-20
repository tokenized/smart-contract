package protocol

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	// CodePublicMessage identifies data as a PublicMessage message.
	CodePublicMessage = "0002"

	// CodePrivateMessage identifies data as a PrivateMessage message.
	CodePrivateMessage = "0003"

	// CodeOffer identifies data as a Offer message.
	CodeOffer = "1001"

	// CodeSignatureRequest identifies data as a SignatureRequest message.
	CodeSignatureRequest = "1002"
)

// MessageTypeMapping holds a mapping of message codes to message types.
func MessageTypeMapping(code string) PayloadMessage {
	switch code {
	case CodePublicMessage:
		result := PublicMessage{}
		return &result
	case CodePrivateMessage:
		result := PrivateMessage{}
		return &result
	case CodeOffer:
		result := Offer{}
		return &result
	case CodeSignatureRequest:
		result := SignatureRequest{}
		return &result
	default:
		return nil
	}
}

// PublicMessage Generic public message or public announcement. Sent to an
// address(es). Can be used for an official issuer announcement.
type PublicMessage struct {
	Version       uint8     `json:"version,omitempty"`        // Payload Version
	Timestamp     Timestamp `json:"timestamp,omitempty"`      // Timestamp in nanoseconds for when the message sender creates the transaction.
	PublicMessage string    `json:"public_message,omitempty"` // Tokenized Ltd. announces product launch.
}

// Type returns the type identifer for this message.
func (action PublicMessage) Type() string {
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
	if err := write(buf, action.Version); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized Version : buf len %d\n", buf.Len())

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Serialized Timestamp : buf len %d\n", buf.Len())

	// PublicMessage (string)
	// fmt.Printf("Serializing PublicMessage\n")
	if err := WriteVarChar(buf, action.PublicMessage, 32); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized PublicMessage : buf len %d\n", buf.Len())

	return buf.Bytes(), nil
}

// Write populates the fields in PublicMessage from the byte slice
func (action *PublicMessage) Write(b []byte) (int, error) {
	// fmt.Printf("Reading PublicMessage : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	// fmt.Printf("Reading Version : %d bytes remaining\n", buf.Len())
	if err := read(buf, &action.Version); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	// fmt.Printf("Reading Timestamp : %d bytes remaining\n", buf.Len())
	if err := action.Timestamp.Write(buf); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// PublicMessage (string)
	// fmt.Printf("Reading PublicMessage : %d bytes remaining\n", buf.Len())
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

// PayloadMessage returns the PayloadMessage, if any.
func (action PublicMessage) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
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
func (action PrivateMessage) Type() string {
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
	if err := write(buf, action.Version); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized Version : buf len %d\n", buf.Len())

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Serialized Timestamp : buf len %d\n", buf.Len())

	// PrivateMessage ([]byte)
	// fmt.Printf("Serializing PrivateMessage\n")
	if err := WriteVarBin(buf, action.PrivateMessage, 32); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized PrivateMessage : buf len %d\n", buf.Len())

	return buf.Bytes(), nil
}

// Write populates the fields in PrivateMessage from the byte slice
func (action *PrivateMessage) Write(b []byte) (int, error) {
	// fmt.Printf("Reading PrivateMessage : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	// fmt.Printf("Reading Version : %d bytes remaining\n", buf.Len())
	if err := read(buf, &action.Version); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	// fmt.Printf("Reading Timestamp : %d bytes remaining\n", buf.Len())
	if err := action.Timestamp.Write(buf); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// PrivateMessage ([]byte)
	// fmt.Printf("Reading PrivateMessage : %d bytes remaining\n", buf.Len())
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

// PayloadMessage returns the PayloadMessage, if any.
func (action PrivateMessage) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action PrivateMessage) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("PrivateMessage:%#x", action.PrivateMessage))

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
	Timestamp Timestamp `json:"timestamp,omitempty"` // Timestamp in nanoseconds for when the message sender creates the transaction.
	RefTxId   TxId      `json:"ref_tx_id,omitempty"` // Tx Id of the request transaction referenced by the offer.
	Payload   []byte    `json:"payload,omitempty"`   // Full serialized op return script containing an offer to another party, usually to exchange tokens/bitcoin. The message needs data added by another party.
}

// Type returns the type identifer for this message.
func (action Offer) Type() string {
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
	if err := write(buf, action.Version); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized Version : buf len %d\n", buf.Len())

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Serialized Timestamp : buf len %d\n", buf.Len())

	// RefTxId (TxId)
	// fmt.Printf("Serializing RefTxId\n")
	{
		b, err := action.RefTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Serialized RefTxId : buf len %d\n", buf.Len())

	// Payload ([]byte)
	// fmt.Printf("Serializing Payload\n")
	if err := WriteVarBin(buf, action.Payload, 32); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized Payload : buf len %d\n", buf.Len())

	return buf.Bytes(), nil
}

// Write populates the fields in Offer from the byte slice
func (action *Offer) Write(b []byte) (int, error) {
	// fmt.Printf("Reading Offer : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	// fmt.Printf("Reading Version : %d bytes remaining\n", buf.Len())
	if err := read(buf, &action.Version); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	// fmt.Printf("Reading Timestamp : %d bytes remaining\n", buf.Len())
	if err := action.Timestamp.Write(buf); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// RefTxId (TxId)
	// fmt.Printf("Reading RefTxId : %d bytes remaining\n", buf.Len())
	if err := action.RefTxId.Write(buf); err != nil {
		return 0, err
	}

	// fmt.Printf("Read RefTxId : %d bytes remaining\n%+v\n", buf.Len(), action.RefTxId)

	// Payload ([]byte)
	// fmt.Printf("Reading Payload : %d bytes remaining\n", buf.Len())
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

// PayloadMessage returns the PayloadMessage, if any.
func (action Offer) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Offer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("RefTxId:%#+v", action.RefTxId))
	vals = append(vals, fmt.Sprintf("Payload:%#x", action.Payload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// SignatureRequest Partially-signed transactions (Tokenized actions,
// Bitcoin, Multisig, Threshold Signatures, etc.) can be passed around
// on-chain to the parties (including Smart Contracts) that still have to
// sign the transaction.
type SignatureRequest struct {
	Version    uint8     `json:"version,omitempty"`     // Payload Version
	Timestamp  Timestamp `json:"timestamp,omitempty"`   // Timestamp in nanoseconds for when the message sender creates the transaction.
	SigRequest []byte    `json:"sig_request,omitempty"` // Full serialized bitcoin tx with multiple inputs from different wallets/users.
}

// Type returns the type identifer for this message.
func (action SignatureRequest) Type() string {
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
	if err := write(buf, action.Version); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized Version : buf len %d\n", buf.Len())

	// Timestamp (Timestamp)
	// fmt.Printf("Serializing Timestamp\n")
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}
	// fmt.Printf("Serialized Timestamp : buf len %d\n", buf.Len())

	// SigRequest ([]byte)
	// fmt.Printf("Serializing SigRequest\n")
	if err := WriteVarBin(buf, action.SigRequest, 32); err != nil {
		return nil, err
	}
	// fmt.Printf("Serialized SigRequest : buf len %d\n", buf.Len())

	return buf.Bytes(), nil
}

// Write populates the fields in SignatureRequest from the byte slice
func (action *SignatureRequest) Write(b []byte) (int, error) {
	// fmt.Printf("Reading SignatureRequest : %d bytes\n", len(b))
	buf := bytes.NewBuffer(b)

	// Version (uint8)
	// fmt.Printf("Reading Version : %d bytes remaining\n", buf.Len())
	if err := read(buf, &action.Version); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Version : %d bytes remaining\n%+v\n", buf.Len(), action.Version)

	// Timestamp (Timestamp)
	// fmt.Printf("Reading Timestamp : %d bytes remaining\n", buf.Len())
	if err := action.Timestamp.Write(buf); err != nil {
		return 0, err
	}

	// fmt.Printf("Read Timestamp : %d bytes remaining\n%+v\n", buf.Len(), action.Timestamp)

	// SigRequest ([]byte)
	// fmt.Printf("Reading SigRequest : %d bytes remaining\n", buf.Len())
	{
		var err error
		action.SigRequest, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// fmt.Printf("Read SigRequest : %d bytes remaining\n%+v\n", buf.Len(), action.SigRequest)

	// fmt.Printf("Read SignatureRequest : %d bytes remaining\n", buf.Len())
	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action SignatureRequest) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action SignatureRequest) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Version:%v", action.Version))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))
	vals = append(vals, fmt.Sprintf("SigRequest:%#x", action.SigRequest))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
