// Package protocol provides base level structs and validation for
// the protocol.
//
// The code in this file is auto-generated. Do not edit it by hand as it will
// be overwritten when code is regenerated.
package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// ProtocolID is the current protocol ID
	ProtocolID uint32 = 0x00000020

	// CodeAssetDefinition identifies data as a AssetDefinition message.
	CodeAssetDefinition = "A1"

	// CodeAssetCreation identifies data as a AssetCreation message.
	CodeAssetCreation = "A2"

	// CodeAssetModification identifies data as a AssetModification message.
	CodeAssetModification = "A3"

	// CodeContractOffer identifies data as a ContractOffer message.
	CodeContractOffer = "C1"

	// CodeContractFormation identifies data as a ContractFormation message.
	CodeContractFormation = "C2"

	// CodeContractAmendment identifies data as a ContractAmendment message.
	CodeContractAmendment = "C3"

	// CodeOrder identifies data as a Order message.
	CodeOrder = "E1"

	// CodeFreeze identifies data as a Freeze message.
	CodeFreeze = "E2"

	// CodeThaw identifies data as a Thaw message.
	CodeThaw = "E3"

	// CodeConfiscation identifies data as a Confiscation message.
	CodeConfiscation = "E4"

	// CodeReconciliation identifies data as a Reconciliation message.
	CodeReconciliation = "E5"

	// CodeInitiative identifies data as a Initiative message.
	CodeInitiative = "G1"

	// CodeReferendum identifies data as a Referendum message.
	CodeReferendum = "G2"

	// CodeVote identifies data as a Vote message.
	CodeVote = "G3"

	// CodeBallotCast identifies data as a BallotCast message.
	CodeBallotCast = "G4"

	// CodeBallotCounted identifies data as a BallotCounted message.
	CodeBallotCounted = "G5"

	// CodeResult identifies data as a Result message.
	CodeResult = "G6"

	// CodeMessage identifies data as a Message message.
	CodeMessage = "M1"

	// CodeRejection identifies data as a Rejection message.
	CodeRejection = "M2"

	// CodeEstablishment identifies data as a Establishment message.
	CodeEstablishment = "R1"

	// CodeAddition identifies data as a Addition message.
	CodeAddition = "R2"

	// CodeAlteration identifies data as a Alteration message.
	CodeAlteration = "R3"

	// CodeRemoval identifies data as a Removal message.
	CodeRemoval = "R4"

	// CodeSend identifies data as a Send message.
	CodeSend = "T1"

	// CodeExchange identifies data as a Exchange message.
	CodeExchange = "T2"

	// CodeSwap identifies data as a Swap message.
	CodeSwap = "T3"

	// CodeSettlement identifies data as a Settlement message.
	CodeSettlement = "T4"
)

// TypeMapping holds a mapping of message codes to message types.
var TypeMapping = map[string]OpReturnMessage{

	CodeAssetDefinition: &AssetDefinition{},

	CodeAssetCreation: &AssetCreation{},

	CodeAssetModification: &AssetModification{},

	CodeContractOffer: &ContractOffer{},

	CodeContractFormation: &ContractFormation{},

	CodeContractAmendment: &ContractAmendment{},

	CodeOrder: &Order{},

	CodeFreeze: &Freeze{},

	CodeThaw: &Thaw{},

	CodeConfiscation: &Confiscation{},

	CodeReconciliation: &Reconciliation{},

	CodeInitiative: &Initiative{},

	CodeReferendum: &Referendum{},

	CodeVote: &Vote{},

	CodeBallotCast: &BallotCast{},

	CodeBallotCounted: &BallotCounted{},

	CodeResult: &Result{},

	CodeMessage: &Message{},

	CodeRejection: &Rejection{},

	CodeEstablishment: &Establishment{},

	CodeAddition: &Addition{},

	CodeAlteration: &Alteration{},

	CodeRemoval: &Removal{},

	CodeSend: &Send{},

	CodeExchange: &Exchange{},

	CodeSwap: &Swap{},

	CodeSettlement: &Settlement{},
}

// PayloadMessage is the interface for messages that are derived from
// payloads, such as asset types.
type PayloadMessage interface {
	io.ReadWriter
	Type() string
	Len() int64
}

// OpReturnMessage implements a base interface for all message types.
type OpReturnMessage interface {
	PayloadMessage
	String() string
	PayloadMessage() (PayloadMessage, error)
}

// New returns a new message, as an OpReturnMessage, from the OP_RETURN
// payload.
func New(b []byte) (OpReturnMessage, error) {
	code, err := Code(b)
	if err != nil {
		return nil, err
	}

	t, ok := TypeMapping[code]
	if !ok {
		return nil, fmt.Errorf("Unknown code :  %v", code)
	}

	if _, err := t.Write(b); err != nil {
		return nil, err
	}

	return t, nil
}

// Code returns the identifying code from the OP_RETURN payload.
func Code(b []byte) (string, error) {
	if len(b) < 9 || b[0] != 0x6a {
		return "", errors.New("Not an OP_RETURN payload")
	}

	offset := 7

	if b[1] < 0x4c {
		offset = 6
	}

	return string(b[offset : offset+2]), nil
}

// BaseMessage is a common struct for all messages.
type BaseMessage struct {
}

// pad returns a []byte with the given length.
func (bm BaseMessage) pad(b []byte, l int) []byte {
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

// write writes the  value to the buffer.
func (bm BaseMessage) write(buf *bytes.Buffer, v interface{}) error {
	return binary.Write(buf, binary.BigEndian, v)
}

// read fills the value with the appropriate number of bytes from the buffer.
//
// This is useful for fixed size types such as int, float etc.
func (bm BaseMessage) read(buf *bytes.Buffer, v interface{}) error {
	return binary.Read(buf, binary.BigEndian, v)
}

// readLen reads the number of bytes from the buffer to fill the slice of
// []byte.
func (bm BaseMessage) readLen(buf *bytes.Buffer, b []byte) error {
	_, err := io.ReadFull(buf, b)
	return err
}

//
// Asset Operations
//

// AssetDefinition : This action is used by the issuer to define the
// properties/characteristics of the Asset (token) that it wants to create.
type AssetDefinition struct {
	BaseMessage
	Header              []byte
	ProtocolID          uint32
	ActionPrefix        []byte
	Version             uint8
	AssetType           []byte
	AssetID             []byte
	AuthorizationFlags  []byte
	VotingSystem        byte
	VoteMultiplier      uint8
	Qty                 uint64
	ContractFeeCurrency []byte
	ContractFeeVar      float32
	ContractFeeFixed    float32
	Payload             []byte
}

// NewAssetDefinition returns a new AssetDefinition with defaults set.
func NewAssetDefinition() AssetDefinition {
	return AssetDefinition{
		Header:       []byte{0x6a, 0x4c, 0xd2},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeAssetDefinition),
	}
}

// Type returns the type identifer for this message.
func (m AssetDefinition) Type() string {
	return CodeAssetDefinition
}

// Len returns the byte size of this message.
func (m AssetDefinition) Len() int64 {
	return int64(len(m.Header)) + 210
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetDefinition) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m AssetDefinition) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMultiplier); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFeeCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Payload, 145)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetDefinition) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.VoteMultiplier)

	m.read(buf, &m.Qty)

	m.ContractFeeCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.ContractFeeCurrency); err != nil {
		return 0, err
	}

	m.ContractFeeCurrency = bytes.Trim(m.ContractFeeCurrency, "\x00")

	m.read(buf, &m.ContractFeeVar)

	m.read(buf, &m.ContractFeeFixed)

	m.Payload = make([]byte, 145)
	if err := m.readLen(buf, m.Payload); err != nil {
		return 0, err
	}

	return 213, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m AssetDefinition) PayloadMessage() (PayloadMessage, error) {
	p, err := NewPayloadMessageFromCode(m.AssetType)
	if p == nil || err != nil {
		return nil, err
	}

	if _, err := p.Write(m.Payload); err != nil {
		return nil, err
	}

	return p, nil
}

func (m AssetDefinition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", m.VoteMultiplier))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("ContractFeeCurrency:\"%v\"", string(m.ContractFeeCurrency)))

	vals = append(vals, fmt.Sprintf("ContractFeeVar:%v", m.ContractFeeVar))

	vals = append(vals, fmt.Sprintf("ContractFeeFixed:%v", m.ContractFeeFixed))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetCreation : This action creates an Asset in response to the Issuer's
// instructions in the Definition Action.
type AssetCreation struct {
	BaseMessage
	Header              []byte
	ProtocolID          uint32
	ActionPrefix        []byte
	Version             uint8
	AssetType           []byte
	AssetID             []byte
	AssetRevision       uint16
	Timestamp           uint64
	AuthorizationFlags  []byte
	VotingSystem        byte
	VoteMultiplier      uint8
	Qty                 uint64
	ContractFeeCurrency []byte
	ContractFeeVar      float32
	ContractFeeFixed    float32
	Payload             []byte
}

// NewAssetCreation returns a new AssetCreation with defaults set.
func NewAssetCreation() AssetCreation {
	return AssetCreation{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeAssetCreation),
	}
}

// Type returns the type identifer for this message.
func (m AssetCreation) Type() string {
	return CodeAssetCreation
}

// Len returns the byte size of this message.
func (m AssetCreation) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetCreation) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m AssetCreation) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.AssetRevision); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMultiplier); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFeeCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Payload, 145)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetCreation) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.AssetRevision)

	m.read(buf, &m.Timestamp)

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.VoteMultiplier)

	m.read(buf, &m.Qty)

	m.ContractFeeCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.ContractFeeCurrency); err != nil {
		return 0, err
	}

	m.ContractFeeCurrency = bytes.Trim(m.ContractFeeCurrency, "\x00")

	m.read(buf, &m.ContractFeeVar)

	m.read(buf, &m.ContractFeeFixed)

	m.Payload = make([]byte, 145)
	if err := m.readLen(buf, m.Payload); err != nil {
		return 0, err
	}

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m AssetCreation) PayloadMessage() (PayloadMessage, error) {
	p, err := NewPayloadMessageFromCode(m.AssetType)
	if p == nil || err != nil {
		return nil, err
	}

	if _, err := p.Write(m.Payload); err != nil {
		return nil, err
	}

	return p, nil
}

func (m AssetCreation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("AssetRevision:%v", m.AssetRevision))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", m.VoteMultiplier))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("ContractFeeCurrency:\"%v\"", string(m.ContractFeeCurrency)))

	vals = append(vals, fmt.Sprintf("ContractFeeVar:%v", m.ContractFeeVar))

	vals = append(vals, fmt.Sprintf("ContractFeeFixed:%v", m.ContractFeeFixed))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetModification : Token Dilutions, Call Backs/Revocations, burning
// etc. Any field can be amended except for the Asset Revision field
// (incremental counter based on the previous Asset Creation Txn) and the
// Action Prefix. Asset Types specific payloads are locked to the Asset
// Type. Asset Type specific payload amendments must be done as a protocol
// Version upgrade. Authorization flags can restrict some fields or all
// fields from being amended. Some amendments require a Token Owner vote
// for the smart contract to permit.
type AssetModification struct {
	BaseMessage
	Header              []byte
	ProtocolID          uint32
	ActionPrefix        []byte
	Version             uint8
	AssetType           []byte
	AssetID             []byte
	AssetRevision       uint16
	AuthorizationFlags  []byte
	VotingSystem        byte
	VoteMultiplier      uint8
	Qty                 uint64
	ContractFeeCurrency []byte
	ContractFeeVar      float32
	ContractFeeFixed    float32
	Payload             []byte
}

// NewAssetModification returns a new AssetModification with defaults set.
func NewAssetModification() AssetModification {
	return AssetModification{
		Header:       []byte{0x6a, 0x4c, 0xd4},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeAssetModification),
	}
}

// Type returns the type identifer for this message.
func (m AssetModification) Type() string {
	return CodeAssetModification
}

// Len returns the byte size of this message.
func (m AssetModification) Len() int64 {
	return int64(len(m.Header)) + 212
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m AssetModification) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m AssetModification) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.AssetRevision); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMultiplier); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFeeCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractFeeFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Payload, 145)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *AssetModification) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.AssetRevision)

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.VoteMultiplier)

	m.read(buf, &m.Qty)

	m.ContractFeeCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.ContractFeeCurrency); err != nil {
		return 0, err
	}

	m.ContractFeeCurrency = bytes.Trim(m.ContractFeeCurrency, "\x00")

	m.read(buf, &m.ContractFeeVar)

	m.read(buf, &m.ContractFeeFixed)

	m.Payload = make([]byte, 145)
	if err := m.readLen(buf, m.Payload); err != nil {
		return 0, err
	}

	return 215, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m AssetModification) PayloadMessage() (PayloadMessage, error) {
	p, err := NewPayloadMessageFromCode(m.AssetType)
	if p == nil || err != nil {
		return nil, err
	}

	if _, err := p.Write(m.Payload); err != nil {
		return nil, err
	}

	return p, nil
}

func (m AssetModification) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("AssetRevision:%v", m.AssetRevision))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", m.VoteMultiplier))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("ContractFeeCurrency:\"%v\"", string(m.ContractFeeCurrency)))

	vals = append(vals, fmt.Sprintf("ContractFeeVar:%v", m.ContractFeeVar))

	vals = append(vals, fmt.Sprintf("ContractFeeFixed:%v", m.ContractFeeFixed))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Contract Operations
//

// ContractOffer : The Contract Offer action allows the Issuer to tell the
// smart contract what they want the details (labels, data, T&C's, etc.) of
// the Contract to be on-chain in a public and immutable way. The Contract
// Offer action 'initializes' a generic smart contract that has been spun
// up by either the Smart Contract Operator or the Issuer. This on chain
// action allows for the positive response from the smart contract with
// either a Contract Formation Action or a Rejection Action.
type ContractOffer struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	ContractName                []byte
	ContractFileHash            []byte
	GoverningLaw                []byte
	Jurisdiction                []byte
	ContractExpiration          uint64
	URI                         []byte
	IssuerID                    []byte
	IssuerType                  byte
	ContractOperatorID          []byte
	AuthorizationFlags          []byte
	VotingSystem                byte
	InitiativeThreshold         float32
	InitiativeThresholdCurrency []byte
	RestrictedQty               uint64
}

// NewContractOffer returns a new ContractOffer with defaults set.
func NewContractOffer() ContractOffer {
	return ContractOffer{
		Header:       []byte{0x6a, 0x4c, 0xd2},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeContractOffer),
	}
}

// Type returns the type identifer for this message.
func (m ContractOffer) Type() string {
	return CodeContractOffer
}

// Len returns the byte size of this message.
func (m ContractOffer) Len() int64 {
	return int64(len(m.Header)) + 210
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m ContractOffer) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m ContractOffer) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractName, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFileHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.GoverningLaw, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Jurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractExpiration); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.URI, 70)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.IssuerID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.IssuerType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractOperatorID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.InitiativeThreshold); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.InitiativeThresholdCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.RestrictedQty); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *ContractOffer) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.ContractName = make([]byte, 32)
	if err := m.readLen(buf, m.ContractName); err != nil {
		return 0, err
	}

	m.ContractName = bytes.Trim(m.ContractName, "\x00")

	m.ContractFileHash = make([]byte, 32)
	if err := m.readLen(buf, m.ContractFileHash); err != nil {
		return 0, err
	}

	m.ContractFileHash = bytes.Trim(m.ContractFileHash, "\x00")

	m.GoverningLaw = make([]byte, 5)
	if err := m.readLen(buf, m.GoverningLaw); err != nil {
		return 0, err
	}

	m.GoverningLaw = bytes.Trim(m.GoverningLaw, "\x00")

	m.Jurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.Jurisdiction); err != nil {
		return 0, err
	}

	m.Jurisdiction = bytes.Trim(m.Jurisdiction, "\x00")

	m.read(buf, &m.ContractExpiration)

	m.URI = make([]byte, 70)
	if err := m.readLen(buf, m.URI); err != nil {
		return 0, err
	}

	m.URI = bytes.Trim(m.URI, "\x00")

	m.IssuerID = make([]byte, 16)
	if err := m.readLen(buf, m.IssuerID); err != nil {
		return 0, err
	}

	m.IssuerID = bytes.Trim(m.IssuerID, "\x00")

	m.read(buf, &m.IssuerType)

	m.ContractOperatorID = make([]byte, 16)
	if err := m.readLen(buf, m.ContractOperatorID); err != nil {
		return 0, err
	}

	m.ContractOperatorID = bytes.Trim(m.ContractOperatorID, "\x00")

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.InitiativeThreshold)

	m.InitiativeThresholdCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.InitiativeThresholdCurrency); err != nil {
		return 0, err
	}

	m.InitiativeThresholdCurrency = bytes.Trim(m.InitiativeThresholdCurrency, "\x00")

	m.read(buf, &m.RestrictedQty)

	return 213, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m ContractOffer) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m ContractOffer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("ContractName:\"%v\"", string(m.ContractName)))

	vals = append(vals, fmt.Sprintf("ContractFileHash:\"%x\"", m.ContractFileHash))

	vals = append(vals, fmt.Sprintf("GoverningLaw:\"%v\"", string(m.GoverningLaw)))

	vals = append(vals, fmt.Sprintf("Jurisdiction:\"%v\"", string(m.Jurisdiction)))

	vals = append(vals, fmt.Sprintf("ContractExpiration:%v", m.ContractExpiration))

	vals = append(vals, fmt.Sprintf("URI:\"%v\"", string(m.URI)))

	vals = append(vals, fmt.Sprintf("IssuerID:\"%v\"", string(m.IssuerID)))

	vals = append(vals, fmt.Sprintf("IssuerType:\"%v\"", string(m.IssuerType)))

	vals = append(vals, fmt.Sprintf("ContractOperatorID:\"%v\"", string(m.ContractOperatorID)))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("InitiativeThreshold:%v", m.InitiativeThreshold))

	vals = append(vals, fmt.Sprintf("InitiativeThresholdCurrency:\"%v\"", string(m.InitiativeThresholdCurrency)))

	vals = append(vals, fmt.Sprintf("RestrictedQty:%v", m.RestrictedQty))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractFormation : This txn is created by the Contract (smart
// contract/off-chain agent/token contract) upon receipt of a valid
// Contract Offer Action from the issuer. The Smart Contract will execute
// on a server controlled by the Issuer. or a Smart Contract Operator on
// their behalf .
type ContractFormation struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	ContractName                []byte
	ContractFileHash            []byte
	GoverningLaw                []byte
	Jurisdiction                []byte
	Timestamp                   uint64
	ContractExpiration          uint64
	URI                         []byte
	ContractRevision            uint16
	IssuerID                    []byte
	IssuerType                  byte
	ContractOperatorID          []byte
	AuthorizationFlags          []byte
	VotingSystem                byte
	InitiativeThreshold         float32
	InitiativeThresholdCurrency []byte
	RestrictedQty               uint64
}

// NewContractFormation returns a new ContractFormation with defaults set.
func NewContractFormation() ContractFormation {
	return ContractFormation{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeContractFormation),
	}
}

// Type returns the type identifer for this message.
func (m ContractFormation) Type() string {
	return CodeContractFormation
}

// Len returns the byte size of this message.
func (m ContractFormation) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m ContractFormation) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m ContractFormation) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractName, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFileHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.GoverningLaw, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Jurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractExpiration); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.URI, 70)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractRevision); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.IssuerID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.IssuerType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractOperatorID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.InitiativeThreshold); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.InitiativeThresholdCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.RestrictedQty); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *ContractFormation) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.ContractName = make([]byte, 32)
	if err := m.readLen(buf, m.ContractName); err != nil {
		return 0, err
	}

	m.ContractName = bytes.Trim(m.ContractName, "\x00")

	m.ContractFileHash = make([]byte, 32)
	if err := m.readLen(buf, m.ContractFileHash); err != nil {
		return 0, err
	}

	m.ContractFileHash = bytes.Trim(m.ContractFileHash, "\x00")

	m.GoverningLaw = make([]byte, 5)
	if err := m.readLen(buf, m.GoverningLaw); err != nil {
		return 0, err
	}

	m.GoverningLaw = bytes.Trim(m.GoverningLaw, "\x00")

	m.Jurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.Jurisdiction); err != nil {
		return 0, err
	}

	m.Jurisdiction = bytes.Trim(m.Jurisdiction, "\x00")

	m.read(buf, &m.Timestamp)

	m.read(buf, &m.ContractExpiration)

	m.URI = make([]byte, 70)
	if err := m.readLen(buf, m.URI); err != nil {
		return 0, err
	}

	m.URI = bytes.Trim(m.URI, "\x00")

	m.read(buf, &m.ContractRevision)

	m.IssuerID = make([]byte, 16)
	if err := m.readLen(buf, m.IssuerID); err != nil {
		return 0, err
	}

	m.IssuerID = bytes.Trim(m.IssuerID, "\x00")

	m.read(buf, &m.IssuerType)

	m.ContractOperatorID = make([]byte, 16)
	if err := m.readLen(buf, m.ContractOperatorID); err != nil {
		return 0, err
	}

	m.ContractOperatorID = bytes.Trim(m.ContractOperatorID, "\x00")

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.InitiativeThreshold)

	m.InitiativeThresholdCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.InitiativeThresholdCurrency); err != nil {
		return 0, err
	}

	m.InitiativeThresholdCurrency = bytes.Trim(m.InitiativeThresholdCurrency, "\x00")

	m.read(buf, &m.RestrictedQty)

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m ContractFormation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m ContractFormation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("ContractName:\"%v\"", string(m.ContractName)))

	vals = append(vals, fmt.Sprintf("ContractFileHash:\"%x\"", m.ContractFileHash))

	vals = append(vals, fmt.Sprintf("GoverningLaw:\"%v\"", string(m.GoverningLaw)))

	vals = append(vals, fmt.Sprintf("Jurisdiction:\"%v\"", string(m.Jurisdiction)))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("ContractExpiration:%v", m.ContractExpiration))

	vals = append(vals, fmt.Sprintf("URI:\"%v\"", string(m.URI)))

	vals = append(vals, fmt.Sprintf("ContractRevision:%v", m.ContractRevision))

	vals = append(vals, fmt.Sprintf("IssuerID:\"%v\"", string(m.IssuerID)))

	vals = append(vals, fmt.Sprintf("IssuerType:\"%v\"", string(m.IssuerType)))

	vals = append(vals, fmt.Sprintf("ContractOperatorID:\"%v\"", string(m.ContractOperatorID)))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("InitiativeThreshold:%v", m.InitiativeThreshold))

	vals = append(vals, fmt.Sprintf("InitiativeThresholdCurrency:\"%v\"", string(m.InitiativeThresholdCurrency)))

	vals = append(vals, fmt.Sprintf("RestrictedQty:%v", m.RestrictedQty))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractAmendment : the issuer can initiate an amendment to the contract
// establishment metadata. This can be due to a change of name, change of
// contract terms, change of authorizations, or change of the URI. The
// ability to make an amendment to the contract is limited by the
// Authorization Flag set on the previous Contract Formation action. The
// Authorization Flags can be set to allow Contract Amendments, but only if
// a Token Owner vote has passed in favour of making the Amendment.
// Contract revision/protocol identifier and action prefix can't be
// amended. The rest of the fields are open to change. However, the Issuer
// is responsible for acting lawfully (in their jurisdiction) and in
// accordance with the terms of the Investment Contract.
type ContractAmendment struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	ContractName                []byte
	ContractFileHash            []byte
	GoverningLaw                []byte
	Jurisdiction                []byte
	ContractExpiration          uint64
	URI                         []byte
	ContractRevision            uint16
	IssuerID                    []byte
	IssuerType                  byte
	ContractOperatorID          []byte
	AuthorizationFlags          []byte
	VotingSystem                byte
	InitiativeThreshold         float32
	InitiativeThresholdCurrency []byte
	RestrictedQty               uint64
}

// NewContractAmendment returns a new ContractAmendment with defaults set.
func NewContractAmendment() ContractAmendment {
	return ContractAmendment{
		Header:       []byte{0x6a, 0x4c, 0xd4},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeContractAmendment),
	}
}

// Type returns the type identifer for this message.
func (m ContractAmendment) Type() string {
	return CodeContractAmendment
}

// Len returns the byte size of this message.
func (m ContractAmendment) Len() int64 {
	return int64(len(m.Header)) + 212
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m ContractAmendment) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m ContractAmendment) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractName, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractFileHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.GoverningLaw, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Jurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractExpiration); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.URI, 70)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ContractRevision); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.IssuerID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.IssuerType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ContractOperatorID, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AuthorizationFlags, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VotingSystem); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.InitiativeThreshold); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.InitiativeThresholdCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.RestrictedQty); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *ContractAmendment) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.ContractName = make([]byte, 32)
	if err := m.readLen(buf, m.ContractName); err != nil {
		return 0, err
	}

	m.ContractName = bytes.Trim(m.ContractName, "\x00")

	m.ContractFileHash = make([]byte, 32)
	if err := m.readLen(buf, m.ContractFileHash); err != nil {
		return 0, err
	}

	m.ContractFileHash = bytes.Trim(m.ContractFileHash, "\x00")

	m.GoverningLaw = make([]byte, 5)
	if err := m.readLen(buf, m.GoverningLaw); err != nil {
		return 0, err
	}

	m.GoverningLaw = bytes.Trim(m.GoverningLaw, "\x00")

	m.Jurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.Jurisdiction); err != nil {
		return 0, err
	}

	m.Jurisdiction = bytes.Trim(m.Jurisdiction, "\x00")

	m.read(buf, &m.ContractExpiration)

	m.URI = make([]byte, 70)
	if err := m.readLen(buf, m.URI); err != nil {
		return 0, err
	}

	m.URI = bytes.Trim(m.URI, "\x00")

	m.read(buf, &m.ContractRevision)

	m.IssuerID = make([]byte, 16)
	if err := m.readLen(buf, m.IssuerID); err != nil {
		return 0, err
	}

	m.IssuerID = bytes.Trim(m.IssuerID, "\x00")

	m.read(buf, &m.IssuerType)

	m.ContractOperatorID = make([]byte, 16)
	if err := m.readLen(buf, m.ContractOperatorID); err != nil {
		return 0, err
	}

	m.ContractOperatorID = bytes.Trim(m.ContractOperatorID, "\x00")

	m.AuthorizationFlags = make([]byte, 2)
	if err := m.readLen(buf, m.AuthorizationFlags); err != nil {
		return 0, err
	}

	m.AuthorizationFlags = bytes.Trim(m.AuthorizationFlags, "\x00")

	m.read(buf, &m.VotingSystem)

	m.read(buf, &m.InitiativeThreshold)

	m.InitiativeThresholdCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.InitiativeThresholdCurrency); err != nil {
		return 0, err
	}

	m.InitiativeThresholdCurrency = bytes.Trim(m.InitiativeThresholdCurrency, "\x00")

	m.read(buf, &m.RestrictedQty)

	return 215, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m ContractAmendment) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m ContractAmendment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("ContractName:\"%v\"", string(m.ContractName)))

	vals = append(vals, fmt.Sprintf("ContractFileHash:\"%x\"", m.ContractFileHash))

	vals = append(vals, fmt.Sprintf("GoverningLaw:\"%v\"", string(m.GoverningLaw)))

	vals = append(vals, fmt.Sprintf("Jurisdiction:\"%v\"", string(m.Jurisdiction)))

	vals = append(vals, fmt.Sprintf("ContractExpiration:%v", m.ContractExpiration))

	vals = append(vals, fmt.Sprintf("URI:\"%v\"", string(m.URI)))

	vals = append(vals, fmt.Sprintf("ContractRevision:%v", m.ContractRevision))

	vals = append(vals, fmt.Sprintf("IssuerID:\"%v\"", string(m.IssuerID)))

	vals = append(vals, fmt.Sprintf("IssuerType:\"%v\"", string(m.IssuerType)))

	vals = append(vals, fmt.Sprintf("ContractOperatorID:\"%v\"", string(m.ContractOperatorID)))

	vals = append(vals, fmt.Sprintf("VotingSystem:\"%v\"", string(m.VotingSystem)))

	vals = append(vals, fmt.Sprintf("InitiativeThreshold:%v", m.InitiativeThreshold))

	vals = append(vals, fmt.Sprintf("InitiativeThresholdCurrency:\"%v\"", string(m.InitiativeThresholdCurrency)))

	vals = append(vals, fmt.Sprintf("RestrictedQty:%v", m.RestrictedQty))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Enforcement Operations
//

// Order : Issuer to signal to the smart contract that the tokens that a
// particular PKH owns are to be confiscated, frozen or thawed.
type Order struct {
	BaseMessage
	Header                 []byte
	ProtocolID             uint32
	ActionPrefix           []byte
	Version                uint8
	AssetType              []byte
	AssetID                []byte
	ComplianceAction       byte
	TargetAddress          []byte
	DepositAddress         []byte
	SupportingEvidenceHash []byte
	Qty                    uint64
	Expiration             uint64
	Message                []byte
}

// NewOrder returns a new Order with defaults set.
func NewOrder() Order {
	return Order{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeOrder),
	}
}

// Type returns the type identifer for this message.
func (m Order) Type() string {
	return CodeOrder
}

// Len returns the byte size of this message.
func (m Order) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Order) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Order) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ComplianceAction); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.TargetAddress, 34)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.DepositAddress, 34)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.SupportingEvidenceHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Expiration); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 61)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Order) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.ComplianceAction)

	m.TargetAddress = make([]byte, 34)
	if err := m.readLen(buf, m.TargetAddress); err != nil {
		return 0, err
	}

	m.TargetAddress = bytes.Trim(m.TargetAddress, "\x00")

	m.DepositAddress = make([]byte, 34)
	if err := m.readLen(buf, m.DepositAddress); err != nil {
		return 0, err
	}

	m.DepositAddress = bytes.Trim(m.DepositAddress, "\x00")

	m.SupportingEvidenceHash = make([]byte, 32)
	if err := m.readLen(buf, m.SupportingEvidenceHash); err != nil {
		return 0, err
	}

	m.SupportingEvidenceHash = bytes.Trim(m.SupportingEvidenceHash, "\x00")

	m.read(buf, &m.Qty)

	m.read(buf, &m.Expiration)

	m.Message = make([]byte, 61)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Order) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Order) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("ComplianceAction:\"%v\"", string(m.ComplianceAction)))

	vals = append(vals, fmt.Sprintf("TargetAddress:\"%v\"", string(m.TargetAddress)))

	vals = append(vals, fmt.Sprintf("DepositAddress:\"%v\"", string(m.DepositAddress)))

	vals = append(vals, fmt.Sprintf("SupportingEvidenceHash:\"%x\"", m.SupportingEvidenceHash))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("Expiration:%v", m.Expiration))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Freeze : To be used to comply with contractual/legal requirements. The
// whitelist public address will be marked as frozen. However the Freeze
// action publishes this fact to the public blockchain for transparency.
// The Contract (referencing the whitelist) will not settle any exchange
// that involves the frozen Token Owner's public address.
type Freeze struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	Timestamp    uint64
	Qty          uint64
	Expiration   uint64
	Message      []byte
}

// NewFreeze returns a new Freeze with defaults set.
func NewFreeze() Freeze {
	return Freeze{
		Header:       []byte{0x6a, 0x4c, 0x7f},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeFreeze),
	}
}

// Type returns the type identifer for this message.
func (m Freeze) Type() string {
	return CodeFreeze
}

// Len returns the byte size of this message.
func (m Freeze) Len() int64 {
	return int64(len(m.Header)) + 127
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Freeze) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Freeze) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Expiration); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 61)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Freeze) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.Timestamp)

	m.read(buf, &m.Qty)

	m.read(buf, &m.Expiration)

	m.Message = make([]byte, 61)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 130, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Freeze) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Freeze) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("Expiration:%v", m.Expiration))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Thaw : to be used to comply with contractual obligations or legal
// requirements. The Alleged Offender's tokens will be unfrozen to allow
// them to resume normal exchange and governance activities.
type Thaw struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	Timestamp    uint64
	Qty          uint64
	Message      []byte
}

// NewThaw returns a new Thaw with defaults set.
func NewThaw() Thaw {
	return Thaw{
		Header:       []byte{0x6a, 0x4c, 0x77},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeThaw),
	}
}

// Type returns the type identifer for this message.
func (m Thaw) Type() string {
	return CodeThaw
}

// Len returns the byte size of this message.
func (m Thaw) Len() int64 {
	return int64(len(m.Header)) + 119
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Thaw) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Thaw) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Qty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 61)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Thaw) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.Timestamp)

	m.read(buf, &m.Qty)

	m.Message = make([]byte, 61)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 122, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Thaw) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Thaw) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("Qty:%v", m.Qty))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Confiscation : to be used to comply with contractual obligations and/or
// legal requirements.
type Confiscation struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	Timestamp    uint64
	TargetsQty   uint64
	DepositsQty  uint64
	Message      []byte
}

// NewConfiscation returns a new Confiscation with defaults set.
func NewConfiscation() Confiscation {
	return Confiscation{
		Header:       []byte{0x6a, 0x4c, 0x7f},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeConfiscation),
	}
}

// Type returns the type identifer for this message.
func (m Confiscation) Type() string {
	return CodeConfiscation
}

// Len returns the byte size of this message.
func (m Confiscation) Len() int64 {
	return int64(len(m.Header)) + 127
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Confiscation) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Confiscation) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.TargetsQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DepositsQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 61)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Confiscation) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.Timestamp)

	m.read(buf, &m.TargetsQty)

	m.read(buf, &m.DepositsQty)

	m.Message = make([]byte, 61)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 130, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Confiscation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Confiscation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("TargetsQty:%v", m.TargetsQty))

	vals = append(vals, fmt.Sprintf("DepositsQty:%v", m.DepositsQty))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Reconciliation : to be used at the direction of the issuer to fix record
// keeping errors with bitcoin and token balances.
type Reconciliation struct {
	BaseMessage
	Header           []byte
	ProtocolID       uint32
	ActionPrefix     []byte
	Version          uint8
	AssetType        []byte
	AssetID          []byte
	RefTxnID         []byte
	TargetAddressQty uint64
	Timestamp        uint64
	Message          []byte
}

// NewReconciliation returns a new Reconciliation with defaults set.
func NewReconciliation() Reconciliation {
	return Reconciliation{
		Header:       []byte{0x6a, 0x4c, 0x97},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeReconciliation),
	}
}

// Type returns the type identifer for this message.
func (m Reconciliation) Type() string {
	return CodeReconciliation
}

// Len returns the byte size of this message.
func (m Reconciliation) Len() int64 {
	return int64(len(m.Header)) + 151
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Reconciliation) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Reconciliation) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.RefTxnID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.TargetAddressQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 61)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Reconciliation) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.RefTxnID = make([]byte, 32)
	if err := m.readLen(buf, m.RefTxnID); err != nil {
		return 0, err
	}

	m.RefTxnID = bytes.Trim(m.RefTxnID, "\x00")

	m.read(buf, &m.TargetAddressQty)

	m.read(buf, &m.Timestamp)

	m.Message = make([]byte, 61)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 154, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Reconciliation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Reconciliation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("RefTxnID:\"%x\"", m.RefTxnID))

	vals = append(vals, fmt.Sprintf("TargetAddressQty:%v", m.TargetAddressQty))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Governance Operations
//

// Initiative : Allows Token Owners to propose a Initiative (aka
// Initiative/Shareholder vote). A significant cost - specified in the
// Contract Formation - is attached to this action to reduce spam, as the
// resulting vote will be put to all token owners.
type Initiative struct {
	BaseMessage
	Header               []byte
	ProtocolID           uint32
	ActionPrefix         []byte
	Version              uint8
	AssetType            []byte
	AssetID              []byte
	VoteType             byte
	VoteOptions          []byte
	VoteMax              uint8
	VoteLogic            byte
	ProposalDescription  []byte
	ProposalDocumentHash []byte
	VoteCutOffTimestamp  uint64
}

// NewInitiative returns a new Initiative with defaults set.
func NewInitiative() Initiative {
	return Initiative{
		Header:       []byte{0x6a, 0x4c, 0xb6},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeInitiative),
	}
}

// Type returns the type identifer for this message.
func (m Initiative) Type() string {
	return CodeInitiative
}

// Len returns the byte size of this message.
func (m Initiative) Len() int64 {
	return int64(len(m.Header)) + 182
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Initiative) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Initiative) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteOptions, 15)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMax); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteLogic); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDescription, 82)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDocumentHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteCutOffTimestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Initiative) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.VoteType)

	m.VoteOptions = make([]byte, 15)
	if err := m.readLen(buf, m.VoteOptions); err != nil {
		return 0, err
	}

	m.VoteOptions = bytes.Trim(m.VoteOptions, "\x00")

	m.read(buf, &m.VoteMax)

	m.read(buf, &m.VoteLogic)

	m.ProposalDescription = make([]byte, 82)
	if err := m.readLen(buf, m.ProposalDescription); err != nil {
		return 0, err
	}

	m.ProposalDescription = bytes.Trim(m.ProposalDescription, "\x00")

	m.ProposalDocumentHash = make([]byte, 32)
	if err := m.readLen(buf, m.ProposalDocumentHash); err != nil {
		return 0, err
	}

	m.ProposalDocumentHash = bytes.Trim(m.ProposalDocumentHash, "\x00")

	m.read(buf, &m.VoteCutOffTimestamp)

	return 185, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Initiative) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Initiative) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteType:\"%v\"", string(m.VoteType)))

	vals = append(vals, fmt.Sprintf("VoteOptions:\"%v\"", string(m.VoteOptions)))

	vals = append(vals, fmt.Sprintf("VoteMax:%v", m.VoteMax))

	vals = append(vals, fmt.Sprintf("VoteLogic:\"%v\"", string(m.VoteLogic)))

	vals = append(vals, fmt.Sprintf("ProposalDescription:\"%v\"", string(m.ProposalDescription)))

	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:\"%x\"", m.ProposalDocumentHash))

	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%v", m.VoteCutOffTimestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Referendum : Issuer instructs the Contract to Initiate a Token Owner
// Vote. Usually used for contract amendments, organizational governance,
// etc.
type Referendum struct {
	BaseMessage
	Header               []byte
	ProtocolID           uint32
	ActionPrefix         []byte
	Version              uint8
	AssetType            []byte
	AssetID              []byte
	VoteType             byte
	VoteOptions          []byte
	VoteMax              uint8
	VoteLogic            byte
	ProposalDescription  []byte
	ProposalDocumentHash []byte
	VoteCutOffTimestamp  uint64
}

// NewReferendum returns a new Referendum with defaults set.
func NewReferendum() Referendum {
	return Referendum{
		Header:       []byte{0x6a, 0x4c, 0xb6},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeReferendum),
	}
}

// Type returns the type identifer for this message.
func (m Referendum) Type() string {
	return CodeReferendum
}

// Len returns the byte size of this message.
func (m Referendum) Len() int64 {
	return int64(len(m.Header)) + 182
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Referendum) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Referendum) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteOptions, 15)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMax); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteLogic); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDescription, 82)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDocumentHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteCutOffTimestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Referendum) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.VoteType)

	m.VoteOptions = make([]byte, 15)
	if err := m.readLen(buf, m.VoteOptions); err != nil {
		return 0, err
	}

	m.VoteOptions = bytes.Trim(m.VoteOptions, "\x00")

	m.read(buf, &m.VoteMax)

	m.read(buf, &m.VoteLogic)

	m.ProposalDescription = make([]byte, 82)
	if err := m.readLen(buf, m.ProposalDescription); err != nil {
		return 0, err
	}

	m.ProposalDescription = bytes.Trim(m.ProposalDescription, "\x00")

	m.ProposalDocumentHash = make([]byte, 32)
	if err := m.readLen(buf, m.ProposalDocumentHash); err != nil {
		return 0, err
	}

	m.ProposalDocumentHash = bytes.Trim(m.ProposalDocumentHash, "\x00")

	m.read(buf, &m.VoteCutOffTimestamp)

	return 185, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Referendum) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Referendum) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteType:\"%v\"", string(m.VoteType)))

	vals = append(vals, fmt.Sprintf("VoteOptions:\"%v\"", string(m.VoteOptions)))

	vals = append(vals, fmt.Sprintf("VoteMax:%v", m.VoteMax))

	vals = append(vals, fmt.Sprintf("VoteLogic:\"%v\"", string(m.VoteLogic)))

	vals = append(vals, fmt.Sprintf("ProposalDescription:\"%v\"", string(m.ProposalDescription)))

	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:\"%x\"", m.ProposalDocumentHash))

	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%v", m.VoteCutOffTimestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Vote : A vote is created by the Contract in response to a valid
// Referendum (Issuer) or Initiative (User) Action. Votes can be made by
// Token Owners.
type Vote struct {
	BaseMessage
	Header               []byte
	ProtocolID           uint32
	ActionPrefix         []byte
	Version              uint8
	AssetType            []byte
	AssetID              []byte
	VoteType             byte
	VoteOptions          []byte
	VoteMax              uint8
	VoteLogic            byte
	ProposalDescription  []byte
	ProposalDocumentHash []byte
	VoteCutOffTimestamp  uint64
	Timestamp            uint64
}

// NewVote returns a new Vote with defaults set.
func NewVote() Vote {
	return Vote{
		Header:       []byte{0x6a, 0x4c, 0xbe},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeVote),
	}
}

// Type returns the type identifer for this message.
func (m Vote) Type() string {
	return CodeVote
}

// Len returns the byte size of this message.
func (m Vote) Len() int64 {
	return int64(len(m.Header)) + 190
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Vote) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Vote) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteOptions, 15)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteMax); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteLogic); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDescription, 82)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ProposalDocumentHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteCutOffTimestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Vote) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.VoteType)

	m.VoteOptions = make([]byte, 15)
	if err := m.readLen(buf, m.VoteOptions); err != nil {
		return 0, err
	}

	m.VoteOptions = bytes.Trim(m.VoteOptions, "\x00")

	m.read(buf, &m.VoteMax)

	m.read(buf, &m.VoteLogic)

	m.ProposalDescription = make([]byte, 82)
	if err := m.readLen(buf, m.ProposalDescription); err != nil {
		return 0, err
	}

	m.ProposalDescription = bytes.Trim(m.ProposalDescription, "\x00")

	m.ProposalDocumentHash = make([]byte, 32)
	if err := m.readLen(buf, m.ProposalDocumentHash); err != nil {
		return 0, err
	}

	m.ProposalDocumentHash = bytes.Trim(m.ProposalDocumentHash, "\x00")

	m.read(buf, &m.VoteCutOffTimestamp)

	m.read(buf, &m.Timestamp)

	return 193, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Vote) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Vote) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteType:\"%v\"", string(m.VoteType)))

	vals = append(vals, fmt.Sprintf("VoteOptions:\"%v\"", string(m.VoteOptions)))

	vals = append(vals, fmt.Sprintf("VoteMax:%v", m.VoteMax))

	vals = append(vals, fmt.Sprintf("VoteLogic:\"%v\"", string(m.VoteLogic)))

	vals = append(vals, fmt.Sprintf("ProposalDescription:\"%v\"", string(m.ProposalDescription)))

	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:\"%x\"", m.ProposalDocumentHash))

	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%v", m.VoteCutOffTimestamp))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCast : Used to allow Token Owners to cast their ballot (vote) on
// proposals raised by the Issuer or other token holders. 1 Vote per token
// unless a vote multiplier is specified in the relevant Asset Definition
// action.
type BallotCast struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	VoteTxnID    []byte
	Vote         []byte
}

// NewBallotCast returns a new BallotCast with defaults set.
func NewBallotCast() BallotCast {
	return BallotCast{
		Header:       []byte{0x6a, 0x4c, 0x59},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeBallotCast),
	}
}

// Type returns the type identifer for this message.
func (m BallotCast) Type() string {
	return CodeBallotCast
}

// Len returns the byte size of this message.
func (m BallotCast) Len() int64 {
	return int64(len(m.Header)) + 89
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m BallotCast) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m BallotCast) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteTxnID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Vote, 15)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *BallotCast) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.VoteTxnID = make([]byte, 32)
	if err := m.readLen(buf, m.VoteTxnID); err != nil {
		return 0, err
	}

	m.VoteTxnID = bytes.Trim(m.VoteTxnID, "\x00")

	m.Vote = make([]byte, 15)
	if err := m.readLen(buf, m.Vote); err != nil {
		return 0, err
	}

	m.Vote = bytes.Trim(m.Vote, "\x00")

	return 92, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m BallotCast) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m BallotCast) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteTxnID:\"%x\"", m.VoteTxnID))

	vals = append(vals, fmt.Sprintf("Vote:\"%v\"", string(m.Vote)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCounted : The smart contract will respond to a Ballot Cast action
// with a Ballot Counted action if the Ballot Cast is valid. If the Ballot
// Cast is not valid, then the smart contract will respond with a Rejection
// Action.
type BallotCounted struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	VoteTxnID    []byte
	Vote         []byte
	Timestamp    uint64
}

// NewBallotCounted returns a new BallotCounted with defaults set.
func NewBallotCounted() BallotCounted {
	return BallotCounted{
		Header:       []byte{0x6a, 0x4c, 0x62},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeBallotCounted),
	}
}

// Type returns the type identifer for this message.
func (m BallotCounted) Type() string {
	return CodeBallotCounted
}

// Len returns the byte size of this message.
func (m BallotCounted) Len() int64 {
	return int64(len(m.Header)) + 98
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m BallotCounted) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m BallotCounted) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteTxnID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Vote, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *BallotCounted) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.VoteTxnID = make([]byte, 32)
	if err := m.readLen(buf, m.VoteTxnID); err != nil {
		return 0, err
	}

	m.VoteTxnID = bytes.Trim(m.VoteTxnID, "\x00")

	m.Vote = make([]byte, 16)
	if err := m.readLen(buf, m.Vote); err != nil {
		return 0, err
	}

	m.Vote = bytes.Trim(m.Vote, "\x00")

	m.read(buf, &m.Timestamp)

	return 101, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m BallotCounted) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m BallotCounted) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteTxnID:\"%x\"", m.VoteTxnID))

	vals = append(vals, fmt.Sprintf("Vote:\"%v\"", string(m.Vote)))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Result : Once a vote has been completed the results are published.
type Result struct {
	BaseMessage
	Header        []byte
	ProtocolID    uint32
	ActionPrefix  []byte
	Version       uint8
	AssetType     []byte
	AssetID       []byte
	VoteType      byte
	VoteTxnID     []byte
	Timestamp     uint64
	Option1Tally  uint64
	Option2Tally  uint64
	Option3Tally  uint64
	Option4Tally  uint64
	Option5Tally  uint64
	Option6Tally  uint64
	Option7Tally  uint64
	Option8Tally  uint64
	Option9Tally  uint64
	Option10Tally uint64
	Option11Tally uint64
	Option12Tally uint64
	Option13Tally uint64
	Option14Tally uint64
	Option15Tally uint64
	Result        []byte
}

// NewResult returns a new Result with defaults set.
func NewResult() Result {
	return Result{
		Header:       []byte{0x6a, 0x4c, 0xdb},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeResult),
	}
}

// Type returns the type identifer for this message.
func (m Result) Type() string {
	return CodeResult
}

// Len returns the byte size of this message.
func (m Result) Len() int64 {
	return int64(len(m.Header)) + 219
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Result) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Result) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.VoteType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.VoteTxnID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option1Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option2Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option3Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option4Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option5Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option6Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option7Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option8Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option9Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option10Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option11Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option12Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option13Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option14Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Option15Tally); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Result, 16)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Result) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.VoteType)

	m.VoteTxnID = make([]byte, 32)
	if err := m.readLen(buf, m.VoteTxnID); err != nil {
		return 0, err
	}

	m.VoteTxnID = bytes.Trim(m.VoteTxnID, "\x00")

	m.read(buf, &m.Timestamp)

	m.read(buf, &m.Option1Tally)

	m.read(buf, &m.Option2Tally)

	m.read(buf, &m.Option3Tally)

	m.read(buf, &m.Option4Tally)

	m.read(buf, &m.Option5Tally)

	m.read(buf, &m.Option6Tally)

	m.read(buf, &m.Option7Tally)

	m.read(buf, &m.Option8Tally)

	m.read(buf, &m.Option9Tally)

	m.read(buf, &m.Option10Tally)

	m.read(buf, &m.Option11Tally)

	m.read(buf, &m.Option12Tally)

	m.read(buf, &m.Option13Tally)

	m.read(buf, &m.Option14Tally)

	m.read(buf, &m.Option15Tally)

	m.Result = make([]byte, 16)
	if err := m.readLen(buf, m.Result); err != nil {
		return 0, err
	}

	m.Result = bytes.Trim(m.Result, "\x00")

	return 222, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Result) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Result) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("VoteType:\"%v\"", string(m.VoteType)))

	vals = append(vals, fmt.Sprintf("VoteTxnID:\"%x\"", m.VoteTxnID))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("Option1Tally:%v", m.Option1Tally))

	vals = append(vals, fmt.Sprintf("Option2Tally:%v", m.Option2Tally))

	vals = append(vals, fmt.Sprintf("Option3Tally:%v", m.Option3Tally))

	vals = append(vals, fmt.Sprintf("Option4Tally:%v", m.Option4Tally))

	vals = append(vals, fmt.Sprintf("Option5Tally:%v", m.Option5Tally))

	vals = append(vals, fmt.Sprintf("Option6Tally:%v", m.Option6Tally))

	vals = append(vals, fmt.Sprintf("Option7Tally:%v", m.Option7Tally))

	vals = append(vals, fmt.Sprintf("Option8Tally:%v", m.Option8Tally))

	vals = append(vals, fmt.Sprintf("Option9Tally:%v", m.Option9Tally))

	vals = append(vals, fmt.Sprintf("Option10Tally:%v", m.Option10Tally))

	vals = append(vals, fmt.Sprintf("Option11Tally:%v", m.Option11Tally))

	vals = append(vals, fmt.Sprintf("Option12Tally:%v", m.Option12Tally))

	vals = append(vals, fmt.Sprintf("Option13Tally:%v", m.Option13Tally))

	vals = append(vals, fmt.Sprintf("Option14Tally:%v", m.Option14Tally))

	vals = append(vals, fmt.Sprintf("Option15Tally:%v", m.Option15Tally))

	vals = append(vals, fmt.Sprintf("Result:\"%v\"", string(m.Result)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Messaging Operations
//

// Message : the message action is a general purpose communication action.
// 'Twitter/sms' for Issuers/Investors/Users. The message txn can also be
// used for passing partially signed txns on-chain, establishing private
// communication channels including receipting, invoices, PO, and private
// offers/bids. The messages are broken down by type for easy filtering in
// the a user’s wallet. The Message Types are listed in the Message Types
// table.
type Message struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	Timestamp    uint64
	MessageType  []byte
	Message      []byte
}

// NewMessage returns a new Message with defaults set.
func NewMessage() Message {
	return Message{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeMessage),
	}
}

// Type returns the type identifer for this message.
func (m Message) Type() string {
	return CodeMessage
}

// Len returns the byte size of this message.
func (m Message) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Message) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Message) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.MessageType, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 203)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Message) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.read(buf, &m.Timestamp)

	m.MessageType = make([]byte, 2)
	if err := m.readLen(buf, m.MessageType); err != nil {
		return 0, err
	}

	m.MessageType = bytes.Trim(m.MessageType, "\x00")

	m.Message = make([]byte, 203)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Message) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Message) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("MessageType:\"%v\"", string(m.MessageType)))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Rejection : used to reject Exchange, Send, Initiative, Referendum,
// Order, and Ballot Cast actions that do not comply with the Contract. If
// money is to be returned to a User then it is used in lieu of the
// Settlement Action to properly account for token balances. All
// Issuer/User Actions must be responded to by the Contract with an Action.
// The only exception to this rule is when there is not enough fees in the
// first Action for the Contract response action to remain revenue neutral.
// If not enough fees are attached to pay for the Contract response then
// the Contract will not respond. For example
type Rejection struct {
	BaseMessage
	Header        []byte
	ProtocolID    uint32
	ActionPrefix  []byte
	Version       uint8
	Timestamp     uint64
	AssetType     []byte
	AssetID       []byte
	RejectionType byte
	Message       []byte
}

// NewRejection returns a new Rejection with defaults set.
func NewRejection() Rejection {
	return Rejection{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeRejection),
	}
}

// Type returns the type identifer for this message.
func (m Rejection) Type() string {
	return CodeRejection
}

// Len returns the byte size of this message.
func (m Rejection) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Rejection) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Rejection) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.RejectionType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 169)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Rejection) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.read(buf, &m.Timestamp)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.RejectionType)

	m.Message = make([]byte, 169)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Rejection) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Rejection) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("RejectionType:\"%v\"", string(m.RejectionType)))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Registry Operations
//

// Establishment : Establishes a register. The register is intended to be
// used primarily for whitelisting. However, other types of registers can
// be used.
type Establishment struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	Registrar                   []byte
	RegisterType                byte
	KYCJurisdiction             []byte
	DOB                         uint64
	CountryOfResidence          []byte
	SupportingDocumentationHash []byte
	Message                     []byte
}

// NewEstablishment returns a new Establishment with defaults set.
func NewEstablishment() Establishment {
	return Establishment{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeEstablishment),
	}
}

// Type returns the type identifer for this message.
func (m Establishment) Type() string {
	return CodeEstablishment
}

// Len returns the byte size of this message.
func (m Establishment) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Establishment) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Establishment) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Registrar, 16)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.RegisterType); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.KYCJurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DOB); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.CountryOfResidence, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.SupportingDocumentationHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 148)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Establishment) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.Registrar = make([]byte, 16)
	if err := m.readLen(buf, m.Registrar); err != nil {
		return 0, err
	}

	m.Registrar = bytes.Trim(m.Registrar, "\x00")

	m.read(buf, &m.RegisterType)

	m.KYCJurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.KYCJurisdiction); err != nil {
		return 0, err
	}

	m.KYCJurisdiction = bytes.Trim(m.KYCJurisdiction, "\x00")

	m.read(buf, &m.DOB)

	m.CountryOfResidence = make([]byte, 3)
	if err := m.readLen(buf, m.CountryOfResidence); err != nil {
		return 0, err
	}

	m.CountryOfResidence = bytes.Trim(m.CountryOfResidence, "\x00")

	m.SupportingDocumentationHash = make([]byte, 32)
	if err := m.readLen(buf, m.SupportingDocumentationHash); err != nil {
		return 0, err
	}

	m.SupportingDocumentationHash = bytes.Trim(m.SupportingDocumentationHash, "\x00")

	m.Message = make([]byte, 148)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Establishment) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Establishment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Registrar:\"%v\"", string(m.Registrar)))

	vals = append(vals, fmt.Sprintf("RegisterType:\"%v\"", string(m.RegisterType)))

	vals = append(vals, fmt.Sprintf("KYCJurisdiction:\"%v\"", string(m.KYCJurisdiction)))

	vals = append(vals, fmt.Sprintf("DOB:%v", m.DOB))

	vals = append(vals, fmt.Sprintf("CountryOfResidence:\"%v\"", string(m.CountryOfResidence)))

	vals = append(vals, fmt.Sprintf("SupportingDocumentationHash:\"%x\"", m.SupportingDocumentationHash))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Addition : Adds a User's public address to a global distributed
// whitelist. Entities (eg. Issuer) can filter by the public address of
// known and trusted entities (eg. KYC Databases such as coinbase) and
// therefore are able to create sublists - or subsets - of the main global
// whitelist.
type Addition struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	Sublist                     []byte
	KYC                         byte
	KYCJurisdiction             []byte
	DOB                         uint64
	CountryOfResidence          []byte
	SupportingDocumentationHash []byte
	Message                     []byte
}

// NewAddition returns a new Addition with defaults set.
func NewAddition() Addition {
	return Addition{
		Header:       []byte{0x6a, 0x4c, 0xd0},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeAddition),
	}
}

// Type returns the type identifer for this message.
func (m Addition) Type() string {
	return CodeAddition
}

// Len returns the byte size of this message.
func (m Addition) Len() int64 {
	return int64(len(m.Header)) + 208
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Addition) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Addition) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Sublist, 4)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.KYC); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.KYCJurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DOB); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.CountryOfResidence, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.SupportingDocumentationHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 148)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Addition) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.Sublist = make([]byte, 4)
	if err := m.readLen(buf, m.Sublist); err != nil {
		return 0, err
	}

	m.Sublist = bytes.Trim(m.Sublist, "\x00")

	m.read(buf, &m.KYC)

	m.KYCJurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.KYCJurisdiction); err != nil {
		return 0, err
	}

	m.KYCJurisdiction = bytes.Trim(m.KYCJurisdiction, "\x00")

	m.read(buf, &m.DOB)

	m.CountryOfResidence = make([]byte, 3)
	if err := m.readLen(buf, m.CountryOfResidence); err != nil {
		return 0, err
	}

	m.CountryOfResidence = bytes.Trim(m.CountryOfResidence, "\x00")

	m.SupportingDocumentationHash = make([]byte, 32)
	if err := m.readLen(buf, m.SupportingDocumentationHash); err != nil {
		return 0, err
	}

	m.SupportingDocumentationHash = bytes.Trim(m.SupportingDocumentationHash, "\x00")

	m.Message = make([]byte, 148)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 211, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Addition) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Addition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Sublist:\"%v\"", string(m.Sublist)))

	vals = append(vals, fmt.Sprintf("KYC:\"%v\"", string(m.KYC)))

	vals = append(vals, fmt.Sprintf("KYCJurisdiction:\"%v\"", string(m.KYCJurisdiction)))

	vals = append(vals, fmt.Sprintf("DOB:%v", m.DOB))

	vals = append(vals, fmt.Sprintf("CountryOfResidence:\"%v\"", string(m.CountryOfResidence)))

	vals = append(vals, fmt.Sprintf("SupportingDocumentationHash:\"%x\"", m.SupportingDocumentationHash))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Alteration : A registry entry can be altered.
type Alteration struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	Sublist                     []byte
	KYC                         byte
	KYCJurisdiction             []byte
	DOB                         uint64
	CountryOfResidence          []byte
	SupportingDocumentationHash []byte
	Message                     []byte
}

// NewAlteration returns a new Alteration with defaults set.
func NewAlteration() Alteration {
	return Alteration{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeAlteration),
	}
}

// Type returns the type identifer for this message.
func (m Alteration) Type() string {
	return CodeAlteration
}

// Len returns the byte size of this message.
func (m Alteration) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Alteration) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Alteration) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Sublist, 4)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.KYC); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.KYCJurisdiction, 5)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.DOB); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.CountryOfResidence, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.SupportingDocumentationHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 160)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Alteration) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.Sublist = make([]byte, 4)
	if err := m.readLen(buf, m.Sublist); err != nil {
		return 0, err
	}

	m.Sublist = bytes.Trim(m.Sublist, "\x00")

	m.read(buf, &m.KYC)

	m.KYCJurisdiction = make([]byte, 5)
	if err := m.readLen(buf, m.KYCJurisdiction); err != nil {
		return 0, err
	}

	m.KYCJurisdiction = bytes.Trim(m.KYCJurisdiction, "\x00")

	m.read(buf, &m.DOB)

	m.CountryOfResidence = make([]byte, 3)
	if err := m.readLen(buf, m.CountryOfResidence); err != nil {
		return 0, err
	}

	m.CountryOfResidence = bytes.Trim(m.CountryOfResidence, "\x00")

	m.SupportingDocumentationHash = make([]byte, 32)
	if err := m.readLen(buf, m.SupportingDocumentationHash); err != nil {
		return 0, err
	}

	m.SupportingDocumentationHash = bytes.Trim(m.SupportingDocumentationHash, "\x00")

	m.Message = make([]byte, 160)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Alteration) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Alteration) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Sublist:\"%v\"", string(m.Sublist)))

	vals = append(vals, fmt.Sprintf("KYC:\"%v\"", string(m.KYC)))

	vals = append(vals, fmt.Sprintf("KYCJurisdiction:\"%v\"", string(m.KYCJurisdiction)))

	vals = append(vals, fmt.Sprintf("DOB:%v", m.DOB))

	vals = append(vals, fmt.Sprintf("CountryOfResidence:\"%v\"", string(m.CountryOfResidence)))

	vals = append(vals, fmt.Sprintf("SupportingDocumentationHash:\"%x\"", m.SupportingDocumentationHash))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Removal : Removes a User's public address from the global distributed
// whitelist.
type Removal struct {
	BaseMessage
	Header                      []byte
	ProtocolID                  uint32
	ActionPrefix                []byte
	Version                     uint8
	SupportingDocumentationHash []byte
	Message                     []byte
}

// NewRemoval returns a new Removal with defaults set.
func NewRemoval() Removal {
	return Removal{
		Header:       []byte{0x6a, 0x4c, 0xdc},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeRemoval),
	}
}

// Type returns the type identifer for this message.
func (m Removal) Type() string {
	return CodeRemoval
}

// Len returns the byte size of this message.
func (m Removal) Len() int64 {
	return int64(len(m.Header)) + 220
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Removal) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Removal) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.SupportingDocumentationHash, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Message, 181)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Removal) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.SupportingDocumentationHash = make([]byte, 32)
	if err := m.readLen(buf, m.SupportingDocumentationHash); err != nil {
		return 0, err
	}

	m.SupportingDocumentationHash = bytes.Trim(m.SupportingDocumentationHash, "\x00")

	m.Message = make([]byte, 181)
	if err := m.readLen(buf, m.Message); err != nil {
		return 0, err
	}

	m.Message = bytes.Trim(m.Message, "\x00")

	return 223, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Removal) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Removal) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("SupportingDocumentationHash:\"%x\"", m.SupportingDocumentationHash))

	vals = append(vals, fmt.Sprintf("Message:\"%v\"", string(m.Message)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

//
// Transfer Operations
//

// Send : A Token Owner Sends a Token to a Receiver. The Send Action
// requires no sign-off by the Token Receiving Party and does not provide
// any on-chain consideration to the Token Sending Party. Can be used for
// User Revocation (remove tokens from wallet by sending back to Issuer).
// Can be used for redeeming a ticket, however, it is probably better for
// most ticket use cases to use the exchange action for ticket redemption.
type Send struct {
	BaseMessage
	Header       []byte
	ProtocolID   uint32
	ActionPrefix []byte
	Version      uint8
	AssetType    []byte
	AssetID      []byte
	TokenQty     uint64
}

// NewSend returns a new Send with defaults set.
func NewSend() Send {
	return Send{
		Header:       []byte{0x6a, 0x32},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeSend),
	}
}

// Type returns the type identifer for this message.
func (m Send) Type() string {
	return CodeSend
}

// Len returns the byte size of this message.
func (m Send) Len() int64 {
	return int64(len(m.Header)) + 50
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Send) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Send) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.TokenQty); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Send) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 2)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.TokenQty)

	return 52, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Send) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Send) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("TokenQty:%v", m.TokenQty))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Exchange : Example
type Exchange struct {
	BaseMessage
	Header              []byte
	ProtocolID          uint32
	ActionPrefix        []byte
	Version             uint8
	Party1AssetType     []byte
	Party1AssetID       []byte
	Party1TokenQty      uint64
	OfferValidUntil     uint64
	ExchangeFeeCurrency []byte
	ExchangeFeeVar      float32
	ExchangeFeeFixed    float32
	ExchangeFeeAddress  []byte
}

// NewExchange returns a new Exchange with defaults set.
func NewExchange() Exchange {
	return Exchange{
		Header:       []byte{0x6a, 0x4c, 0x67},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeExchange),
	}
}

// Type returns the type identifer for this message.
func (m Exchange) Type() string {
	return CodeExchange
}

// Len returns the byte size of this message.
func (m Exchange) Len() int64 {
	return int64(len(m.Header)) + 103
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Exchange) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Exchange) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party1AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party1AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Party1TokenQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.OfferValidUntil); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ExchangeFeeCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExchangeFeeVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExchangeFeeFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ExchangeFeeAddress, 34)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Exchange) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.Party1AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.Party1AssetType); err != nil {
		return 0, err
	}

	m.Party1AssetType = bytes.Trim(m.Party1AssetType, "\x00")

	m.Party1AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.Party1AssetID); err != nil {
		return 0, err
	}

	m.Party1AssetID = bytes.Trim(m.Party1AssetID, "\x00")

	m.read(buf, &m.Party1TokenQty)

	m.read(buf, &m.OfferValidUntil)

	m.ExchangeFeeCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.ExchangeFeeCurrency); err != nil {
		return 0, err
	}

	m.ExchangeFeeCurrency = bytes.Trim(m.ExchangeFeeCurrency, "\x00")

	m.read(buf, &m.ExchangeFeeVar)

	m.read(buf, &m.ExchangeFeeFixed)

	m.ExchangeFeeAddress = make([]byte, 34)
	if err := m.readLen(buf, m.ExchangeFeeAddress); err != nil {
		return 0, err
	}

	m.ExchangeFeeAddress = bytes.Trim(m.ExchangeFeeAddress, "\x00")

	return 106, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Exchange) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Exchange) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Party1AssetType:\"%v\"", string(m.Party1AssetType)))

	vals = append(vals, fmt.Sprintf("Party1AssetID:\"%v\"", string(m.Party1AssetID)))

	vals = append(vals, fmt.Sprintf("Party1TokenQty:%v", m.Party1TokenQty))

	vals = append(vals, fmt.Sprintf("OfferValidUntil:%v", m.OfferValidUntil))

	vals = append(vals, fmt.Sprintf("ExchangeFeeCurrency:\"%v\"", string(m.ExchangeFeeCurrency)))

	vals = append(vals, fmt.Sprintf("ExchangeFeeVar:%v", m.ExchangeFeeVar))

	vals = append(vals, fmt.Sprintf("ExchangeFeeFixed:%v", m.ExchangeFeeFixed))

	vals = append(vals, fmt.Sprintf("ExchangeFeeAddress:\"%v\"", string(m.ExchangeFeeAddress)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Swap : Two parties want to swap a token (Atomic Swap) directly for
// another token. No BCH is used in the txn other than for paying the
// necessary network/transaction fees.
type Swap struct {
	BaseMessage
	Header              []byte
	ProtocolID          uint32
	ActionPrefix        []byte
	Version             uint8
	Party1AssetType     []byte
	Party1AssetID       []byte
	Party1TokenQty      uint64
	OfferValidUntil     uint64
	Party2AssetType     []byte
	Party2AssetID       []byte
	Party2TokenQty      uint64
	ExchangeFeeCurrency []byte
	ExchangeFeeVar      float32
	ExchangeFeeFixed    float32
	ExchangeFeeAddress  []byte
}

// NewSwap returns a new Swap with defaults set.
func NewSwap() Swap {
	return Swap{
		Header:       []byte{0x6a, 0x4c, 0x92},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeSwap),
	}
}

// Type returns the type identifer for this message.
func (m Swap) Type() string {
	return CodeSwap
}

// Len returns the byte size of this message.
func (m Swap) Len() int64 {
	return int64(len(m.Header)) + 146
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Swap) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Swap) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party1AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party1AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Party1TokenQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.OfferValidUntil); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party2AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.Party2AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Party2TokenQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ExchangeFeeCurrency, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExchangeFeeVar); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ExchangeFeeFixed); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ExchangeFeeAddress, 34)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Swap) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 3)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.Party1AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.Party1AssetType); err != nil {
		return 0, err
	}

	m.Party1AssetType = bytes.Trim(m.Party1AssetType, "\x00")

	m.Party1AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.Party1AssetID); err != nil {
		return 0, err
	}

	m.Party1AssetID = bytes.Trim(m.Party1AssetID, "\x00")

	m.read(buf, &m.Party1TokenQty)

	m.read(buf, &m.OfferValidUntil)

	m.Party2AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.Party2AssetType); err != nil {
		return 0, err
	}

	m.Party2AssetType = bytes.Trim(m.Party2AssetType, "\x00")

	m.Party2AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.Party2AssetID); err != nil {
		return 0, err
	}

	m.Party2AssetID = bytes.Trim(m.Party2AssetID, "\x00")

	m.read(buf, &m.Party2TokenQty)

	m.ExchangeFeeCurrency = make([]byte, 3)
	if err := m.readLen(buf, m.ExchangeFeeCurrency); err != nil {
		return 0, err
	}

	m.ExchangeFeeCurrency = bytes.Trim(m.ExchangeFeeCurrency, "\x00")

	m.read(buf, &m.ExchangeFeeVar)

	m.read(buf, &m.ExchangeFeeFixed)

	m.ExchangeFeeAddress = make([]byte, 34)
	if err := m.readLen(buf, m.ExchangeFeeAddress); err != nil {
		return 0, err
	}

	m.ExchangeFeeAddress = bytes.Trim(m.ExchangeFeeAddress, "\x00")

	return 149, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Swap) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Swap) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("Party1AssetType:\"%v\"", string(m.Party1AssetType)))

	vals = append(vals, fmt.Sprintf("Party1AssetID:\"%v\"", string(m.Party1AssetID)))

	vals = append(vals, fmt.Sprintf("Party1TokenQty:%v", m.Party1TokenQty))

	vals = append(vals, fmt.Sprintf("OfferValidUntil:%v", m.OfferValidUntil))

	vals = append(vals, fmt.Sprintf("Party2AssetType:\"%v\"", string(m.Party2AssetType)))

	vals = append(vals, fmt.Sprintf("Party2AssetID:\"%v\"", string(m.Party2AssetID)))

	vals = append(vals, fmt.Sprintf("Party2TokenQty:%v", m.Party2TokenQty))

	vals = append(vals, fmt.Sprintf("ExchangeFeeCurrency:\"%v\"", string(m.ExchangeFeeCurrency)))

	vals = append(vals, fmt.Sprintf("ExchangeFeeVar:%v", m.ExchangeFeeVar))

	vals = append(vals, fmt.Sprintf("ExchangeFeeFixed:%v", m.ExchangeFeeFixed))

	vals = append(vals, fmt.Sprintf("ExchangeFeeAddress:\"%v\"", string(m.ExchangeFeeAddress)))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Settlement : (to be used for finalizing the transfer of bitcoins and
// tokens from exchange, issuance, swap actions)
type Settlement struct {
	BaseMessage
	Header         []byte
	ProtocolID     uint32
	ActionPrefix   []byte
	Version        uint8
	AssetType      []byte
	AssetID        []byte
	Party1TokenQty uint64
	Party2TokenQty uint64
	Timestamp      uint64
}

// NewSettlement returns a new Settlement with defaults set.
func NewSettlement() Settlement {
	return Settlement{
		Header:       []byte{0x6a, 0x42},
		ProtocolID:   ProtocolID,
		ActionPrefix: []byte(CodeSettlement),
	}
}

// Type returns the type identifer for this message.
func (m Settlement) Type() string {
	return CodeSettlement
}

// Len returns the byte size of this message.
func (m Settlement) Len() int64 {
	return int64(len(m.Header)) + 66
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (m Settlement) Read(b []byte) (int, error) {
	data, err := m.Bytes()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// Bytes returns the full OP_RETURN payload in bytes.
func (m Settlement) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := m.write(buf, m.pad(m.Header, len(m.Header))); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.ProtocolID); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.ActionPrefix, 2)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Version); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetType, 3)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.pad(m.AssetID, 32)); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Party1TokenQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Party2TokenQty); err != nil {
		return nil, err
	}

	if err := m.write(buf, m.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (m *Settlement) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)

	m.Header = make([]byte, 2)
	if err := m.readLen(buf, m.Header); err != nil {
		return 0, err
	}

	m.Header = bytes.Trim(m.Header, "\x00")

	m.read(buf, &m.ProtocolID)

	m.ActionPrefix = make([]byte, 2)
	if err := m.readLen(buf, m.ActionPrefix); err != nil {
		return 0, err
	}

	m.ActionPrefix = bytes.Trim(m.ActionPrefix, "\x00")

	m.read(buf, &m.Version)

	m.AssetType = make([]byte, 3)
	if err := m.readLen(buf, m.AssetType); err != nil {
		return 0, err
	}

	m.AssetType = bytes.Trim(m.AssetType, "\x00")

	m.AssetID = make([]byte, 32)
	if err := m.readLen(buf, m.AssetID); err != nil {
		return 0, err
	}

	m.AssetID = bytes.Trim(m.AssetID, "\x00")

	m.read(buf, &m.Party1TokenQty)

	m.read(buf, &m.Party2TokenQty)

	m.read(buf, &m.Timestamp)

	return 68, nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (m Settlement) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (m Settlement) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ProtocolID:%v", m.ProtocolID))

	vals = append(vals, fmt.Sprintf("ActionPrefix:\"%v\"", string(m.ActionPrefix)))

	vals = append(vals, fmt.Sprintf("Version:%v", m.Version))

	vals = append(vals, fmt.Sprintf("AssetType:\"%v\"", string(m.AssetType)))

	vals = append(vals, fmt.Sprintf("AssetID:\"%v\"", string(m.AssetID)))

	vals = append(vals, fmt.Sprintf("Party1TokenQty:%v", m.Party1TokenQty))

	vals = append(vals, fmt.Sprintf("Party2TokenQty:%v", m.Party2TokenQty))

	vals = append(vals, fmt.Sprintf("Timestamp:%v", m.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
