package protocol

import (
	"bytes"
	"fmt"
	"strings"
)

const (
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

	// CodeStaticContractFormation identifies data as a
	// StaticContractFormation message.
	CodeStaticContractFormation = "C4"

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

	// CodeProposal identifies data as a Proposal message.
	CodeProposal = "G1"

	// CodeVote identifies data as a Vote message.
	CodeVote = "G2"

	// CodeBallotCast identifies data as a BallotCast message.
	CodeBallotCast = "G3"

	// CodeBallotCounted identifies data as a BallotCounted message.
	CodeBallotCounted = "G4"

	// CodeResult identifies data as a Result message.
	CodeResult = "G5"

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

	// CodeTransfer identifies data as a Transfer message.
	CodeTransfer = "T1"

	// CodeSettlement identifies data as a Settlement message.
	CodeSettlement = "T2"

	// ComplianceActionFreeze identifies a freeze type
	ComplianceActionFreeze = byte('F')

	// ComplianceActionThaw identifies a thaw type
	ComplianceActionThaw = byte('T')

	// ComplianceActionConfiscation identifies a confiscation type
	ComplianceActionConfiscation = byte('C')

	// ComplianceActionReconciliation identifies a reconcilation type
	ComplianceActionReconciliation = byte('R')
)

// TypeMapping holds a mapping of action codes to action types.
func TypeMapping(code string) OpReturnMessage {
	switch code {
	case CodeAssetDefinition:
		result := AssetDefinition{}
		return &result
	case CodeAssetCreation:
		result := AssetCreation{}
		return &result
	case CodeAssetModification:
		result := AssetModification{}
		return &result
	case CodeContractOffer:
		result := ContractOffer{}
		return &result
	case CodeContractFormation:
		result := ContractFormation{}
		return &result
	case CodeContractAmendment:
		result := ContractAmendment{}
		return &result
	case CodeStaticContractFormation:
		result := StaticContractFormation{}
		return &result
	case CodeOrder:
		result := Order{}
		return &result
	case CodeFreeze:
		result := Freeze{}
		return &result
	case CodeThaw:
		result := Thaw{}
		return &result
	case CodeConfiscation:
		result := Confiscation{}
		return &result
	case CodeReconciliation:
		result := Reconciliation{}
		return &result
	case CodeProposal:
		result := Proposal{}
		return &result
	case CodeVote:
		result := Vote{}
		return &result
	case CodeBallotCast:
		result := BallotCast{}
		return &result
	case CodeBallotCounted:
		result := BallotCounted{}
		return &result
	case CodeResult:
		result := Result{}
		return &result
	case CodeMessage:
		result := Message{}
		return &result
	case CodeRejection:
		result := Rejection{}
		return &result
	case CodeEstablishment:
		result := Establishment{}
		return &result
	case CodeAddition:
		result := Addition{}
		return &result
	case CodeAlteration:
		result := Alteration{}
		return &result
	case CodeRemoval:
		result := Removal{}
		return &result
	case CodeTransfer:
		result := Transfer{}
		return &result
	case CodeSettlement:
		result := Settlement{}
		return &result
	default:
		return nil
	}
}

// AssetDefinition This action is used by the issuer to define the
// properties/characteristics of the Asset (token) that it wants to create.
// An asset has a unique identifier called AssetID = AssetType +
// base58(AssetCode + checksum). An asset is always linked to a Contract
// that is identified by the public address of the Contract wallet.
type AssetDefinition struct {
	AssetType                   string    `json:"asset_type,omitempty"`                    // Three letter character that specifies the asset type.
	AssetCode                   AssetCode `json:"asset_code,omitempty"`                    // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetAuthFlags              []byte    `json:"asset_auth_flags,omitempty"`              // A set of switches that define the authorization rules for this asset. See the Authorization Flags documentation for more detail.
	TransfersPermitted          bool      `json:"transfers_permitted,omitempty"`           // Set to true if transfers are permitted between two parties, otherwise set to false to prevent peer-to-peer transfers.
	TradeRestrictions           []Polity  `json:"trade_restrictions,omitempty"`            // If specified, the asset can only be traded within the specified trade restriction zone. For example, AUS would restrict to Australian residents only.
	EnforcementOrdersPermitted  bool      `json:"enforcement_orders_permitted,omitempty"`  // Set to true if the issuer is permitted to make enforcement orders on user tokens and balances, otherwise set to false to disable this feature.
	VotingRights                bool      `json:"voting_rights,omitempty"`                 // When false holders of this asset will not be able to vote for tokens of this asset even on voting systems in which vote multiplers are not permitted.
	VoteMultiplier              uint8     `json:"vote_multiplier,omitempty"`               // Multiplies a vote by the specified integer. Where 1 token is equal to 1 vote with a 1 for vote multipler (normal), 1 token = 3 votes with a multiplier of 3, for example. If zero, then holders of this asset don't get any votes for their tokens.
	IssuerProposal              bool      `json:"issuer_proposal,omitempty"`               // Set to true if an Issuer is permitted to make proposals outside of the smart contract scope.
	HolderProposal              bool      `json:"holder_proposal,omitempty"`               // Set to true if a holder is permitted to make proposals outside of the smart contract scope.
	AssetModificationGovernance uint8     `json:"asset_modification_governance,omitempty"` // Supported values: 1 - Contract-wide Asset Governance. 0 - Asset-wide Asset Governance.  If a referendum or initiative is used to propose a modification to a subfield controlled by the asset auth flags, then the vote will either be a contract-wide vote (all assets vote on the referendum/initiative) or an asset-wide vote (only this asset votes on the referendum/initiative) depending on the value in this subfield.  The voting system specifies the voting rules.
	TokenQty                    uint64    `json:"token_qty,omitempty"`                     // The number of tokens to issue with this asset. Set to greater than zero for fungible tokens. If the value is 1 then it becomes a non-fungible token, where the contract would have many assets. Set to 0 to represent a placeholder asset, where tokens are to be issued later, or to represent a decomissioned asset where all tokens have been revoked.
	AssetPayload                []byte    `json:"asset_payload,omitempty"`                 // A custom payload that contains meta data about this asset. Payload structure and length is dependent on the asset type chosen. See Asset documentation for more details.
}

// Type returns the type identifer for this message.
func (action AssetDefinition) Type() string {
	return CodeAssetDefinition
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *AssetDefinition) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *AssetDefinition) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetAuthFlags ([]byte)
	{
		if err := WriteVarBin(buf, action.AssetAuthFlags, 8); err != nil {
			return nil, err
		}
	}

	// TransfersPermitted (bool)
	{
		if err := write(buf, action.TransfersPermitted); err != nil {
			return nil, err
		}
	}

	// TradeRestrictions ([]Polity)
	{
		if err := WriteVariableSize(buf, uint64(len(action.TradeRestrictions)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.TradeRestrictions {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
		if err := write(buf, action.EnforcementOrdersPermitted); err != nil {
			return nil, err
		}
	}

	// VotingRights (bool)
	{
		if err := write(buf, action.VotingRights); err != nil {
			return nil, err
		}
	}

	// VoteMultiplier (uint8)
	{
		if err := write(buf, action.VoteMultiplier); err != nil {
			return nil, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := write(buf, action.IssuerProposal); err != nil {
			return nil, err
		}
	}

	// HolderProposal (bool)
	{
		if err := write(buf, action.HolderProposal); err != nil {
			return nil, err
		}
	}

	// AssetModificationGovernance (uint8)
	{
		if err := write(buf, action.AssetModificationGovernance); err != nil {
			return nil, err
		}
	}

	// TokenQty (uint64)
	{
		if err := write(buf, action.TokenQty); err != nil {
			return nil, err
		}
	}

	// AssetPayload ([]byte)
	{
		if err := WriteVarBin(buf, action.AssetPayload, 16); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in AssetDefinition from the byte slice
func (action *AssetDefinition) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetAuthFlags ([]byte)
	{
		var err error
		action.AssetAuthFlags, err = ReadVarBin(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// TransfersPermitted (bool)
	{
		if err := read(buf, &action.TransfersPermitted); err != nil {
			return 0, err
		}
	}

	// TradeRestrictions ([]Polity)
	{
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.TradeRestrictions = make([]Polity, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Polity
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.TradeRestrictions = append(action.TradeRestrictions, newValue)
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
		if err := read(buf, &action.EnforcementOrdersPermitted); err != nil {
			return 0, err
		}
	}

	// VotingRights (bool)
	{
		if err := read(buf, &action.VotingRights); err != nil {
			return 0, err
		}
	}

	// VoteMultiplier (uint8)
	{
		if err := read(buf, &action.VoteMultiplier); err != nil {
			return 0, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := read(buf, &action.IssuerProposal); err != nil {
			return 0, err
		}
	}

	// HolderProposal (bool)
	{
		if err := read(buf, &action.HolderProposal); err != nil {
			return 0, err
		}
	}

	// AssetModificationGovernance (uint8)
	{
		if err := read(buf, &action.AssetModificationGovernance); err != nil {
			return 0, err
		}
	}

	// TokenQty (uint64)
	{
		if err := read(buf, &action.TokenQty); err != nil {
			return 0, err
		}
	}

	// AssetPayload ([]byte)
	{
		var err error
		action.AssetPayload, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *AssetDefinition) Validate() error {

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

	// AssetAuthFlags ([]byte)
	{
		if len(m.AssetAuthFlags) > (2<<8)-1 {
			return fmt.Errorf("varbin field AssetAuthFlags too long %d/%d", len(m.AssetAuthFlags), (2<<8)-1)
		}
	}

	// TransfersPermitted (bool)
	{
	}

	// TradeRestrictions ([]Polity)
	{
		if len(m.TradeRestrictions) > (2<<16)-1 {
			return fmt.Errorf("list field TradeRestrictions has too many items %d/%d", len(m.TradeRestrictions), (2<<16)-1)
		}

		for i, value := range m.TradeRestrictions {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field TradeRestrictions[%d] is invalid : %s", i, err)
			}
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
	}

	// VotingRights (bool)
	{
	}

	// VoteMultiplier (uint8)
	{
	}

	// IssuerProposal (bool)
	{
	}

	// HolderProposal (bool)
	{
	}

	// AssetModificationGovernance (uint8)
	{
		if m.AssetModificationGovernance != 0 && m.AssetModificationGovernance != 1 {
			return fmt.Errorf("field AssetModificationGovernance value is invalid : %d", m.AssetModificationGovernance)
		}

	}

	// TokenQty (uint64)
	{
	}

	// AssetPayload ([]byte)
	{
		if len(m.AssetPayload) > (2<<16)-1 {
			return fmt.Errorf("varbin field AssetPayload too long %d/%d", len(m.AssetPayload), (2<<16)-1)
		}
	}

	return nil
}

func (action AssetDefinition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetAuthFlags:%#x", action.AssetAuthFlags))
	vals = append(vals, fmt.Sprintf("TransfersPermitted:%#+v", action.TransfersPermitted))
	vals = append(vals, fmt.Sprintf("TradeRestrictions:%#+v", action.TradeRestrictions))
	vals = append(vals, fmt.Sprintf("EnforcementOrdersPermitted:%#+v", action.EnforcementOrdersPermitted))
	vals = append(vals, fmt.Sprintf("VotingRights:%#+v", action.VotingRights))
	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", action.VoteMultiplier))
	vals = append(vals, fmt.Sprintf("IssuerProposal:%#+v", action.IssuerProposal))
	vals = append(vals, fmt.Sprintf("HolderProposal:%#+v", action.HolderProposal))
	vals = append(vals, fmt.Sprintf("AssetModificationGovernance:%v", action.AssetModificationGovernance))
	vals = append(vals, fmt.Sprintf("TokenQty:%v", action.TokenQty))
	vals = append(vals, fmt.Sprintf("AssetPayload:%#x", action.AssetPayload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetCreation This action creates an Asset in response to the Issuer's
// instructions in the Definition Action.
type AssetCreation struct {
	AssetType                   string    `json:"asset_type,omitempty"`                    // Three letter character that specifies the asset type.
	AssetCode                   AssetCode `json:"asset_code,omitempty"`                    // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetAuthFlags              []byte    `json:"asset_auth_flags,omitempty"`              // A set of switches that define the authorization rules for this asset. See the Authorization Flags documentation for more detail.
	TransfersPermitted          bool      `json:"transfers_permitted,omitempty"`           // Set to true if transfers are permitted between two parties, otherwise set to false to prevent peer-to-peer transfers.
	TradeRestrictions           []Polity  `json:"trade_restrictions,omitempty"`            // If specified, the asset can only be traded within the specified trade restriction zone. For example, AUS would restrict to Australian residents only.
	EnforcementOrdersPermitted  bool      `json:"enforcement_orders_permitted,omitempty"`  // Set to true if the issuer is permitted to make enforcement orders on user tokens and balances, otherwise set to false to disable this feature.
	VotingRights                bool      `json:"voting_rights,omitempty"`                 // When false holders of this asset will not be able to vote for tokens of this asset even on voting systems in which vote multiplers are not permitted.
	VoteMultiplier              uint8     `json:"vote_multiplier,omitempty"`               // Multiplies a vote by the specified integer. Where 1 token is equal to 1 vote with a 1 for vote multipler (normal), 1 token = 3 votes with a multiplier of 3, for example. If zero, then holders of this asset don't get any votes for their tokens.
	IssuerProposal              bool      `json:"issuer_proposal,omitempty"`               // Set to true if an Issuer is permitted to make proposals outside of the smart contract scope.
	HolderProposal              bool      `json:"holder_proposal,omitempty"`               // Set to true if a holder is permitted to make proposals outside of the smart contract scope.
	AssetModificationGovernance uint8     `json:"asset_modification_governance,omitempty"` // Supported values: 1 - Contract-wide Asset Governance.  0 - Asset-wide Asset Governance.  If a referendum or initiative is used to propose a modification to a subfield controlled by the asset auth flags, then the vote will either be a contract-wide vote (all assets vote on the referendum/initiative) or an asset-wide vote (only this asset votes on the referendum/initiative) depending on the value in this subfield.  The voting system specifies the voting rules.
	TokenQty                    uint64    `json:"token_qty,omitempty"`                     // The number of tokens to issue with this asset. Set to greater than zero for fungible tokens. If the value is 1 then it becomes a non-fungible token, where the contract would have many assets. Set to 0 to represent a placeholder asset, where tokens are to be issued later, or to represent a decomissioned asset where all tokens have been revoked.
	AssetPayload                []byte    `json:"asset_payload,omitempty"`                 // A custom payload that contains meta data about this asset. Payload structure and length is dependent on the asset type chosen. See Asset documentation for more details.
	AssetRevision               uint32    `json:"asset_revision,omitempty"`                // A counter for the number of times this asset has been revised using a modification action.
	Timestamp                   Timestamp `json:"timestamp,omitempty"`                     // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action AssetCreation) Type() string {
	return CodeAssetCreation
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *AssetCreation) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *AssetCreation) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetAuthFlags ([]byte)
	{
		if err := WriteVarBin(buf, action.AssetAuthFlags, 8); err != nil {
			return nil, err
		}
	}

	// TransfersPermitted (bool)
	{
		if err := write(buf, action.TransfersPermitted); err != nil {
			return nil, err
		}
	}

	// TradeRestrictions ([]Polity)
	{
		if err := WriteVariableSize(buf, uint64(len(action.TradeRestrictions)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.TradeRestrictions {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
		if err := write(buf, action.EnforcementOrdersPermitted); err != nil {
			return nil, err
		}
	}

	// VotingRights (bool)
	{
		if err := write(buf, action.VotingRights); err != nil {
			return nil, err
		}
	}

	// VoteMultiplier (uint8)
	{
		if err := write(buf, action.VoteMultiplier); err != nil {
			return nil, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := write(buf, action.IssuerProposal); err != nil {
			return nil, err
		}
	}

	// HolderProposal (bool)
	{
		if err := write(buf, action.HolderProposal); err != nil {
			return nil, err
		}
	}

	// AssetModificationGovernance (uint8)
	{
		if err := write(buf, action.AssetModificationGovernance); err != nil {
			return nil, err
		}
	}

	// TokenQty (uint64)
	{
		if err := write(buf, action.TokenQty); err != nil {
			return nil, err
		}
	}

	// AssetPayload ([]byte)
	{
		if err := WriteVarBin(buf, action.AssetPayload, 16); err != nil {
			return nil, err
		}
	}

	// AssetRevision (uint32)
	{
		if err := write(buf, action.AssetRevision); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in AssetCreation from the byte slice
func (action *AssetCreation) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetAuthFlags ([]byte)
	{
		var err error
		action.AssetAuthFlags, err = ReadVarBin(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// TransfersPermitted (bool)
	{
		if err := read(buf, &action.TransfersPermitted); err != nil {
			return 0, err
		}
	}

	// TradeRestrictions ([]Polity)
	{
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.TradeRestrictions = make([]Polity, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Polity
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.TradeRestrictions = append(action.TradeRestrictions, newValue)
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
		if err := read(buf, &action.EnforcementOrdersPermitted); err != nil {
			return 0, err
		}
	}

	// VotingRights (bool)
	{
		if err := read(buf, &action.VotingRights); err != nil {
			return 0, err
		}
	}

	// VoteMultiplier (uint8)
	{
		if err := read(buf, &action.VoteMultiplier); err != nil {
			return 0, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := read(buf, &action.IssuerProposal); err != nil {
			return 0, err
		}
	}

	// HolderProposal (bool)
	{
		if err := read(buf, &action.HolderProposal); err != nil {
			return 0, err
		}
	}

	// AssetModificationGovernance (uint8)
	{
		if err := read(buf, &action.AssetModificationGovernance); err != nil {
			return 0, err
		}
	}

	// TokenQty (uint64)
	{
		if err := read(buf, &action.TokenQty); err != nil {
			return 0, err
		}
	}

	// AssetPayload ([]byte)
	{
		var err error
		action.AssetPayload, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// AssetRevision (uint32)
	{
		if err := read(buf, &action.AssetRevision); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *AssetCreation) Validate() error {

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

	// AssetAuthFlags ([]byte)
	{
		if len(m.AssetAuthFlags) > (2<<8)-1 {
			return fmt.Errorf("varbin field AssetAuthFlags too long %d/%d", len(m.AssetAuthFlags), (2<<8)-1)
		}
	}

	// TransfersPermitted (bool)
	{
	}

	// TradeRestrictions ([]Polity)
	{
		if len(m.TradeRestrictions) > (2<<16)-1 {
			return fmt.Errorf("list field TradeRestrictions has too many items %d/%d", len(m.TradeRestrictions), (2<<16)-1)
		}

		for i, value := range m.TradeRestrictions {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field TradeRestrictions[%d] is invalid : %s", i, err)
			}
		}
	}

	// EnforcementOrdersPermitted (bool)
	{
	}

	// VotingRights (bool)
	{
	}

	// VoteMultiplier (uint8)
	{
	}

	// IssuerProposal (bool)
	{
	}

	// HolderProposal (bool)
	{
	}

	// AssetModificationGovernance (uint8)
	{
		if m.AssetModificationGovernance != 0 && m.AssetModificationGovernance != 1 {
			return fmt.Errorf("field AssetModificationGovernance value is invalid : %d", m.AssetModificationGovernance)
		}

	}

	// TokenQty (uint64)
	{
	}

	// AssetPayload ([]byte)
	{
		if len(m.AssetPayload) > (2<<16)-1 {
			return fmt.Errorf("varbin field AssetPayload too long %d/%d", len(m.AssetPayload), (2<<16)-1)
		}
	}

	// AssetRevision (uint32)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action AssetCreation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetAuthFlags:%#x", action.AssetAuthFlags))
	vals = append(vals, fmt.Sprintf("TransfersPermitted:%#+v", action.TransfersPermitted))
	vals = append(vals, fmt.Sprintf("TradeRestrictions:%#+v", action.TradeRestrictions))
	vals = append(vals, fmt.Sprintf("EnforcementOrdersPermitted:%#+v", action.EnforcementOrdersPermitted))
	vals = append(vals, fmt.Sprintf("VotingRights:%#+v", action.VotingRights))
	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", action.VoteMultiplier))
	vals = append(vals, fmt.Sprintf("IssuerProposal:%#+v", action.IssuerProposal))
	vals = append(vals, fmt.Sprintf("HolderProposal:%#+v", action.HolderProposal))
	vals = append(vals, fmt.Sprintf("AssetModificationGovernance:%v", action.AssetModificationGovernance))
	vals = append(vals, fmt.Sprintf("TokenQty:%v", action.TokenQty))
	vals = append(vals, fmt.Sprintf("AssetPayload:%#x", action.AssetPayload))
	vals = append(vals, fmt.Sprintf("AssetRevision:%v", action.AssetRevision))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetModification Token Dilutions, Call Backs/Revocations, burning etc.
type AssetModification struct {
	AssetType     string      `json:"asset_type,omitempty"`     // Three letter character that specifies the asset type.
	AssetCode     AssetCode   `json:"asset_code,omitempty"`     // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetRevision uint32      `json:"asset_revision,omitempty"` // The current revision figure to ensure the modification provided is based on the latest version.
	Amendments    []Amendment `json:"amendments,omitempty"`     // A collection of modifications to perform on this asset.
	RefTxID       TxId        `json:"ref_tx_id,omitempty"`      // The Bitcoin transaction ID of the associated result action that permitted the modifications. See Governance for more details.
}

// Type returns the type identifer for this message.
func (action AssetModification) Type() string {
	return CodeAssetModification
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *AssetModification) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *AssetModification) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetRevision (uint32)
	{
		if err := write(buf, action.AssetRevision); err != nil {
			return nil, err
		}
	}

	// Amendments ([]Amendment)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Amendments)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Amendments {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// RefTxID (TxId)
	{
		b, err := action.RefTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in AssetModification from the byte slice
func (action *AssetModification) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetRevision (uint32)
	{
		if err := read(buf, &action.AssetRevision); err != nil {
			return 0, err
		}
	}

	// Amendments ([]Amendment)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Amendments = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Amendments = append(action.Amendments, newValue)
		}
	}

	// RefTxID (TxId)
	{
		if err := action.RefTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *AssetModification) Validate() error {

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

	// AssetRevision (uint32)
	{
	}

	// Amendments ([]Amendment)
	{
		if len(m.Amendments) > (2<<0)-1 {
			return fmt.Errorf("list field Amendments has too many items %d/%d", len(m.Amendments), (2<<0)-1)
		}

		for i, value := range m.Amendments {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Amendments[%d] is invalid : %s", i, err)
			}
		}
	}

	// RefTxID (TxId)
	{
		if err := m.RefTxID.Validate(); err != nil {
			return fmt.Errorf("field RefTxID is invalid : %s", err)
		}

	}

	return nil
}

func (action AssetModification) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetRevision:%v", action.AssetRevision))
	vals = append(vals, fmt.Sprintf("Amendments:%#+v", action.Amendments))
	vals = append(vals, fmt.Sprintf("RefTxID:%#+v", action.RefTxID))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractOffer Allows the Issuer to tell the smart contract what they
// want the details (labels, data, T&C's, etc.) of the Contract to be
// on-chain in a public and immutable way. The Contract Offer action
// 'initializes' a generic smart contract that has been spun up by either
// the Smart Contract Operator or the Issuer. This on-chain action allows
// for the positive response from the smart contract with either a Contract
// Formation Action or a Rejection Action.
type ContractOffer struct {
	ContractName string `json:"contract_name,omitempty"` // Can be any unique identifying string, including human readable names for branding/vanity purposes. Contract identifier (instance) is the bitcoin public key hash address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address. Smart contracts can be branded and specialized to suit any terms and conditions.

	BodyOfAgreementType      uint8          `json:"body_of_agreement_type,omitempty"`     // 1 - SHA-256 Hash, 2 - Tokenized Body of Agreement Format
	BodyOfAgreement          []byte         `json:"body_of_agreement,omitempty"`          // SHA-256 hash of the body of the agreement (full contract in pdf format or the like) or the full terms and conditions of an agreement in the Tokenized Body of Agreement format.  This is specific to the smart contract and relevant Assets.  Legal and technical information.
	ContractType             string         `json:"contract_type,omitempty"`              // Describes the purpose of the contract.
	SupportingDocsFileType   uint8          `json:"supporting_docs_file_type,omitempty"`  // 1 - 7z
	SupportingDocs           []byte         `json:"supporting_docs,omitempty"`            //
	GoverningLaw             string         `json:"governing_law,omitempty"`              // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction             string         `json:"jurisdiction,omitempty"`               // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	ContractExpiration       Timestamp      `json:"contract_expiration,omitempty"`        // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate total smart contract running costs for the entire life of the contract. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI              string         `json:"contract_uri,omitempty"`               // Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on-chain information or even a public address (DNS).
	Issuer                   Entity         `json:"issuer,omitempty"`                     // The issuer of this contract.
	IssuerLogoURL            string         `json:"issuer_logo_url,omitempty"`            // The URL of the Issuers logo.
	ContractOperatorIncluded bool           `json:"contract_operator_included,omitempty"` // If true, then the second input is a contract operator. If false, then all additional inputs are just funding and "includes" fields are skipped in serialization.
	ContractOperator         Entity         `json:"contract_operator,omitempty"`          // An additional entity with operator access to the contract.
	ContractAuthFlags        []byte         `json:"contract_auth_flags,omitempty"`        // A set of switches that define the authorization rules for this contract. See the Authorization Flags documentation for more detail. Other terms and conditions that are out of the smart contract's control should be listed in the Contract File.
	ContractFee              uint64         `json:"contract_fee,omitempty"`               // Satoshis required to be paid to the contract for each asset action.
	VotingSystems            []VotingSystem `json:"voting_systems,omitempty"`             // A list of voting systems.
	RestrictedQtyAssets      uint64         `json:"restricted_qty_assets,omitempty"`      // Number of Assets (non-fungible) permitted on this contract. 0 if unlimited which will display an infinity symbol in UI
	IssuerProposal           bool           `json:"issuer_proposal,omitempty"`            // Set to true if an Issuer is permitted to make proposals outside of the smart contract scope.
	HolderProposal           bool           `json:"holder_proposal,omitempty"`            // Set to true if a holder is permitted to make proposals outside of the smart contract scope.
	Oracles                  []Oracle       `json:"oracles,omitempty"`                    // A list of oracles that provide approval for all token transfers for all assets under the contract.
}

// Type returns the type identifer for this message.
func (action ContractOffer) Type() string {
	return CodeContractOffer
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *ContractOffer) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *ContractOffer) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// ContractName (string)
	{
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := write(buf, action.BodyOfAgreementType); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		if err := WriteVarBin(buf, action.BodyOfAgreement, 32); err != nil {
			return nil, err
		}
	}

	// ContractType (string)
	{
		if err := WriteVarChar(buf, action.ContractType, 8); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs ([]byte)
	{
		if err := WriteVarBin(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	{
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	{
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	{
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// Issuer (Entity)
	{
		b, err := action.Issuer.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// IssuerLogoURL (string)
	{
		if err := WriteVarChar(buf, action.IssuerLogoURL, 8); err != nil {
			return nil, err
		}
	}

	// ContractOperatorIncluded (bool)
	{
		if err := write(buf, action.ContractOperatorIncluded); err != nil {
			return nil, err
		}
	}

	// ContractOperator (Entity)
	if action.ContractOperatorIncluded {
		b, err := action.ContractOperator.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractAuthFlags ([]byte)
	{
		if err := WriteVarBin(buf, action.ContractAuthFlags, 16); err != nil {
			return nil, err
		}
	}

	// ContractFee (uint64)
	{
		if err := write(buf, action.ContractFee); err != nil {
			return nil, err
		}
	}

	// VotingSystems ([]VotingSystem)
	{
		if err := WriteVariableSize(buf, uint64(len(action.VotingSystems)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.VotingSystems {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// RestrictedQtyAssets (uint64)
	{
		if err := write(buf, action.RestrictedQtyAssets); err != nil {
			return nil, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := write(buf, action.IssuerProposal); err != nil {
			return nil, err
		}
	}

	// HolderProposal (bool)
	{
		if err := write(buf, action.HolderProposal); err != nil {
			return nil, err
		}
	}

	// Oracles ([]Oracle)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Oracles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Oracles {
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

// write populates the fields in ContractOffer from the byte slice
func (action *ContractOffer) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// ContractName (string)
	{
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := read(buf, &action.BodyOfAgreementType); err != nil {
			return 0, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		var err error
		action.BodyOfAgreement, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ContractType (string)
	{
		var err error
		action.ContractType, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs ([]byte)
	{
		var err error
		action.SupportingDocs, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	{
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	{
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	{
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Issuer (Entity)
	{
		if err := action.Issuer.Write(buf); err != nil {
			return 0, err
		}
	}

	// IssuerLogoURL (string)
	{
		var err error
		action.IssuerLogoURL, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractOperatorIncluded (bool)
	{
		if err := read(buf, &action.ContractOperatorIncluded); err != nil {
			return 0, err
		}
	}

	// ContractOperator (Entity)
	if action.ContractOperatorIncluded {
		if err := action.ContractOperator.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractAuthFlags ([]byte)
	{
		var err error
		action.ContractAuthFlags, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// ContractFee (uint64)
	{
		if err := read(buf, &action.ContractFee); err != nil {
			return 0, err
		}
	}

	// VotingSystems ([]VotingSystem)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.VotingSystems = make([]VotingSystem, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue VotingSystem
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.VotingSystems = append(action.VotingSystems, newValue)
		}
	}

	// RestrictedQtyAssets (uint64)
	{
		if err := read(buf, &action.RestrictedQtyAssets); err != nil {
			return 0, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := read(buf, &action.IssuerProposal); err != nil {
			return 0, err
		}
	}

	// HolderProposal (bool)
	{
		if err := read(buf, &action.HolderProposal); err != nil {
			return 0, err
		}
	}

	// Oracles ([]Oracle)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Oracles = make([]Oracle, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Oracle
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Oracles = append(action.Oracles, newValue)
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *ContractOffer) Validate() error {

	// ContractName (string)
	{
		if len(m.ContractName) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractName too long %d/%d", len(m.ContractName), (2<<8)-1)
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if m.BodyOfAgreementType != 1 && m.BodyOfAgreementType != 2 {
			return fmt.Errorf("field BodyOfAgreementType value is invalid : %d", m.BodyOfAgreementType)
		}

	}

	// BodyOfAgreement ([]byte)
	{
		if len(m.BodyOfAgreement) > (2<<32)-1 {
			return fmt.Errorf("varbin field BodyOfAgreement too long %d/%d", len(m.BodyOfAgreement), (2<<32)-1)
		}
	}

	// ContractType (string)
	{
		if len(m.ContractType) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractType too long %d/%d", len(m.ContractType), (2<<8)-1)
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if m.SupportingDocsFileType != 1 {
			return fmt.Errorf("field SupportingDocsFileType value is invalid : %d", m.SupportingDocsFileType)
		}

	}

	// SupportingDocs ([]byte)
	{
		if len(m.SupportingDocs) > (2<<32)-1 {
			return fmt.Errorf("varbin field SupportingDocs too long %d/%d", len(m.SupportingDocs), (2<<32)-1)
		}
	}

	// GoverningLaw (string)
	{
		if len(m.GoverningLaw) > 5 {
			return fmt.Errorf("fixedchar field GoverningLaw too long %d/%d", len(m.GoverningLaw), 5)
		}
	}

	// Jurisdiction (string)
	{
		if len(m.Jurisdiction) > 5 {
			return fmt.Errorf("fixedchar field Jurisdiction too long %d/%d", len(m.Jurisdiction), 5)
		}
	}

	// ContractExpiration (Timestamp)
	{
		if err := m.ContractExpiration.Validate(); err != nil {
			return fmt.Errorf("field ContractExpiration is invalid : %s", err)
		}

	}

	// ContractURI (string)
	{
		if len(m.ContractURI) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractURI too long %d/%d", len(m.ContractURI), (2<<8)-1)
		}
	}

	// Issuer (Entity)
	{
		if err := m.Issuer.Validate(); err != nil {
			return fmt.Errorf("field Issuer is invalid : %s", err)
		}

	}

	// IssuerLogoURL (string)
	{
		if len(m.IssuerLogoURL) > (2<<8)-1 {
			return fmt.Errorf("varchar field IssuerLogoURL too long %d/%d", len(m.IssuerLogoURL), (2<<8)-1)
		}
	}

	// ContractOperatorIncluded (bool)
	{
	}

	// ContractOperator (Entity)
	if m.ContractOperatorIncluded {
		if err := m.ContractOperator.Validate(); err != nil {
			return fmt.Errorf("field ContractOperator is invalid : %s", err)
		}

	}

	// ContractAuthFlags ([]byte)
	{
		if len(m.ContractAuthFlags) > (2<<16)-1 {
			return fmt.Errorf("varbin field ContractAuthFlags too long %d/%d", len(m.ContractAuthFlags), (2<<16)-1)
		}
	}

	// ContractFee (uint64)
	{
	}

	// VotingSystems ([]VotingSystem)
	{
		if len(m.VotingSystems) > (2<<0)-1 {
			return fmt.Errorf("list field VotingSystems has too many items %d/%d", len(m.VotingSystems), (2<<0)-1)
		}

		for i, value := range m.VotingSystems {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field VotingSystems[%d] is invalid : %s", i, err)
			}
		}
	}

	// RestrictedQtyAssets (uint64)
	{
	}

	// IssuerProposal (bool)
	{
	}

	// HolderProposal (bool)
	{
	}

	// Oracles ([]Oracle)
	{
		if len(m.Oracles) > (2<<0)-1 {
			return fmt.Errorf("list field Oracles has too many items %d/%d", len(m.Oracles), (2<<0)-1)
		}

		for i, value := range m.Oracles {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Oracles[%d] is invalid : %s", i, err)
			}
		}
	}

	return nil
}

func (action ContractOffer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("BodyOfAgreementType:%v", action.BodyOfAgreementType))
	vals = append(vals, fmt.Sprintf("BodyOfAgreement:%#x", action.BodyOfAgreement))
	vals = append(vals, fmt.Sprintf("ContractType:%#+v", action.ContractType))
	vals = append(vals, fmt.Sprintf("SupportingDocsFileType:%v", action.SupportingDocsFileType))
	vals = append(vals, fmt.Sprintf("SupportingDocs:%#x", action.SupportingDocs))
	vals = append(vals, fmt.Sprintf("GoverningLaw:%#+v", action.GoverningLaw))
	vals = append(vals, fmt.Sprintf("Jurisdiction:%#+v", action.Jurisdiction))
	vals = append(vals, fmt.Sprintf("ContractExpiration:%#+v", action.ContractExpiration))
	vals = append(vals, fmt.Sprintf("ContractURI:%#+v", action.ContractURI))
	vals = append(vals, fmt.Sprintf("Issuer:%#+v", action.Issuer))
	vals = append(vals, fmt.Sprintf("IssuerLogoURL:%#+v", action.IssuerLogoURL))
	vals = append(vals, fmt.Sprintf("ContractOperatorIncluded:%#+v", action.ContractOperatorIncluded))
	vals = append(vals, fmt.Sprintf("ContractOperator:%#+v", action.ContractOperator))
	vals = append(vals, fmt.Sprintf("ContractAuthFlags:%#x", action.ContractAuthFlags))
	vals = append(vals, fmt.Sprintf("ContractFee:%v", action.ContractFee))
	vals = append(vals, fmt.Sprintf("VotingSystems:%#+v", action.VotingSystems))
	vals = append(vals, fmt.Sprintf("RestrictedQtyAssets:%v", action.RestrictedQtyAssets))
	vals = append(vals, fmt.Sprintf("IssuerProposal:%#+v", action.IssuerProposal))
	vals = append(vals, fmt.Sprintf("HolderProposal:%#+v", action.HolderProposal))
	vals = append(vals, fmt.Sprintf("Oracles:%#+v", action.Oracles))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractFormation This txn is created by the Contract (smart
// contract/off-chain agent/token contract) upon receipt of a valid
// Contract Offer Action from the issuer. The Smart Contract will execute
// on a server controlled by the Issuer. or a Smart Contract Operator on
// their behalf.
type ContractFormation struct {
	ContractName string `json:"contract_name,omitempty"` // Can be any unique identifying string, including human readable names for branding/vanity purposes. Contract identifier (instance) is the bitcoin public key hash address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address. Smart contracts can be branded and specialized to suit any terms and conditions.

	BodyOfAgreementType      uint8          `json:"body_of_agreement_type,omitempty"`     // 1 - SHA-256 Hash, 2 - Tokenized Body of Agreement Format
	BodyOfAgreement          []byte         `json:"body_of_agreement,omitempty"`          // SHA-256 hash of the body of the agreement (full contract in pdf format or the like) or the full terms and conditions of an agreement in the Tokenized Body of Agreement format.  This is specific to the smart contract and relevant Assets.  Legal and technical information.
	ContractType             string         `json:"contract_type,omitempty"`              // Describes the purpose of the contract.
	SupportingDocsFileType   uint8          `json:"supporting_docs_file_type,omitempty"`  // 1 - 7z
	SupportingDocs           []byte         `json:"supporting_docs,omitempty"`            //
	GoverningLaw             string         `json:"governing_law,omitempty"`              // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction             string         `json:"jurisdiction,omitempty"`               // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	ContractExpiration       Timestamp      `json:"contract_expiration,omitempty"`        // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate smart contract running costs. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI              string         `json:"contract_uri,omitempty"`               // Length 0-255 bytes.  0 is valid. Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on chain information or even a public address (DNS).
	Issuer                   Entity         `json:"issuer,omitempty"`                     // The issuer of this contract.
	IssuerLogoURL            string         `json:"issuer_logo_url,omitempty"`            // The URL of the Issuers logo.
	ContractOperatorIncluded bool           `json:"contract_operator_included,omitempty"` // If true, then the second input is a contract operator. If false, then all additional inputs are just funding and "includes" fields are skipped in serialization.
	ContractOperator         Entity         `json:"contract_operator,omitempty"`          // An additional entity with operator access to the contract.
	ContractAuthFlags        []byte         `json:"contract_auth_flags,omitempty"`        // A set of switches that define the authorization rules for this contract. See the Authorization Flags documentation for more detail. Other terms and conditions that are out of the smart contract's control should be listed in the Contract File.
	ContractFee              uint64         `json:"contract_fee,omitempty"`               // Satoshis required to be paid to the contract for each asset action.
	VotingSystems            []VotingSystem `json:"voting_systems,omitempty"`             // A list voting systems.
	RestrictedQtyAssets      uint64         `json:"restricted_qty_assets,omitempty"`      // Number of Assets (non-fungible) permitted on this contract. 0 if unlimited which will display an infinity symbol in UI
	IssuerProposal           bool           `json:"issuer_proposal,omitempty"`            // Set to true if an Issuer is permitted to make proposals outside of the smart contract scope.
	HolderProposal           bool           `json:"holder_proposal,omitempty"`            // Set to true if a holder is permitted to make proposals outside of the smart contract scope.
	Oracles                  []Oracle       `json:"oracles,omitempty"`                    // A list of oracles that provide approval for all token transfers for all assets under the contract.
	ContractRevision         uint32         `json:"contract_revision,omitempty"`          // A counter for the number of times this contract has been revised using an amendment action.
	Timestamp                Timestamp      `json:"timestamp,omitempty"`                  // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action ContractFormation) Type() string {
	return CodeContractFormation
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *ContractFormation) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *ContractFormation) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// ContractName (string)
	{
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := write(buf, action.BodyOfAgreementType); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		if err := WriteVarBin(buf, action.BodyOfAgreement, 32); err != nil {
			return nil, err
		}
	}

	// ContractType (string)
	{
		if err := WriteVarChar(buf, action.ContractType, 8); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs ([]byte)
	{
		if err := WriteVarBin(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	{
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	{
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	{
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// Issuer (Entity)
	{
		b, err := action.Issuer.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// IssuerLogoURL (string)
	{
		if err := WriteVarChar(buf, action.IssuerLogoURL, 8); err != nil {
			return nil, err
		}
	}

	// ContractOperatorIncluded (bool)
	{
		if err := write(buf, action.ContractOperatorIncluded); err != nil {
			return nil, err
		}
	}

	// ContractOperator (Entity)
	if action.ContractOperatorIncluded {
		b, err := action.ContractOperator.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractAuthFlags ([]byte)
	{
		if err := WriteVarBin(buf, action.ContractAuthFlags, 16); err != nil {
			return nil, err
		}
	}

	// ContractFee (uint64)
	{
		if err := write(buf, action.ContractFee); err != nil {
			return nil, err
		}
	}

	// VotingSystems ([]VotingSystem)
	{
		if err := WriteVariableSize(buf, uint64(len(action.VotingSystems)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.VotingSystems {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// RestrictedQtyAssets (uint64)
	{
		if err := write(buf, action.RestrictedQtyAssets); err != nil {
			return nil, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := write(buf, action.IssuerProposal); err != nil {
			return nil, err
		}
	}

	// HolderProposal (bool)
	{
		if err := write(buf, action.HolderProposal); err != nil {
			return nil, err
		}
	}

	// Oracles ([]Oracle)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Oracles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Oracles {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// ContractRevision (uint32)
	{
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in ContractFormation from the byte slice
func (action *ContractFormation) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// ContractName (string)
	{
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := read(buf, &action.BodyOfAgreementType); err != nil {
			return 0, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		var err error
		action.BodyOfAgreement, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ContractType (string)
	{
		var err error
		action.ContractType, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs ([]byte)
	{
		var err error
		action.SupportingDocs, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	{
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	{
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	{
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Issuer (Entity)
	{
		if err := action.Issuer.Write(buf); err != nil {
			return 0, err
		}
	}

	// IssuerLogoURL (string)
	{
		var err error
		action.IssuerLogoURL, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractOperatorIncluded (bool)
	{
		if err := read(buf, &action.ContractOperatorIncluded); err != nil {
			return 0, err
		}
	}

	// ContractOperator (Entity)
	if action.ContractOperatorIncluded {
		if err := action.ContractOperator.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractAuthFlags ([]byte)
	{
		var err error
		action.ContractAuthFlags, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// ContractFee (uint64)
	{
		if err := read(buf, &action.ContractFee); err != nil {
			return 0, err
		}
	}

	// VotingSystems ([]VotingSystem)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.VotingSystems = make([]VotingSystem, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue VotingSystem
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.VotingSystems = append(action.VotingSystems, newValue)
		}
	}

	// RestrictedQtyAssets (uint64)
	{
		if err := read(buf, &action.RestrictedQtyAssets); err != nil {
			return 0, err
		}
	}

	// IssuerProposal (bool)
	{
		if err := read(buf, &action.IssuerProposal); err != nil {
			return 0, err
		}
	}

	// HolderProposal (bool)
	{
		if err := read(buf, &action.HolderProposal); err != nil {
			return 0, err
		}
	}

	// Oracles ([]Oracle)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Oracles = make([]Oracle, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Oracle
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Oracles = append(action.Oracles, newValue)
		}
	}

	// ContractRevision (uint32)
	{
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *ContractFormation) Validate() error {

	// ContractName (string)
	{
		if len(m.ContractName) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractName too long %d/%d", len(m.ContractName), (2<<8)-1)
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if m.BodyOfAgreementType != 1 && m.BodyOfAgreementType != 2 {
			return fmt.Errorf("field BodyOfAgreementType value is invalid : %d", m.BodyOfAgreementType)
		}

	}

	// BodyOfAgreement ([]byte)
	{
		if len(m.BodyOfAgreement) > (2<<32)-1 {
			return fmt.Errorf("varbin field BodyOfAgreement too long %d/%d", len(m.BodyOfAgreement), (2<<32)-1)
		}
	}

	// ContractType (string)
	{
		if len(m.ContractType) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractType too long %d/%d", len(m.ContractType), (2<<8)-1)
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if m.SupportingDocsFileType != 1 {
			return fmt.Errorf("field SupportingDocsFileType value is invalid : %d", m.SupportingDocsFileType)
		}

	}

	// SupportingDocs ([]byte)
	{
		if len(m.SupportingDocs) > (2<<32)-1 {
			return fmt.Errorf("varbin field SupportingDocs too long %d/%d", len(m.SupportingDocs), (2<<32)-1)
		}
	}

	// GoverningLaw (string)
	{
		if len(m.GoverningLaw) > 5 {
			return fmt.Errorf("fixedchar field GoverningLaw too long %d/%d", len(m.GoverningLaw), 5)
		}
	}

	// Jurisdiction (string)
	{
		if len(m.Jurisdiction) > 5 {
			return fmt.Errorf("fixedchar field Jurisdiction too long %d/%d", len(m.Jurisdiction), 5)
		}
	}

	// ContractExpiration (Timestamp)
	{
		if err := m.ContractExpiration.Validate(); err != nil {
			return fmt.Errorf("field ContractExpiration is invalid : %s", err)
		}

	}

	// ContractURI (string)
	{
		if len(m.ContractURI) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractURI too long %d/%d", len(m.ContractURI), (2<<8)-1)
		}
	}

	// Issuer (Entity)
	{
		if err := m.Issuer.Validate(); err != nil {
			return fmt.Errorf("field Issuer is invalid : %s", err)
		}

	}

	// IssuerLogoURL (string)
	{
		if len(m.IssuerLogoURL) > (2<<8)-1 {
			return fmt.Errorf("varchar field IssuerLogoURL too long %d/%d", len(m.IssuerLogoURL), (2<<8)-1)
		}
	}

	// ContractOperatorIncluded (bool)
	{
	}

	// ContractOperator (Entity)
	if m.ContractOperatorIncluded {
		if err := m.ContractOperator.Validate(); err != nil {
			return fmt.Errorf("field ContractOperator is invalid : %s", err)
		}

	}

	// ContractAuthFlags ([]byte)
	{
		if len(m.ContractAuthFlags) > (2<<16)-1 {
			return fmt.Errorf("varbin field ContractAuthFlags too long %d/%d", len(m.ContractAuthFlags), (2<<16)-1)
		}
	}

	// ContractFee (uint64)
	{
	}

	// VotingSystems ([]VotingSystem)
	{
		if len(m.VotingSystems) > (2<<0)-1 {
			return fmt.Errorf("list field VotingSystems has too many items %d/%d", len(m.VotingSystems), (2<<0)-1)
		}

		for i, value := range m.VotingSystems {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field VotingSystems[%d] is invalid : %s", i, err)
			}
		}
	}

	// RestrictedQtyAssets (uint64)
	{
	}

	// IssuerProposal (bool)
	{
	}

	// HolderProposal (bool)
	{
	}

	// Oracles ([]Oracle)
	{
		if len(m.Oracles) > (2<<0)-1 {
			return fmt.Errorf("list field Oracles has too many items %d/%d", len(m.Oracles), (2<<0)-1)
		}

		for i, value := range m.Oracles {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Oracles[%d] is invalid : %s", i, err)
			}
		}
	}

	// ContractRevision (uint32)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action ContractFormation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("BodyOfAgreementType:%v", action.BodyOfAgreementType))
	vals = append(vals, fmt.Sprintf("BodyOfAgreement:%#x", action.BodyOfAgreement))
	vals = append(vals, fmt.Sprintf("ContractType:%#+v", action.ContractType))
	vals = append(vals, fmt.Sprintf("SupportingDocsFileType:%v", action.SupportingDocsFileType))
	vals = append(vals, fmt.Sprintf("SupportingDocs:%#x", action.SupportingDocs))
	vals = append(vals, fmt.Sprintf("GoverningLaw:%#+v", action.GoverningLaw))
	vals = append(vals, fmt.Sprintf("Jurisdiction:%#+v", action.Jurisdiction))
	vals = append(vals, fmt.Sprintf("ContractExpiration:%#+v", action.ContractExpiration))
	vals = append(vals, fmt.Sprintf("ContractURI:%#+v", action.ContractURI))
	vals = append(vals, fmt.Sprintf("Issuer:%#+v", action.Issuer))
	vals = append(vals, fmt.Sprintf("IssuerLogoURL:%#+v", action.IssuerLogoURL))
	vals = append(vals, fmt.Sprintf("ContractOperatorIncluded:%#+v", action.ContractOperatorIncluded))
	vals = append(vals, fmt.Sprintf("ContractOperator:%#+v", action.ContractOperator))
	vals = append(vals, fmt.Sprintf("ContractAuthFlags:%#x", action.ContractAuthFlags))
	vals = append(vals, fmt.Sprintf("ContractFee:%v", action.ContractFee))
	vals = append(vals, fmt.Sprintf("VotingSystems:%#+v", action.VotingSystems))
	vals = append(vals, fmt.Sprintf("RestrictedQtyAssets:%v", action.RestrictedQtyAssets))
	vals = append(vals, fmt.Sprintf("IssuerProposal:%#+v", action.IssuerProposal))
	vals = append(vals, fmt.Sprintf("HolderProposal:%#+v", action.HolderProposal))
	vals = append(vals, fmt.Sprintf("Oracles:%#+v", action.Oracles))
	vals = append(vals, fmt.Sprintf("ContractRevision:%v", action.ContractRevision))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractAmendment The issuer can initiate an amendment to the contract
// establishment metadata. The ability to make an amendment to the contract
// is restricted by the Authorization Flag set on the current revision of
// Contract Formation action.
type ContractAmendment struct {
	ChangeIssuerAddress   bool        `json:"change_issuer_address,omitempty"`   // Used to change the issuer address.  The new issuer address must be in the input[1] position.
	ChangeOperatorAddress bool        `json:"change_operator_address,omitempty"` // Used to change the smart contract operator address.  The new operator address must be in the input[1] position, unless issuer is being changed too, then it is in input[2].
	ContractRevision      uint32      `json:"contract_revision,omitempty"`       // Counter 0 to (2^32)-1
	Amendments            []Amendment `json:"amendments,omitempty"`              // A collection of modifications to perform on this contract.
	RefTxID               TxId        `json:"ref_tx_id,omitempty"`               // The Bitcoin transaction ID of the associated result action that permitted the modifications. See Governance for more details.
}

// Type returns the type identifer for this message.
func (action ContractAmendment) Type() string {
	return CodeContractAmendment
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *ContractAmendment) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *ContractAmendment) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// ChangeIssuerAddress (bool)
	{
		if err := write(buf, action.ChangeIssuerAddress); err != nil {
			return nil, err
		}
	}

	// ChangeOperatorAddress (bool)
	{
		if err := write(buf, action.ChangeOperatorAddress); err != nil {
			return nil, err
		}
	}

	// ContractRevision (uint32)
	{
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// Amendments ([]Amendment)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Amendments)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Amendments {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// RefTxID (TxId)
	{
		b, err := action.RefTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in ContractAmendment from the byte slice
func (action *ContractAmendment) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// ChangeIssuerAddress (bool)
	{
		if err := read(buf, &action.ChangeIssuerAddress); err != nil {
			return 0, err
		}
	}

	// ChangeOperatorAddress (bool)
	{
		if err := read(buf, &action.ChangeOperatorAddress); err != nil {
			return 0, err
		}
	}

	// ContractRevision (uint32)
	{
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// Amendments ([]Amendment)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Amendments = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Amendments = append(action.Amendments, newValue)
		}
	}

	// RefTxID (TxId)
	{
		if err := action.RefTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *ContractAmendment) Validate() error {

	// ChangeIssuerAddress (bool)
	{
	}

	// ChangeOperatorAddress (bool)
	{
	}

	// ContractRevision (uint32)
	{
	}

	// Amendments ([]Amendment)
	{
		if len(m.Amendments) > (2<<0)-1 {
			return fmt.Errorf("list field Amendments has too many items %d/%d", len(m.Amendments), (2<<0)-1)
		}

		for i, value := range m.Amendments {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Amendments[%d] is invalid : %s", i, err)
			}
		}
	}

	// RefTxID (TxId)
	{
		if err := m.RefTxID.Validate(); err != nil {
			return fmt.Errorf("field RefTxID is invalid : %s", err)
		}

	}

	return nil
}

func (action ContractAmendment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ChangeIssuerAddress:%#+v", action.ChangeIssuerAddress))
	vals = append(vals, fmt.Sprintf("ChangeOperatorAddress:%#+v", action.ChangeOperatorAddress))
	vals = append(vals, fmt.Sprintf("ContractRevision:%v", action.ContractRevision))
	vals = append(vals, fmt.Sprintf("Amendments:%#+v", action.Amendments))
	vals = append(vals, fmt.Sprintf("RefTxID:%#+v", action.RefTxID))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// StaticContractFormation Static Contract Formation Action
type StaticContractFormation struct {
	ContractName string `json:"contract_name,omitempty"` // Can be any unique identifying string, including human readable names for branding/vanity purposes. Contract identifier (instance) is the bitcoin public address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address. Smart contracts can be branded and specialized to suit any terms and conditions.

	ContractCode           ContractCode `json:"contract_code,omitempty"`             // 32 randomly generated bytes. Each Contract Code should be unique. The Contract ID will be human facing and will be the Contract Code, with a checksum, encoded in base58 and prefixed by 'CON'. Contract ID = CON + base58(ContractCode + checksum).  Eg. Contract ID = 'CON18RDoKK7Ed5zid2FkKVy7q3rULr4tgfjr4'
	BodyOfAgreementType    uint8        `json:"body_of_agreement_type,omitempty"`    // 1 - SHA-256 Hash, 2 - Tokenized Body of Agreement Format
	BodyOfAgreement        []byte       `json:"body_of_agreement,omitempty"`         // SHA-256 hash of the body of the agreement (full contract in pdf format or the like) or the full terms and conditions of an agreement in the Tokenized Body of Agreement format.  This is specific to the smart contract and relevant Assets.  Legal and technical information.
	ContractType           string       `json:"contract_type,omitempty"`             // Describes the purpose of the contract.
	SupportingDocsFileType uint8        `json:"supporting_docs_file_type,omitempty"` // 1 - 7z
	SupportingDocs         string       `json:"supporting_docs,omitempty"`           //
	ContractRevision       uint32       `json:"contract_revision,omitempty"`         // Counter 0 to (2^32)-1
	GoverningLaw           string       `json:"governing_law,omitempty"`             // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction           string       `json:"jurisdiction,omitempty"`              // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	EffectiveDate          Timestamp    `json:"effective_date,omitempty"`            // Start date of the contract.
	ContractExpiration     Timestamp    `json:"contract_expiration,omitempty"`       // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate smart contract running costs. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI            string       `json:"contract_uri,omitempty"`              // Length 0-255 bytes. Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on chain information or even a public address (DNS).
	PrevRevTxID            TxId         `json:"prev_rev_tx_id,omitempty"`            // The Tx-ID of the previous contract revision.
	Entities               []Entity     `json:"entities,omitempty"`                  // A list of legal entities associated with this contract.
}

// Type returns the type identifer for this message.
func (action StaticContractFormation) Type() string {
	return CodeStaticContractFormation
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *StaticContractFormation) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *StaticContractFormation) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// ContractName (string)
	{
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// ContractCode (ContractCode)
	{
		b, err := action.ContractCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := write(buf, action.BodyOfAgreementType); err != nil {
			return nil, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		if err := WriteVarBin(buf, action.BodyOfAgreement, 32); err != nil {
			return nil, err
		}
	}

	// ContractType (string)
	{
		if err := WriteVarChar(buf, action.ContractType, 8); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs (string)
	{
		if err := WriteVarChar(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// ContractRevision (uint32)
	{
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	{
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	{
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// EffectiveDate (Timestamp)
	{
		b, err := action.EffectiveDate.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	{
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// PrevRevTxID (TxId)
	{
		b, err := action.PrevRevTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Entities ([]Entity)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Entities)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Entities {
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

// write populates the fields in StaticContractFormation from the byte slice
func (action *StaticContractFormation) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// ContractName (string)
	{
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractCode (ContractCode)
	{
		if err := action.ContractCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// BodyOfAgreementType (uint8)
	{
		if err := read(buf, &action.BodyOfAgreementType); err != nil {
			return 0, err
		}
	}

	// BodyOfAgreement ([]byte)
	{
		var err error
		action.BodyOfAgreement, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ContractType (string)
	{
		var err error
		action.ContractType, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs (string)
	{
		var err error
		action.SupportingDocs, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ContractRevision (uint32)
	{
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	{
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	{
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// EffectiveDate (Timestamp)
	{
		if err := action.EffectiveDate.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	{
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	{
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// PrevRevTxID (TxId)
	{
		if err := action.PrevRevTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Entities ([]Entity)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Entities = make([]Entity, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Entity
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Entities = append(action.Entities, newValue)
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *StaticContractFormation) Validate() error {

	// ContractName (string)
	{
		if len(m.ContractName) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractName too long %d/%d", len(m.ContractName), (2<<8)-1)
		}
	}

	// ContractCode (ContractCode)
	{
		if err := m.ContractCode.Validate(); err != nil {
			return fmt.Errorf("field ContractCode is invalid : %s", err)
		}

	}

	// BodyOfAgreementType (uint8)
	{
		if m.BodyOfAgreementType != 1 && m.BodyOfAgreementType != 2 {
			return fmt.Errorf("field BodyOfAgreementType value is invalid : %d", m.BodyOfAgreementType)
		}

	}

	// BodyOfAgreement ([]byte)
	{
		if len(m.BodyOfAgreement) > (2<<32)-1 {
			return fmt.Errorf("varbin field BodyOfAgreement too long %d/%d", len(m.BodyOfAgreement), (2<<32)-1)
		}
	}

	// ContractType (string)
	{
		if len(m.ContractType) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractType too long %d/%d", len(m.ContractType), (2<<8)-1)
		}
	}

	// SupportingDocsFileType (uint8)
	{
		if m.SupportingDocsFileType != 1 {
			return fmt.Errorf("field SupportingDocsFileType value is invalid : %d", m.SupportingDocsFileType)
		}

	}

	// SupportingDocs (string)
	{
		if len(m.SupportingDocs) > (2<<32)-1 {
			return fmt.Errorf("varchar field SupportingDocs too long %d/%d", len(m.SupportingDocs), (2<<32)-1)
		}
	}

	// ContractRevision (uint32)
	{
	}

	// GoverningLaw (string)
	{
		if len(m.GoverningLaw) > 5 {
			return fmt.Errorf("fixedchar field GoverningLaw too long %d/%d", len(m.GoverningLaw), 5)
		}
	}

	// Jurisdiction (string)
	{
		if len(m.Jurisdiction) > 5 {
			return fmt.Errorf("fixedchar field Jurisdiction too long %d/%d", len(m.Jurisdiction), 5)
		}
	}

	// EffectiveDate (Timestamp)
	{
		if err := m.EffectiveDate.Validate(); err != nil {
			return fmt.Errorf("field EffectiveDate is invalid : %s", err)
		}

	}

	// ContractExpiration (Timestamp)
	{
		if err := m.ContractExpiration.Validate(); err != nil {
			return fmt.Errorf("field ContractExpiration is invalid : %s", err)
		}

	}

	// ContractURI (string)
	{
		if len(m.ContractURI) > (2<<8)-1 {
			return fmt.Errorf("varchar field ContractURI too long %d/%d", len(m.ContractURI), (2<<8)-1)
		}
	}

	// PrevRevTxID (TxId)
	{
		if err := m.PrevRevTxID.Validate(); err != nil {
			return fmt.Errorf("field PrevRevTxID is invalid : %s", err)
		}

	}

	// Entities ([]Entity)
	{
		if len(m.Entities) > (2<<0)-1 {
			return fmt.Errorf("list field Entities has too many items %d/%d", len(m.Entities), (2<<0)-1)
		}

		for i, value := range m.Entities {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Entities[%d] is invalid : %s", i, err)
			}
		}
	}

	return nil
}

func (action StaticContractFormation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("ContractCode:%#+v", action.ContractCode))
	vals = append(vals, fmt.Sprintf("BodyOfAgreementType:%v", action.BodyOfAgreementType))
	vals = append(vals, fmt.Sprintf("BodyOfAgreement:%#x", action.BodyOfAgreement))
	vals = append(vals, fmt.Sprintf("ContractType:%#+v", action.ContractType))
	vals = append(vals, fmt.Sprintf("SupportingDocsFileType:%v", action.SupportingDocsFileType))
	vals = append(vals, fmt.Sprintf("SupportingDocs:%#+v", action.SupportingDocs))
	vals = append(vals, fmt.Sprintf("ContractRevision:%v", action.ContractRevision))
	vals = append(vals, fmt.Sprintf("GoverningLaw:%#+v", action.GoverningLaw))
	vals = append(vals, fmt.Sprintf("Jurisdiction:%#+v", action.Jurisdiction))
	vals = append(vals, fmt.Sprintf("EffectiveDate:%#+v", action.EffectiveDate))
	vals = append(vals, fmt.Sprintf("ContractExpiration:%#+v", action.ContractExpiration))
	vals = append(vals, fmt.Sprintf("ContractURI:%#+v", action.ContractURI))
	vals = append(vals, fmt.Sprintf("PrevRevTxID:%#+v", action.PrevRevTxID))
	vals = append(vals, fmt.Sprintf("Entities:%#+v", action.Entities))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Order Issuer to signal to the smart contract that the tokens that a
// particular public address(es) owns are to be confiscated, frozen, thawed
// or reconciled.
type Order struct {
	ComplianceAction       byte            `json:"compliance_action,omitempty"`        // Freeze (F), Thaw (T), Confiscate (C), Reconcile (R)
	AssetType              string          `json:"asset_type,omitempty"`               // Three letter character that specifies the asset type.
	AssetCode              AssetCode       `json:"asset_code,omitempty"`               // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	TargetAddresses        []TargetAddress `json:"target_addresses,omitempty"`         // The holders and quantities that are effected by the order. For a contract or asset wide freeze only the contract address is specified. Zero quantities are invalid unless it is for the contract address in an asset wide or contract wide freeze. In a thaw order this field is not serialized, because the entire freeze from the FreezeTxId freeze action will be thawed.
	FreezeTxId             TxId            `json:"freeze_tx_id,omitempty"`             // The tx id of the freeze action that is being thawed. Only serialized for thaw orders.
	FreezePeriod           Timestamp       `json:"freeze_period,omitempty"`            // Used for a 'time out'.  Tokens are automatically unfrozen after the expiration timestamp without requiring a Thaw Action. Null value for Thaw, Confiscation and Reconciallitaion orders.
	DepositAddress         PublicKeyHash   `json:"deposit_address,omitempty"`          // The public address for confiscated tokens to be deposited in.  Null for Freeze, Thaw, actions.
	AuthorityIncluded      bool            `json:"authority_included,omitempty"`       // Specifies if an authority signature is included. For Reconcialitaion actions all authority signature related fields are skipped during serialization.
	AuthorityName          string          `json:"authority_name,omitempty"`           // Length 0-255 bytes. Enforcement Authority Name (eg. Issuer, Queensland Police Service, Tokenized, etc.)
	AuthorityPublicKey     []byte          `json:"authority_public_key,omitempty"`     // Length 0-255 bytes. Public Key associated with the Enforcement Authority
	SignatureAlgorithm     uint8           `json:"signature_algorithm,omitempty"`      // Algorithm used for order signature. Only valid value is currently 1 = ECDSA+secp256k1
	OrderSignature         []byte          `json:"order_signature,omitempty"`          // Length 0-255 bytes. Signature for a message that lists out the target addresses and deposit address. Signature of (Contract PKH, Compliance Action, Authority Name, Asset Code, Supporting Evidence Hash, FreezePeriod, TargetAddresses, and DepositAddress)
	SupportingEvidenceHash [32]byte        `json:"supporting_evidence_hash,omitempty"` // SHA-256: warrant, court order, etc.
	RefTxs                 []byte          `json:"ref_txs,omitempty"`                  // The request/response actions that were dropped.  The entire txn for both actions is included as evidence that the actions were accepted into the mempool at one point and that the senders (token/Bitcoin) signed their intent to transfer.  The management of this record keeping is off-chain and managed by the issuer or operator to preserve the integrity of the state of the tokens. Only applicable for reconcilliation actions.  No subfield when F, T, R is selected as the Compliance Action subfield.
	BitcoinDispersions     []QuantityIndex `json:"bitcoin_dispersions,omitempty"`      // Index of address in TargetAddresses and amount of bitcoin (in satoshis) they are receiving in exchange for their tokens.
	Message                string          `json:"message,omitempty"`                  // A message to include with the enforcement order.
}

// Type returns the type identifer for this message.
func (action Order) Type() string {
	return CodeOrder
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Order) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Order) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// ComplianceAction (byte)
	{
		if err := write(buf, action.ComplianceAction); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// TargetAddresses ([]TargetAddress)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		if err := WriteVariableSize(buf, uint64(len(action.TargetAddresses)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.TargetAddresses {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// FreezeTxId (TxId)
	if action.ComplianceAction == 'T' {
		b, err := action.FreezeTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// FreezePeriod (Timestamp)
	if action.ComplianceAction == 'F' {
		b, err := action.FreezePeriod.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// DepositAddress (PublicKeyHash)
	if action.ComplianceAction == 'C' {
		b, err := action.DepositAddress.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AuthorityIncluded (bool)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'T' || action.ComplianceAction == 'C' {
		if err := write(buf, action.AuthorityIncluded); err != nil {
			return nil, err
		}
	}

	// AuthorityName (string)
	if action.AuthorityIncluded {
		if err := WriteVarChar(buf, action.AuthorityName, 8); err != nil {
			return nil, err
		}
	}

	// AuthorityPublicKey ([]byte)
	if action.AuthorityIncluded {
		if err := WriteVarBin(buf, action.AuthorityPublicKey, 8); err != nil {
			return nil, err
		}
	}

	// SignatureAlgorithm (uint8)
	if action.AuthorityIncluded {
		if err := write(buf, action.SignatureAlgorithm); err != nil {
			return nil, err
		}
	}

	// OrderSignature ([]byte)
	if action.AuthorityIncluded {
		if err := WriteVarBin(buf, action.OrderSignature, 8); err != nil {
			return nil, err
		}
	}

	// SupportingEvidenceHash ([32]byte)
	{
		if err := write(buf, action.SupportingEvidenceHash); err != nil {
			return nil, err
		}
	}

	// RefTxs ([]byte)
	if action.ComplianceAction == 'R' {
		if err := WriteVarBin(buf, action.RefTxs, 32); err != nil {
			return nil, err
		}
	}

	// BitcoinDispersions ([]QuantityIndex)
	if action.ComplianceAction == 'R' {
		if err := WriteVariableSize(buf, uint64(len(action.BitcoinDispersions)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.BitcoinDispersions {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Order from the byte slice
func (action *Order) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// ComplianceAction (byte)
	{
		if err := read(buf, &action.ComplianceAction); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// TargetAddresses ([]TargetAddress)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'C' || action.ComplianceAction == 'R' {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.TargetAddresses = make([]TargetAddress, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue TargetAddress
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.TargetAddresses = append(action.TargetAddresses, newValue)
		}
	}

	// FreezeTxId (TxId)
	if action.ComplianceAction == 'T' {
		if err := action.FreezeTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// FreezePeriod (Timestamp)
	if action.ComplianceAction == 'F' {
		if err := action.FreezePeriod.Write(buf); err != nil {
			return 0, err
		}
	}

	// DepositAddress (PublicKeyHash)
	if action.ComplianceAction == 'C' {
		if err := action.DepositAddress.Write(buf); err != nil {
			return 0, err
		}
	}

	// AuthorityIncluded (bool)
	if action.ComplianceAction == 'F' || action.ComplianceAction == 'T' || action.ComplianceAction == 'C' {
		if err := read(buf, &action.AuthorityIncluded); err != nil {
			return 0, err
		}
	}

	// AuthorityName (string)
	if action.AuthorityIncluded {
		var err error
		action.AuthorityName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// AuthorityPublicKey ([]byte)
	if action.AuthorityIncluded {
		var err error
		action.AuthorityPublicKey, err = ReadVarBin(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SignatureAlgorithm (uint8)
	if action.AuthorityIncluded {
		if err := read(buf, &action.SignatureAlgorithm); err != nil {
			return 0, err
		}
	}

	// OrderSignature ([]byte)
	if action.AuthorityIncluded {
		var err error
		action.OrderSignature, err = ReadVarBin(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SupportingEvidenceHash ([32]byte)
	{
		if err := read(buf, &action.SupportingEvidenceHash); err != nil {
			return 0, err
		}
	}

	// RefTxs ([]byte)
	if action.ComplianceAction == 'R' {
		var err error
		action.RefTxs, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// BitcoinDispersions ([]QuantityIndex)
	if action.ComplianceAction == 'R' {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.BitcoinDispersions = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.BitcoinDispersions = append(action.BitcoinDispersions, newValue)
		}
	}

	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Order) Validate() error {

	// ComplianceAction (byte)
	{
		if m.ComplianceAction != 'F' && m.ComplianceAction != 'T' && m.ComplianceAction != 'C' && m.ComplianceAction != 'R' {
			return fmt.Errorf("field ComplianceAction value is invalid : %d", m.ComplianceAction)
		}
	}

	// AssetType (string)
	if m.ComplianceAction == 'F' || m.ComplianceAction == 'C' || m.ComplianceAction == 'R' {
		if len(m.AssetType) > 3 {
			return fmt.Errorf("fixedchar field AssetType too long %d/%d", len(m.AssetType), 3)
		}
	}

	// AssetCode (AssetCode)
	if m.ComplianceAction == 'F' || m.ComplianceAction == 'C' || m.ComplianceAction == 'R' {
		if err := m.AssetCode.Validate(); err != nil {
			return fmt.Errorf("field AssetCode is invalid : %s", err)
		}

	}

	// TargetAddresses ([]TargetAddress)
	if m.ComplianceAction == 'F' || m.ComplianceAction == 'C' || m.ComplianceAction == 'R' {
		if len(m.TargetAddresses) > (2<<16)-1 {
			return fmt.Errorf("list field TargetAddresses has too many items %d/%d", len(m.TargetAddresses), (2<<16)-1)
		}

		for i, value := range m.TargetAddresses {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field TargetAddresses[%d] is invalid : %s", i, err)
			}
		}
	}

	// FreezeTxId (TxId)
	if m.ComplianceAction == 'T' {
		if err := m.FreezeTxId.Validate(); err != nil {
			return fmt.Errorf("field FreezeTxId is invalid : %s", err)
		}

	}

	// FreezePeriod (Timestamp)
	if m.ComplianceAction == 'F' {
		if err := m.FreezePeriod.Validate(); err != nil {
			return fmt.Errorf("field FreezePeriod is invalid : %s", err)
		}

	}

	// DepositAddress (PublicKeyHash)
	if m.ComplianceAction == 'C' {
		if err := m.DepositAddress.Validate(); err != nil {
			return fmt.Errorf("field DepositAddress is invalid : %s", err)
		}

	}

	// AuthorityIncluded (bool)
	if m.ComplianceAction == 'F' || m.ComplianceAction == 'T' || m.ComplianceAction == 'C' {
	}

	// AuthorityName (string)
	if m.AuthorityIncluded {
		if len(m.AuthorityName) > (2<<8)-1 {
			return fmt.Errorf("varchar field AuthorityName too long %d/%d", len(m.AuthorityName), (2<<8)-1)
		}
	}

	// AuthorityPublicKey ([]byte)
	if m.AuthorityIncluded {
		if len(m.AuthorityPublicKey) > (2<<8)-1 {
			return fmt.Errorf("varbin field AuthorityPublicKey too long %d/%d", len(m.AuthorityPublicKey), (2<<8)-1)
		}
	}

	// SignatureAlgorithm (uint8)
	if m.AuthorityIncluded {
		if m.SignatureAlgorithm != 1 {
			return fmt.Errorf("field SignatureAlgorithm value is invalid : %d", m.SignatureAlgorithm)
		}

	}

	// OrderSignature ([]byte)
	if m.AuthorityIncluded {
		if len(m.OrderSignature) > (2<<8)-1 {
			return fmt.Errorf("varbin field OrderSignature too long %d/%d", len(m.OrderSignature), (2<<8)-1)
		}
	}

	// SupportingEvidenceHash ([32]byte)
	{
	}

	// RefTxs ([]byte)
	if m.ComplianceAction == 'R' {
		if len(m.RefTxs) > (2<<32)-1 {
			return fmt.Errorf("varbin field RefTxs too long %d/%d", len(m.RefTxs), (2<<32)-1)
		}
	}

	// BitcoinDispersions ([]QuantityIndex)
	if m.ComplianceAction == 'R' {
		if len(m.BitcoinDispersions) > (2<<16)-1 {
			return fmt.Errorf("list field BitcoinDispersions has too many items %d/%d", len(m.BitcoinDispersions), (2<<16)-1)
		}

		for i, value := range m.BitcoinDispersions {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field BitcoinDispersions[%d] is invalid : %s", i, err)
			}
		}
	}

	// Message (string)
	{
		if len(m.Message) > (2<<32)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<32)-1)
		}
	}

	return nil
}

func (action Order) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("ComplianceAction:%#+v", action.ComplianceAction))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("TargetAddresses:%#+v", action.TargetAddresses))
	vals = append(vals, fmt.Sprintf("FreezeTxId:%#+v", action.FreezeTxId))
	vals = append(vals, fmt.Sprintf("FreezePeriod:%#+v", action.FreezePeriod))
	vals = append(vals, fmt.Sprintf("DepositAddress:%#+v", action.DepositAddress))
	vals = append(vals, fmt.Sprintf("AuthorityIncluded:%#+v", action.AuthorityIncluded))
	vals = append(vals, fmt.Sprintf("AuthorityName:%#+v", action.AuthorityName))
	vals = append(vals, fmt.Sprintf("AuthorityPublicKey:%#x", action.AuthorityPublicKey))
	vals = append(vals, fmt.Sprintf("SignatureAlgorithm:%v", action.SignatureAlgorithm))
	vals = append(vals, fmt.Sprintf("OrderSignature:%#x", action.OrderSignature))
	vals = append(vals, fmt.Sprintf("SupportingEvidenceHash:%#+v", action.SupportingEvidenceHash))
	vals = append(vals, fmt.Sprintf("RefTxs:%#x", action.RefTxs))
	vals = append(vals, fmt.Sprintf("BitcoinDispersions:%#+v", action.BitcoinDispersions))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Freeze To be used to comply with contractual/legal/issuer requirements.
// The target public address(es) will be marked as frozen. However the
// Freeze action publishes this fact to the public blockchain for
// transparency. The Contract will not respond to any actions requested by
// the frozen address.
type Freeze struct {
	AssetType    string          `json:"asset_type,omitempty"`    // Three letter character that specifies the asset type.
	AssetCode    AssetCode       `json:"asset_code,omitempty"`    // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Quantities   []QuantityIndex `json:"quantities,omitempty"`    // Indices to addresses in outputs and the quantities being frozen. If the only address is the contract address and the asset code is zeros, then it is a contract wide freeze. If the only address is the contract address and the asset code is specified, then it is an asset wide freeze.
	FreezePeriod Timestamp       `json:"freeze_period,omitempty"` // Used for a 'time out'.  Tokens are automatically unfrozen after the expiration timestamp without requiring a Thaw Action.
	Timestamp    Timestamp       `json:"timestamp,omitempty"`     // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Freeze) Type() string {
	return CodeFreeze
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Freeze) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Freeze) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Quantities)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Quantities {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// FreezePeriod (Timestamp)
	{
		b, err := action.FreezePeriod.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Freeze from the byte slice
func (action *Freeze) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Quantities = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Quantities = append(action.Quantities, newValue)
		}
	}

	// FreezePeriod (Timestamp)
	{
		if err := action.FreezePeriod.Write(buf); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Freeze) Validate() error {

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

	// Quantities ([]QuantityIndex)
	{
		if len(m.Quantities) > (2<<16)-1 {
			return fmt.Errorf("list field Quantities has too many items %d/%d", len(m.Quantities), (2<<16)-1)
		}

		for i, value := range m.Quantities {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Quantities[%d] is invalid : %s", i, err)
			}
		}
	}

	// FreezePeriod (Timestamp)
	{
		if err := m.FreezePeriod.Validate(); err != nil {
			return fmt.Errorf("field FreezePeriod is invalid : %s", err)
		}

	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Freeze) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("Quantities:%#+v", action.Quantities))
	vals = append(vals, fmt.Sprintf("FreezePeriod:%#+v", action.FreezePeriod))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Thaw To be used to comply with contractual obligations or legal
// requirements. The Alleged Offender's tokens will be unfrozen to allow
// them to resume normal exchange and governance activities.
type Thaw struct {
	FreezeTxId TxId      `json:"freeze_tx_id,omitempty"` // The tx id of the freeze action that is being reversed.
	Timestamp  Timestamp `json:"timestamp,omitempty"`    // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Thaw) Type() string {
	return CodeThaw
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Thaw) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Thaw) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// FreezeTxId (TxId)
	{
		b, err := action.FreezeTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Thaw from the byte slice
func (action *Thaw) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// FreezeTxId (TxId)
	{
		if err := action.FreezeTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Thaw) Validate() error {

	// FreezeTxId (TxId)
	{
		if err := m.FreezeTxId.Validate(); err != nil {
			return fmt.Errorf("field FreezeTxId is invalid : %s", err)
		}

	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Thaw) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("FreezeTxId:%#+v", action.FreezeTxId))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Confiscation To be used to comply with contractual obligations, legal
// and/or issuer requirements.
type Confiscation struct {
	AssetType  string          `json:"asset_type,omitempty"`  // Three letter character that specifies the asset type.
	AssetCode  AssetCode       `json:"asset_code,omitempty"`  // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Quantities []QuantityIndex `json:"quantities,omitempty"`  // The holders effected by the confiscation and their balance remaining.
	DepositQty uint64          `json:"deposit_qty,omitempty"` // Deposit address's token balance after confiscation.
	Timestamp  Timestamp       `json:"timestamp,omitempty"`   // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Confiscation) Type() string {
	return CodeConfiscation
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Confiscation) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Confiscation) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Quantities)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Quantities {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// DepositQty (uint64)
	{
		if err := write(buf, action.DepositQty); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Confiscation from the byte slice
func (action *Confiscation) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Quantities = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Quantities = append(action.Quantities, newValue)
		}
	}

	// DepositQty (uint64)
	{
		if err := read(buf, &action.DepositQty); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Confiscation) Validate() error {

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

	// Quantities ([]QuantityIndex)
	{
		if len(m.Quantities) > (2<<16)-1 {
			return fmt.Errorf("list field Quantities has too many items %d/%d", len(m.Quantities), (2<<16)-1)
		}

		for i, value := range m.Quantities {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Quantities[%d] is invalid : %s", i, err)
			}
		}
	}

	// DepositQty (uint64)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Confiscation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("Quantities:%#+v", action.Quantities))
	vals = append(vals, fmt.Sprintf("DepositQty:%v", action.DepositQty))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Reconciliation To be used at the direction of the issuer to fix record
// keeping errors with bitcoin and token balances.
type Reconciliation struct {
	AssetType  string          `json:"asset_type,omitempty"` // Three letter character that specifies the asset type.
	AssetCode  AssetCode       `json:"asset_code,omitempty"` // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Quantities []QuantityIndex `json:"quantities,omitempty"` // The holders effected by the reconciliation and their balance remaining.
	Timestamp  Timestamp       `json:"timestamp,omitempty"`  // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Reconciliation) Type() string {
	return CodeReconciliation
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Reconciliation) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Reconciliation) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetType (string)
	{
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	{
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Quantities)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Quantities {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Reconciliation from the byte slice
func (action *Reconciliation) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetType (string)
	{
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	{
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// Quantities ([]QuantityIndex)
	{
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Quantities = make([]QuantityIndex, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue QuantityIndex
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Quantities = append(action.Quantities, newValue)
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Reconciliation) Validate() error {

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

	// Quantities ([]QuantityIndex)
	{
		if len(m.Quantities) > (2<<16)-1 {
			return fmt.Errorf("list field Quantities has too many items %d/%d", len(m.Quantities), (2<<16)-1)
		}

		for i, value := range m.Quantities {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Quantities[%d] is invalid : %s", i, err)
			}
		}
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Reconciliation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("Quantities:%#+v", action.Quantities))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Proposal Allows Issuers/Token Holders to propose a change (aka
// Initiative/Shareholder vote). A significant cost - specified in the
// Contract Formation - can be attached to this action when sent from Token
// Holders to reduce spam, as the resulting vote will be put to all token
// owners.
type Proposal struct {
	Initiator            uint8       `json:"initiator,omitempty"`              // Who initiated the proposal. Supported values: 0 - Issuer, 1 - Holder
	AssetSpecificVote    bool        `json:"asset_specific_vote,omitempty"`    // When true this proposal is specific to an asset and the asset type and asset code fields are serialized.
	AssetType            string      `json:"asset_type,omitempty"`             // Three letter character that specifies the asset type.
	AssetCode            AssetCode   `json:"asset_code,omitempty"`             // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	VoteSystem           uint8       `json:"vote_system,omitempty"`            // X for Vote System X. (1-255, 0 is not valid.)
	Specific             bool        `json:"specific,omitempty"`               // When true the ProposedAmendments field is included and specifies the exact changes to make to the Contract/Asset on chain. When false this is just a general proposal like a strategy/direction and doesn't result in any on chain update.
	ProposedAmendments   []Amendment `json:"proposed_amendments,omitempty"`    // Each element contains details of which fields to modify, or delete. Because the number of fields in a Contract and Asset is dynamic due to some fields being able to be repeated, the index value of the field needs to be calculated against the Contract or Asset the changes are to apply to. In the event of a Vote being created from this Initiative, the changes will be applied to the version of the Contract or Asset at that time.
	VoteOptions          string      `json:"vote_options,omitempty"`           // Length 1-255 bytes. 0 is not valid. Each byte allows for a different vote option. Typical votes will likely be multiple choice or Y/N. Vote instances are identified by the Tx-ID. AB would be used for Y/N (binary) type votes. When Specific is true, only AB is a valid value.
	VoteMax              uint8       `json:"vote_max,omitempty"`               // Range: 1-X. How many selections can a voter make in a Ballot Cast. 1 is selected for Y/N (binary). When Specific is true, only 1 is a valid value.
	ProposalDescription  string      `json:"proposal_description,omitempty"`   // Length restricted by the Bitcoin protocol. 0 is valid. Description or details of the vote
	ProposalDocumentHash [32]byte    `json:"proposal_document_hash,omitempty"` // SHA256 Hash of the proposal document to be distributed to voters.
	VoteCutOffTimestamp  Timestamp   `json:"vote_cut_off_timestamp,omitempty"` // Ballot casts after this timestamp will not be included. The vote has finished.
}

// Type returns the type identifer for this message.
func (action Proposal) Type() string {
	return CodeProposal
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Proposal) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Proposal) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Initiator (uint8)
	{
		if err := write(buf, action.Initiator); err != nil {
			return nil, err
		}
	}

	// AssetSpecificVote (bool)
	{
		if err := write(buf, action.AssetSpecificVote); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	if action.AssetSpecificVote {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	if action.AssetSpecificVote {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// VoteSystem (uint8)
	{
		if err := write(buf, action.VoteSystem); err != nil {
			return nil, err
		}
	}

	// Specific (bool)
	{
		if err := write(buf, action.Specific); err != nil {
			return nil, err
		}
	}

	// ProposedAmendments ([]Amendment)
	if action.Specific {
		if err := WriteVariableSize(buf, uint64(len(action.ProposedAmendments)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ProposedAmendments {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// VoteOptions (string)
	{
		if err := WriteVarChar(buf, action.VoteOptions, 8); err != nil {
			return nil, err
		}
	}

	// VoteMax (uint8)
	{
		if err := write(buf, action.VoteMax); err != nil {
			return nil, err
		}
	}

	// ProposalDescription (string)
	{
		if err := WriteVarChar(buf, action.ProposalDescription, 32); err != nil {
			return nil, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	{
		if err := write(buf, action.ProposalDocumentHash); err != nil {
			return nil, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	{
		b, err := action.VoteCutOffTimestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Proposal from the byte slice
func (action *Proposal) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Initiator (uint8)
	{
		if err := read(buf, &action.Initiator); err != nil {
			return 0, err
		}
	}

	// AssetSpecificVote (bool)
	{
		if err := read(buf, &action.AssetSpecificVote); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	if action.AssetSpecificVote {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	if action.AssetSpecificVote {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// VoteSystem (uint8)
	{
		if err := read(buf, &action.VoteSystem); err != nil {
			return 0, err
		}
	}

	// Specific (bool)
	{
		if err := read(buf, &action.Specific); err != nil {
			return 0, err
		}
	}

	// ProposedAmendments ([]Amendment)
	if action.Specific {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.ProposedAmendments = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ProposedAmendments = append(action.ProposedAmendments, newValue)
		}
	}

	// VoteOptions (string)
	{
		var err error
		action.VoteOptions, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// VoteMax (uint8)
	{
		if err := read(buf, &action.VoteMax); err != nil {
			return 0, err
		}
	}

	// ProposalDescription (string)
	{
		var err error
		action.ProposalDescription, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	{
		if err := read(buf, &action.ProposalDocumentHash); err != nil {
			return 0, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	{
		if err := action.VoteCutOffTimestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Proposal) Validate() error {

	// Initiator (uint8)
	{
		if m.Initiator != 0 && m.Initiator != 1 {
			return fmt.Errorf("field Initiator value is invalid : %d", m.Initiator)
		}

	}

	// AssetSpecificVote (bool)
	{
	}

	// AssetType (string)
	if m.AssetSpecificVote {
		if len(m.AssetType) > 3 {
			return fmt.Errorf("fixedchar field AssetType too long %d/%d", len(m.AssetType), 3)
		}
	}

	// AssetCode (AssetCode)
	if m.AssetSpecificVote {
		if err := m.AssetCode.Validate(); err != nil {
			return fmt.Errorf("field AssetCode is invalid : %s", err)
		}

	}

	// VoteSystem (uint8)
	{
	}

	// Specific (bool)
	{
	}

	// ProposedAmendments ([]Amendment)
	if m.Specific {
		if len(m.ProposedAmendments) > (2<<0)-1 {
			return fmt.Errorf("list field ProposedAmendments has too many items %d/%d", len(m.ProposedAmendments), (2<<0)-1)
		}

		for i, value := range m.ProposedAmendments {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field ProposedAmendments[%d] is invalid : %s", i, err)
			}
		}
	}

	// VoteOptions (string)
	{
		if len(m.VoteOptions) > (2<<8)-1 {
			return fmt.Errorf("varchar field VoteOptions too long %d/%d", len(m.VoteOptions), (2<<8)-1)
		}
	}

	// VoteMax (uint8)
	{
	}

	// ProposalDescription (string)
	{
		if len(m.ProposalDescription) > (2<<32)-1 {
			return fmt.Errorf("varchar field ProposalDescription too long %d/%d", len(m.ProposalDescription), (2<<32)-1)
		}
	}

	// ProposalDocumentHash ([32]byte)
	{
	}

	// VoteCutOffTimestamp (Timestamp)
	{
		if err := m.VoteCutOffTimestamp.Validate(); err != nil {
			return fmt.Errorf("field VoteCutOffTimestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Proposal) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Initiator:%v", action.Initiator))
	vals = append(vals, fmt.Sprintf("AssetSpecificVote:%#+v", action.AssetSpecificVote))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("VoteSystem:%v", action.VoteSystem))
	vals = append(vals, fmt.Sprintf("Specific:%#+v", action.Specific))
	vals = append(vals, fmt.Sprintf("ProposedAmendments:%#+v", action.ProposedAmendments))
	vals = append(vals, fmt.Sprintf("VoteOptions:%#+v", action.VoteOptions))
	vals = append(vals, fmt.Sprintf("VoteMax:%v", action.VoteMax))
	vals = append(vals, fmt.Sprintf("ProposalDescription:%#+v", action.ProposalDescription))
	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:%#+v", action.ProposalDocumentHash))
	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%#+v", action.VoteCutOffTimestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Vote A vote is created by the Contract in response to a valid Proposal
// Action.
type Vote struct {
	Timestamp Timestamp `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Vote) Type() string {
	return CodeVote
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Vote) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Vote) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Vote from the byte slice
func (action *Vote) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Vote) Validate() error {

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Vote) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCast Used by Token Owners to cast their ballot (vote) on
// proposals. 1 Vote per token unless a vote multiplier is specified in the
// relevant Asset Definition action.
type BallotCast struct {
	VoteTxId TxId   `json:"vote_tx_id,omitempty"` // Tx ID of the Vote the Ballot Cast is being made for.
	Vote     string `json:"vote,omitempty"`       // Length 1-255 bytes. 0 is not valid. Max length is the VoteMax value from the Proposal action. Accept, Reject, Abstain, Spoiled, Multiple Choice, or Preference List. 15 options total. Order of preference. 1st position = 1st choice. 2nd position = 2nd choice, etc. A is always Accept and B is always reject in a Y/N votes.
}

// Type returns the type identifer for this message.
func (action BallotCast) Type() string {
	return CodeBallotCast
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *BallotCast) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *BallotCast) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// VoteTxId (TxId)
	{
		b, err := action.VoteTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Vote (string)
	{
		if err := WriteVarChar(buf, action.Vote, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in BallotCast from the byte slice
func (action *BallotCast) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// VoteTxId (TxId)
	{
		if err := action.VoteTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// Vote (string)
	{
		var err error
		action.Vote, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *BallotCast) Validate() error {

	// VoteTxId (TxId)
	{
		if err := m.VoteTxId.Validate(); err != nil {
			return fmt.Errorf("field VoteTxId is invalid : %s", err)
		}

	}

	// Vote (string)
	{
		if len(m.Vote) > (2<<8)-1 {
			return fmt.Errorf("varchar field Vote too long %d/%d", len(m.Vote), (2<<8)-1)
		}
	}

	return nil
}

func (action BallotCast) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("VoteTxId:%#+v", action.VoteTxId))
	vals = append(vals, fmt.Sprintf("Vote:%#+v", action.Vote))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCounted The smart contract will respond to a Ballot Cast action
// with a Ballot Counted action if the Ballot Cast is valid. If the Ballot
// Cast is not valid, then the smart contract will respond with a Rejection
// Action.
type BallotCounted struct {
	VoteTxId  TxId      `json:"vote_tx_id,omitempty"` // Tx ID of the Vote the Ballot Cast is being made for.
	Vote      string    `json:"vote,omitempty"`       // Length 1-255 bytes. 0 is not valid. Max length is the VoteMax value from the Proposal action. Accept, Reject, Abstain, Spoiled, Multiple Choice, or Preference List. 15 options total. Order of preference. 1st position = 1st choice. 2nd position = 2nd choice, etc. A is always Accept and B is always reject in a Y/N votes.
	Quantity  uint64    `json:"quantity,omitempty"`   // Number of votes counted for this ballot. Factors in vote multipliers if there are any allowed, otherwise it is just quantity of tokens held.
	Timestamp Timestamp `json:"timestamp,omitempty"`  // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action BallotCounted) Type() string {
	return CodeBallotCounted
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *BallotCounted) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *BallotCounted) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// VoteTxId (TxId)
	{
		b, err := action.VoteTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Vote (string)
	{
		if err := WriteVarChar(buf, action.Vote, 8); err != nil {
			return nil, err
		}
	}

	// Quantity (uint64)
	{
		if err := write(buf, action.Quantity); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in BallotCounted from the byte slice
func (action *BallotCounted) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// VoteTxId (TxId)
	{
		if err := action.VoteTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// Vote (string)
	{
		var err error
		action.Vote, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Quantity (uint64)
	{
		if err := read(buf, &action.Quantity); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *BallotCounted) Validate() error {

	// VoteTxId (TxId)
	{
		if err := m.VoteTxId.Validate(); err != nil {
			return fmt.Errorf("field VoteTxId is invalid : %s", err)
		}

	}

	// Vote (string)
	{
		if len(m.Vote) > (2<<8)-1 {
			return fmt.Errorf("varchar field Vote too long %d/%d", len(m.Vote), (2<<8)-1)
		}
	}

	// Quantity (uint64)
	{
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action BallotCounted) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("VoteTxId:%#+v", action.VoteTxId))
	vals = append(vals, fmt.Sprintf("Vote:%#+v", action.Vote))
	vals = append(vals, fmt.Sprintf("Quantity:%v", action.Quantity))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Result Once a vote has been completed the results are published. After
// the result is posted, it is up to the issuer to send a contract/asset
// amendement if appropriate.
type Result struct {
	AssetSpecificVote  bool        `json:"asset_specific_vote,omitempty"` // When true this proposal is specific to an asset and the asset type and asset code fields are serialized.
	AssetType          string      `json:"asset_type,omitempty"`          // Three letter character that specifies the asset type.
	AssetCode          AssetCode   `json:"asset_code,omitempty"`          // A unique code that is used to identify the asset, made up of 32 randomly generated bytes. The asset code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Specific           bool        `json:"specific,omitempty"`            // When true the ProposedAmendments field is included and specifies the exact changes to make to the Contract/Asset on chain. When false this is just a general proposal like a strategy/direction and doesn't result in any on chain update.
	ProposedAmendments []Amendment `json:"proposed_amendments,omitempty"` // Each element contains details of which fields to modify, or delete. Because the number of fields in a Contract and Asset is dynamic due to some fields being able to be repeated, the index value of the field needs to be calculated against the Contract or Asset the changes are to apply to. In the event of a Vote being created from this Initiative, the changes will be applied to the version of the Contract or Asset at that time.
	VoteTxId           TxId        `json:"vote_tx_id,omitempty"`          // Link to the Vote Action txn.
	OptionTally        []uint64    `json:"option_tally,omitempty"`        // List of number of valid votes counted for each vote option. Length is encoded like a regular list object, but must match the length of VoteOptions from the Proposal action.
	Result             string      `json:"result,omitempty"`              // Length 1-255 bytes. 0 is not valid. The Option with the most votes. In the event of a draw for 1st place, all winning options are listed.
	Timestamp          Timestamp   `json:"timestamp,omitempty"`           // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Result) Type() string {
	return CodeResult
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Result) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Result) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AssetSpecificVote (bool)
	{
		if err := write(buf, action.AssetSpecificVote); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	if action.AssetSpecificVote {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	if action.AssetSpecificVote {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Specific (bool)
	{
		if err := write(buf, action.Specific); err != nil {
			return nil, err
		}
	}

	// ProposedAmendments ([]Amendment)
	if action.Specific {
		if err := WriteVariableSize(buf, uint64(len(action.ProposedAmendments)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ProposedAmendments {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// VoteTxId (TxId)
	{
		b, err := action.VoteTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// OptionTally ([]uint64)
	{
		if err := WriteVariableSize(buf, uint64(len(action.OptionTally)), 8, 8); err != nil {
			return nil, err
		}
		for _, value := range action.OptionTally {
			if err := write(buf, value); err != nil {
				return nil, err
			}
		}
	}

	// Result (string)
	{
		if err := WriteVarChar(buf, action.Result, 8); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Result from the byte slice
func (action *Result) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AssetSpecificVote (bool)
	{
		if err := read(buf, &action.AssetSpecificVote); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	if action.AssetSpecificVote {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	if action.AssetSpecificVote {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// Specific (bool)
	{
		if err := read(buf, &action.Specific); err != nil {
			return 0, err
		}
	}

	// ProposedAmendments ([]Amendment)
	if action.Specific {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.ProposedAmendments = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ProposedAmendments = append(action.ProposedAmendments, newValue)
		}
	}

	// VoteTxId (TxId)
	{
		if err := action.VoteTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// OptionTally ([]uint64)
	{
		size, err := ReadVariableSize(buf, 8, 8)
		if err != nil {
			return 0, err
		}
		action.OptionTally = make([]uint64, size, size)
		if err := read(buf, &action.OptionTally); err != nil {
			return 0, err
		}
	}

	// Result (string)
	{
		var err error
		action.Result, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Result) Validate() error {

	// AssetSpecificVote (bool)
	{
	}

	// AssetType (string)
	if m.AssetSpecificVote {
		if len(m.AssetType) > 3 {
			return fmt.Errorf("fixedchar field AssetType too long %d/%d", len(m.AssetType), 3)
		}
	}

	// AssetCode (AssetCode)
	if m.AssetSpecificVote {
		if err := m.AssetCode.Validate(); err != nil {
			return fmt.Errorf("field AssetCode is invalid : %s", err)
		}

	}

	// Specific (bool)
	{
	}

	// ProposedAmendments ([]Amendment)
	if m.Specific {
		if len(m.ProposedAmendments) > (2<<0)-1 {
			return fmt.Errorf("list field ProposedAmendments has too many items %d/%d", len(m.ProposedAmendments), (2<<0)-1)
		}

		for i, value := range m.ProposedAmendments {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field ProposedAmendments[%d] is invalid : %s", i, err)
			}
		}
	}

	// VoteTxId (TxId)
	{
		if err := m.VoteTxId.Validate(); err != nil {
			return fmt.Errorf("field VoteTxId is invalid : %s", err)
		}

	}

	// OptionTally ([]uint64)
	{
		if len(m.OptionTally) > (2<<8)-1 {
			return fmt.Errorf("list field OptionTally has too many items %d/%d", len(m.OptionTally), (2<<8)-1)
		}
	}

	// Result (string)
	{
		if len(m.Result) > (2<<8)-1 {
			return fmt.Errorf("varchar field Result too long %d/%d", len(m.Result), (2<<8)-1)
		}
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Result) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AssetSpecificVote:%#+v", action.AssetSpecificVote))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("Specific:%#+v", action.Specific))
	vals = append(vals, fmt.Sprintf("ProposedAmendments:%#+v", action.ProposedAmendments))
	vals = append(vals, fmt.Sprintf("VoteTxId:%#+v", action.VoteTxId))
	vals = append(vals, fmt.Sprintf("OptionTally:%v", action.OptionTally))
	vals = append(vals, fmt.Sprintf("Result:%#+v", action.Result))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Message The message action is a general purpose communication action.
// 'Twitter/SMS' for Issuers/Investors/Users. The message txn can also be
// used for passing partially signed txns on-chain, establishing private
// communication channels and EDI (receipting, invoices, PO, and private
// offers/bids). The messages are broken down by type for easy filtering in
// the a user's wallet. The Message Types are listed in the Message Types
// table.
type Message struct {
	AddressIndexes []uint16 `json:"address_indexes,omitempty"` // Associates the message to a particular output by the index.
	MessageType    uint16   `json:"message_type,omitempty"`    // Potential for up to 65,535 different message types. Values from resources/Messages.yaml
	MessagePayload []byte   `json:"message_payload,omitempty"` // Public or private (RSA public key, Diffie-Hellman). Issuers/Contracts can send the signifying amount of satoshis to themselves for public announcements or private 'notes' if encrypted. See Message Types for a full list of potential use cases.

}

// Type returns the type identifer for this message.
func (action Message) Type() string {
	return CodeMessage
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Message) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Message) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AddressIndexes ([]uint16)
	{
		if err := WriteVariableSize(buf, uint64(len(action.AddressIndexes)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.AddressIndexes {
			if err := write(buf, value); err != nil {
				return nil, err
			}
		}
	}

	// MessageType (uint16)
	{
		if err := write(buf, action.MessageType); err != nil {
			return nil, err
		}
	}

	// MessagePayload ([]byte)
	{
		if err := WriteVarBin(buf, action.MessagePayload, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Message from the byte slice
func (action *Message) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AddressIndexes ([]uint16)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.AddressIndexes = make([]uint16, size, size)
		if err := read(buf, &action.AddressIndexes); err != nil {
			return 0, err
		}
	}

	// MessageType (uint16)
	{
		if err := read(buf, &action.MessageType); err != nil {
			return 0, err
		}
	}

	// MessagePayload ([]byte)
	{
		var err error
		action.MessagePayload, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Message) Validate() error {

	// AddressIndexes ([]uint16)
	{
		if len(m.AddressIndexes) > (2<<0)-1 {
			return fmt.Errorf("list field AddressIndexes has too many items %d/%d", len(m.AddressIndexes), (2<<0)-1)
		}
	}

	// MessageType (uint16)
	{
	}

	// MessagePayload ([]byte)
	{
		if len(m.MessagePayload) > (2<<32)-1 {
			return fmt.Errorf("varbin field MessagePayload too long %d/%d", len(m.MessagePayload), (2<<32)-1)
		}
	}

	return nil
}

func (action Message) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AddressIndexes:%v", action.AddressIndexes))
	vals = append(vals, fmt.Sprintf("MessageType:%#+v", action.MessageType))
	vals = append(vals, fmt.Sprintf("MessagePayload:%#x", action.MessagePayload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Rejection Used to reject request actions that do not comply with the
// Contract. If money is to be returned to a User then it is used in lieu
// of the Settlement Action to properly account for token balances. All
// Issuer/User request Actions must be responded to by the Contract with an
// Action. The only exception to this rule is when there is not enough fees
// in the first Action for the Contract response action to remain revenue
// neutral. If not enough fees are attached to pay for the Contract
// response then the Contract will not respond.
type Rejection struct {
	AddressIndexes     []uint16  `json:"address_indexes,omitempty"`      // Associates the message to a particular output by the index.
	RejectAddressIndex uint16    `json:"reject_address_index,omitempty"` // The address which is believed to have caused the rejection.
	RejectionCode      uint8     `json:"rejection_code,omitempty"`       // Classifies the rejection by a type.
	Message            string    `json:"message,omitempty"`              // Length 0-65,535 bytes. Message that explains the reasoning for a rejection, if needed.  Most rejection types will be captured by the Rejection Type Subfield.
	Timestamp          Timestamp `json:"timestamp,omitempty"`            // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Rejection) Type() string {
	return CodeRejection
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Rejection) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Rejection) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// AddressIndexes ([]uint16)
	{
		if err := WriteVariableSize(buf, uint64(len(action.AddressIndexes)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.AddressIndexes {
			if err := write(buf, value); err != nil {
				return nil, err
			}
		}
	}

	// RejectAddressIndex (uint16)
	{
		if err := write(buf, action.RejectAddressIndex); err != nil {
			return nil, err
		}
	}

	// RejectionCode (uint8)
	{
		if err := write(buf, action.RejectionCode); err != nil {
			return nil, err
		}
	}

	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 16); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Rejection from the byte slice
func (action *Rejection) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// AddressIndexes ([]uint16)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.AddressIndexes = make([]uint16, size, size)
		if err := read(buf, &action.AddressIndexes); err != nil {
			return 0, err
		}
	}

	// RejectAddressIndex (uint16)
	{
		if err := read(buf, &action.RejectAddressIndex); err != nil {
			return 0, err
		}
	}

	// RejectionCode (uint8)
	{
		if err := read(buf, &action.RejectionCode); err != nil {
			return 0, err
		}
	}

	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Rejection) Validate() error {

	// AddressIndexes ([]uint16)
	{
		if len(m.AddressIndexes) > (2<<0)-1 {
			return fmt.Errorf("list field AddressIndexes has too many items %d/%d", len(m.AddressIndexes), (2<<0)-1)
		}
	}

	// RejectAddressIndex (uint16)
	{
	}

	// RejectionCode (uint8)
	{
		if GetRejectionCode(m.RejectionCode) == nil {
			return fmt.Errorf("Invalid rejection code value : %d", m.RejectionCode)
		}
	}

	// Message (string)
	{
		if len(m.Message) > (2<<16)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<16)-1)
		}
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Rejection) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("AddressIndexes:%v", action.AddressIndexes))
	vals = append(vals, fmt.Sprintf("RejectAddressIndex:%v", action.RejectAddressIndex))
	vals = append(vals, fmt.Sprintf("RejectionCode:%#+v", action.RejectionCode))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Establishment Establishes an on-chain register.
type Establishment struct {
	Message string `json:"message,omitempty"` // A custom message to include with this action.
}

// Type returns the type identifer for this message.
func (action Establishment) Type() string {
	return CodeEstablishment
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Establishment) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Establishment) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Establishment from the byte slice
func (action *Establishment) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Establishment) Validate() error {

	// Message (string)
	{
		if len(m.Message) > (2<<32)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<32)-1)
		}
	}

	return nil
}

func (action Establishment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Addition Adds an entry to the Register.
type Addition struct {
	Message string `json:"message,omitempty"` // A custom message to include with this action.
}

// Type returns the type identifer for this message.
func (action Addition) Type() string {
	return CodeAddition
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Addition) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Addition) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Addition from the byte slice
func (action *Addition) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Addition) Validate() error {

	// Message (string)
	{
		if len(m.Message) > (2<<32)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<32)-1)
		}
	}

	return nil
}

func (action Addition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Alteration A register entry/record can be altered.
type Alteration struct {
	EntryTxID TxId   `json:"entry_tx_id,omitempty"` // Transaction ID of the register entry to be altered.
	Message   string `json:"message,omitempty"`     // A custom message to include with this action.
}

// Type returns the type identifer for this message.
func (action Alteration) Type() string {
	return CodeAlteration
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Alteration) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Alteration) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// EntryTxID (TxId)
	{
		b, err := action.EntryTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Alteration from the byte slice
func (action *Alteration) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// EntryTxID (TxId)
	{
		if err := action.EntryTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Alteration) Validate() error {

	// EntryTxID (TxId)
	{
		if err := m.EntryTxID.Validate(); err != nil {
			return fmt.Errorf("field EntryTxID is invalid : %s", err)
		}

	}

	// Message (string)
	{
		if len(m.Message) > (2<<32)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<32)-1)
		}
	}

	return nil
}

func (action Alteration) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("EntryTxID:%#+v", action.EntryTxID))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Removal Removes an entry/record from the Register.
type Removal struct {
	EntryTxID TxId   `json:"entry_tx_id,omitempty"` // Transaction ID of the register entry to be altered.
	Message   string `json:"message,omitempty"`     // A custom message to include with this action.
}

// Type returns the type identifer for this message.
func (action Removal) Type() string {
	return CodeRemoval
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Removal) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Removal) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// EntryTxID (TxId)
	{
		b, err := action.EntryTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Message (string)
	{
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Removal from the byte slice
func (action *Removal) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// EntryTxID (TxId)
	{
		if err := action.EntryTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	{
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Removal) Validate() error {

	// EntryTxID (TxId)
	{
		if err := m.EntryTxID.Validate(); err != nil {
			return fmt.Errorf("field EntryTxID is invalid : %s", err)
		}

	}

	// Message (string)
	{
		if len(m.Message) > (2<<32)-1 {
			return fmt.Errorf("varchar field Message too long %d/%d", len(m.Message), (2<<32)-1)
		}
	}

	return nil
}

func (action Removal) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("EntryTxID:%#+v", action.EntryTxID))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Transfer A Token Owner(s) Sends, Exchanges or Swaps a token(s) or
// Bitcoin for a token(s) or Bitcoin. Can be as simple as sending a single
// token to a receiver. Or can be as complex as many senders sending many
// different assets - controlled by many different smart contracts - to a
// number of receivers. This action also supports atomic swaps (tokens for
// tokens). Since many parties and contracts can be involved in a transfer
// and the corresponding settlement action, the partially signed T1 and T2
// actions will need to be passed around on-chain with an M1 action, or
// off-chain.
type Transfer struct {
	Assets             []AssetTransfer `json:"assets,omitempty"`               // The Assets involved in the Transfer Action.
	OfferExpiry        Timestamp       `json:"offer_expiry,omitempty"`         // This prevents any party from holding on to the partially signed message as a form of an option.  Eg. the exchange at this price is valid for 30 mins.
	ExchangeFee        uint64          `json:"exchange_fee,omitempty"`         // Fixed amount of bitcoin being paid to an exchange for facilitating a transfer.
	ExchangeFeeAddress PublicKeyHash   `json:"exchange_fee_address,omitempty"` // Identifies the public address that the exchange fee should be paid to.
}

// Type returns the type identifer for this message.
func (action Transfer) Type() string {
	return CodeTransfer
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Transfer) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Transfer) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Assets ([]AssetTransfer)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Assets)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Assets {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// OfferExpiry (Timestamp)
	{
		b, err := action.OfferExpiry.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ExchangeFee (uint64)
	{
		if err := write(buf, action.ExchangeFee); err != nil {
			return nil, err
		}
	}

	// ExchangeFeeAddress (PublicKeyHash)
	{
		b, err := action.ExchangeFeeAddress.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Transfer from the byte slice
func (action *Transfer) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Assets ([]AssetTransfer)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Assets = make([]AssetTransfer, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue AssetTransfer
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Assets = append(action.Assets, newValue)
		}
	}

	// OfferExpiry (Timestamp)
	{
		if err := action.OfferExpiry.Write(buf); err != nil {
			return 0, err
		}
	}

	// ExchangeFee (uint64)
	{
		if err := read(buf, &action.ExchangeFee); err != nil {
			return 0, err
		}
	}

	// ExchangeFeeAddress (PublicKeyHash)
	{
		if err := action.ExchangeFeeAddress.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Transfer) Validate() error {

	// Assets ([]AssetTransfer)
	{
		if len(m.Assets) > (2<<0)-1 {
			return fmt.Errorf("list field Assets has too many items %d/%d", len(m.Assets), (2<<0)-1)
		}

		for i, value := range m.Assets {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Assets[%d] is invalid : %s", i, err)
			}
		}
	}

	// OfferExpiry (Timestamp)
	{
		if err := m.OfferExpiry.Validate(); err != nil {
			return fmt.Errorf("field OfferExpiry is invalid : %s", err)
		}

	}

	// ExchangeFee (uint64)
	{
	}

	// ExchangeFeeAddress (PublicKeyHash)
	{
		if err := m.ExchangeFeeAddress.Validate(); err != nil {
			return fmt.Errorf("field ExchangeFeeAddress is invalid : %s", err)
		}

	}

	return nil
}

func (action Transfer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Assets:%#+v", action.Assets))
	vals = append(vals, fmt.Sprintf("OfferExpiry:%#+v", action.OfferExpiry))
	vals = append(vals, fmt.Sprintf("ExchangeFee:%v", action.ExchangeFee))
	vals = append(vals, fmt.Sprintf("ExchangeFeeAddress:%#+v", action.ExchangeFeeAddress))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Settlement Settles the transfer request of bitcoins and tokens from
// transfer (T1) actions.
type Settlement struct {
	Assets    []AssetSettlement `json:"assets,omitempty"`    // The Assets settled by the transfer action.
	Timestamp Timestamp         `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
}

// Type returns the type identifer for this message.
func (action Settlement) Type() string {
	return CodeSettlement
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Settlement) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Settlement) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	// Assets ([]AssetSettlement)
	{
		if err := WriteVariableSize(buf, uint64(len(action.Assets)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Assets {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// Timestamp (Timestamp)
	{
		b, err := action.Timestamp.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Settlement from the byte slice
func (action *Settlement) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	// Assets ([]AssetSettlement)
	{
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Assets = make([]AssetSettlement, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue AssetSettlement
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Assets = append(action.Assets, newValue)
		}
	}

	// Timestamp (Timestamp)
	{
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

func (m *Settlement) Validate() error {

	// Assets ([]AssetSettlement)
	{
		if len(m.Assets) > (2<<0)-1 {
			return fmt.Errorf("list field Assets has too many items %d/%d", len(m.Assets), (2<<0)-1)
		}

		for i, value := range m.Assets {
			err := value.Validate()
			if err != nil {
				return fmt.Errorf("list field Assets[%d] is invalid : %s", i, err)
			}
		}
	}

	// Timestamp (Timestamp)
	{
		if err := m.Timestamp.Validate(); err != nil {
			return fmt.Errorf("field Timestamp is invalid : %s", err)
		}

	}

	return nil
}

func (action Settlement) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Assets:%#+v", action.Assets))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
