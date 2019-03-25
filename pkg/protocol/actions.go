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

	// CodeTransfer identifies data as a Transfer message.
	CodeTransfer = "T1"

	// CodeSettlement identifies data as a Settlement message.
	CodeSettlement = "T4"

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
	case CodeInitiative:
		result := Initiative{}
		return &result
	case CodeReferendum:
		result := Referendum{}
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

// AssetDefinition Asset Definition Action - This action is used by the
// issuer to define the properties/characteristics of the Asset (token)
// that it wants to create.
type AssetDefinition struct {
	Header                      Header    `json:"header,omitempty"`                        // Common header data for all actions
	AssetType                   string    `json:"asset_type,omitempty"`                    // eg. Share - Common
	AssetCode                   AssetCode `json:"asset_code,omitempty"`                    // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetAuthFlags              [8]byte   `json:"asset_auth_flags,omitempty"`              // Authorization Flags,  bitwise operation
	TransfersPermitted          bool      `json:"transfers_permitted,omitempty"`           // 1 = Transfers are permitted.  0 = Transfers are not permitted.
	TradeRestrictions           Polity    `json:"trade_restrictions,omitempty"`            // Asset can only be traded within the trade restrictions.  Eg. AUS - Australian residents only.  EU - European Union residents only.
	EnforcementOrdersPermitted  bool      `json:"enforcement_orders_permitted,omitempty"`  // 1 = Enforcement Orders are permitted. 0 = Enforcement Orders are not permitted.
	VoteMultiplier              uint8     `json:"vote_multiplier,omitempty"`               // Multiplies the vote by the integer. 1 token = 1 vote with a 1 for vote multipler (normal).  1 token = 3 votes with a multiplier of 3, for example.
	ReferendumProposal          bool      `json:"referendum_proposal,omitempty"`           // A Referendum is permitted for Asset-Wide Proposals (outside of smart contract scope) if also permitted by the contract. If the contract has proposals by referendum restricted, then this flag is meaningless.
	InitiativeProposal          bool      `json:"initiative_proposal,omitempty"`           // An initiative is permitted for Asset-Wide Proposals (outside of smart contract scope) if also permitted by the contract. If the contract has proposals by initiative restricted, then this flag is meaningless.
	AssetModificationGovernance bool      `json:"asset_modification_governance,omitempty"` // 1 - Contract-wide Asset Governance.  0 - Asset-wide Asset Governance.  If a referendum or initiative is used to propose a modification to a subfield controlled by the asset auth flags, then the vote will either be a contract-wide vote (all assets vote on the referendum/initiative) or an asset-wide vote (all assets vote on the referendum/initiative) depending on the value in this subfield.  The voting system specifies the voting rules.
	TokenQty                    uint64    `json:"token_qty,omitempty"`                     // Quantity of token - 0 is valid. Fungible 'shares' of the Asset. 1 is used for non-fungible tokens.  Asset Codes become the non-fungible Asset Code and many Asset Codes can be associated with a particular Contract.
	AssetPayload                []byte    `json:"asset_payload,omitempty"`                 // Payload length is dependent on the asset type. Each asset is made up of a defined set of information pertaining to the specific asset type, and may contain fields of variable length type (nvarchar8, 16, 32)
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetAuthFlags ([8]byte)
	_, skip = excludes["AssetAuthFlags"]
	if !skip {
		if err := write(buf, action.AssetAuthFlags); err != nil {
			return nil, err
		}
	}

	// TransfersPermitted (bool)
	_, skip = excludes["TransfersPermitted"]
	if !skip {
		if err := write(buf, action.TransfersPermitted); err != nil {
			return nil, err
		}
	}

	// TradeRestrictions (Polity)
	_, skip = excludes["TradeRestrictions"]
	if !skip {
		b, err := action.TradeRestrictions.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// EnforcementOrdersPermitted (bool)
	_, skip = excludes["EnforcementOrdersPermitted"]
	if !skip {
		if err := write(buf, action.EnforcementOrdersPermitted); err != nil {
			return nil, err
		}
	}

	// VoteMultiplier (uint8)
	_, skip = excludes["VoteMultiplier"]
	if !skip {
		if err := write(buf, action.VoteMultiplier); err != nil {
			return nil, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := write(buf, action.ReferendumProposal); err != nil {
			return nil, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := write(buf, action.InitiativeProposal); err != nil {
			return nil, err
		}
	}

	// AssetModificationGovernance (bool)
	_, skip = excludes["AssetModificationGovernance"]
	if !skip {
		if err := write(buf, action.AssetModificationGovernance); err != nil {
			return nil, err
		}
	}

	// TokenQty (uint64)
	_, skip = excludes["TokenQty"]
	if !skip {
		if err := write(buf, action.TokenQty); err != nil {
			return nil, err
		}
	}

	// AssetPayload ([]byte)
	_, skip = excludes["AssetPayload"]
	if !skip {
		if err := WriteVarBin(buf, action.AssetPayload, 16); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in AssetDefinition from the byte slice
func (action *AssetDefinition) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetAuthFlags ([8]byte)
	_, skip = excludes["AssetAuthFlags"]
	if !skip {
		if err := read(buf, &action.AssetAuthFlags); err != nil {
			return 0, err
		}
	}

	// TransfersPermitted (bool)
	_, skip = excludes["TransfersPermitted"]
	if !skip {
		if err := read(buf, &action.TransfersPermitted); err != nil {
			return 0, err
		}
	}

	// TradeRestrictions (Polity)
	_, skip = excludes["TradeRestrictions"]
	if !skip {
		if err := action.TradeRestrictions.Write(buf); err != nil {
			return 0, err
		}
	}

	// EnforcementOrdersPermitted (bool)
	_, skip = excludes["EnforcementOrdersPermitted"]
	if !skip {
		if err := read(buf, &action.EnforcementOrdersPermitted); err != nil {
			return 0, err
		}
	}

	// VoteMultiplier (uint8)
	_, skip = excludes["VoteMultiplier"]
	if !skip {
		if err := read(buf, &action.VoteMultiplier); err != nil {
			return 0, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := read(buf, &action.ReferendumProposal); err != nil {
			return 0, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := read(buf, &action.InitiativeProposal); err != nil {
			return 0, err
		}
	}

	// AssetModificationGovernance (bool)
	_, skip = excludes["AssetModificationGovernance"]
	if !skip {
		if err := read(buf, &action.AssetModificationGovernance); err != nil {
			return 0, err
		}
	}

	// TokenQty (uint64)
	_, skip = excludes["TokenQty"]
	if !skip {
		if err := read(buf, &action.TokenQty); err != nil {
			return 0, err
		}
	}

	// AssetPayload ([]byte)
	_, skip = excludes["AssetPayload"]
	if !skip {
		var err error
		action.AssetPayload, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action AssetDefinition) PayloadMessage() (PayloadMessage, error) {
	p := AssetTypeMapping(action.AssetType)
	if p == nil {
		return nil, fmt.Errorf("Undefined asset type : %s", action.AssetType)
	}

	if _, err := p.Write(action.AssetPayload); err != nil {
		return nil, err
	}

	return p, nil
}

func (action AssetDefinition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetAuthFlags:%#+v", action.AssetAuthFlags))
	vals = append(vals, fmt.Sprintf("TransfersPermitted:%#+v", action.TransfersPermitted))
	vals = append(vals, fmt.Sprintf("TradeRestrictions:%#+v", action.TradeRestrictions))
	vals = append(vals, fmt.Sprintf("EnforcementOrdersPermitted:%#+v", action.EnforcementOrdersPermitted))
	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", action.VoteMultiplier))
	vals = append(vals, fmt.Sprintf("ReferendumProposal:%#+v", action.ReferendumProposal))
	vals = append(vals, fmt.Sprintf("InitiativeProposal:%#+v", action.InitiativeProposal))
	vals = append(vals, fmt.Sprintf("AssetModificationGovernance:%#+v", action.AssetModificationGovernance))
	vals = append(vals, fmt.Sprintf("TokenQty:%v", action.TokenQty))
	vals = append(vals, fmt.Sprintf("AssetPayload:%#x", action.AssetPayload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetCreation Asset Creation Action - This action creates an Asset in
// response to the Issuer's instructions in the Definition Action.
type AssetCreation struct {
	Header                      Header    `json:"header,omitempty"`                        // Common header data for all actions
	AssetType                   string    `json:"asset_type,omitempty"`                    // eg. Share - Common
	AssetCode                   AssetCode `json:"asset_code,omitempty"`                    // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetAuthFlags              [8]byte   `json:"asset_auth_flags,omitempty"`              // Authorization Flags,  bitwise operation
	TransfersPermitted          bool      `json:"transfers_permitted,omitempty"`           // 1 = Transfers are permitted.  0 = Transfers are not permitted.
	TradeRestrictions           Polity    `json:"trade_restrictions,omitempty"`            // Asset can only be traded within the trade restrictions.  Eg. AUS - Australian residents only.  EU - European Union residents only.
	EnforcementOrdersPermitted  bool      `json:"enforcement_orders_permitted,omitempty"`  // 1 = Enforcement Orders are permitted. 0 = Enforcement Orders are not permitted.
	VoteMultiplier              uint8     `json:"vote_multiplier,omitempty"`               // Multiplies the vote by the integer. 1 token = 1 vote with a 1 for vote multipler (normal).  1 token = 3 votes with a multiplier of 3, for example.
	ReferendumProposal          bool      `json:"referendum_proposal,omitempty"`           // A Referendum is permitted for Asset-Wide Proposals (outside of smart contract scope) if also permitted by the contract. If the contract has proposals by referendum restricted, then this flag is meaningless.
	InitiativeProposal          bool      `json:"initiative_proposal,omitempty"`           // An initiative is permitted for Asset-Wide Proposals (outside of smart contract scope) if also permitted by the contract. If the contract has proposals by initiative restricted, then this flag is meaningless.
	AssetModificationGovernance bool      `json:"asset_modification_governance,omitempty"` // 1 - Contract-wide Asset Governance.  0 - Asset-wide Asset Governance.  If a referendum or initiative is used to propose a modification to a subfield controlled by the asset auth flags, then the vote will either be a contract-wide vote (all assets vote on the referendum/initiative) or an asset-wide vote (all assets vote on the referendum/initiative).  The voting system specifies the voting rules.
	TokenQty                    uint64    `json:"token_qty,omitempty"`                     // Quantity of token - 0 is valid. Fungible 'shares' of the Asset. 1 is used for non-fungible tokens.  Asset Codes become the non-fungible Asset Code and many Asset Codes can be associated with a particular Contract.
	AssetPayload                []byte    `json:"asset_payload,omitempty"`                 // Payload length is dependent on the asset type. Each asset is made up of a defined set of information pertaining to the specific asset type, and may contain fields of variable length type (nvarchar8, 16, 32)
	AssetRevision               uint32    `json:"asset _revision,omitempty"`               // Counter 0 to (2^32)-1
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetAuthFlags ([8]byte)
	_, skip = excludes["AssetAuthFlags"]
	if !skip {
		if err := write(buf, action.AssetAuthFlags); err != nil {
			return nil, err
		}
	}

	// TransfersPermitted (bool)
	_, skip = excludes["TransfersPermitted"]
	if !skip {
		if err := write(buf, action.TransfersPermitted); err != nil {
			return nil, err
		}
	}

	// TradeRestrictions (Polity)
	_, skip = excludes["TradeRestrictions"]
	if !skip {
		b, err := action.TradeRestrictions.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// EnforcementOrdersPermitted (bool)
	_, skip = excludes["EnforcementOrdersPermitted"]
	if !skip {
		if err := write(buf, action.EnforcementOrdersPermitted); err != nil {
			return nil, err
		}
	}

	// VoteMultiplier (uint8)
	_, skip = excludes["VoteMultiplier"]
	if !skip {
		if err := write(buf, action.VoteMultiplier); err != nil {
			return nil, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := write(buf, action.ReferendumProposal); err != nil {
			return nil, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := write(buf, action.InitiativeProposal); err != nil {
			return nil, err
		}
	}

	// AssetModificationGovernance (bool)
	_, skip = excludes["AssetModificationGovernance"]
	if !skip {
		if err := write(buf, action.AssetModificationGovernance); err != nil {
			return nil, err
		}
	}

	// TokenQty (uint64)
	_, skip = excludes["TokenQty"]
	if !skip {
		if err := write(buf, action.TokenQty); err != nil {
			return nil, err
		}
	}

	// AssetPayload ([]byte)
	_, skip = excludes["AssetPayload"]
	if !skip {
		if err := WriteVarBin(buf, action.AssetPayload, 16); err != nil {
			return nil, err
		}
	}

	// AssetRevision (uint32)
	_, skip = excludes["AssetRevision"]
	if !skip {
		if err := write(buf, action.AssetRevision); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetAuthFlags ([8]byte)
	_, skip = excludes["AssetAuthFlags"]
	if !skip {
		if err := read(buf, &action.AssetAuthFlags); err != nil {
			return 0, err
		}
	}

	// TransfersPermitted (bool)
	_, skip = excludes["TransfersPermitted"]
	if !skip {
		if err := read(buf, &action.TransfersPermitted); err != nil {
			return 0, err
		}
	}

	// TradeRestrictions (Polity)
	_, skip = excludes["TradeRestrictions"]
	if !skip {
		if err := action.TradeRestrictions.Write(buf); err != nil {
			return 0, err
		}
	}

	// EnforcementOrdersPermitted (bool)
	_, skip = excludes["EnforcementOrdersPermitted"]
	if !skip {
		if err := read(buf, &action.EnforcementOrdersPermitted); err != nil {
			return 0, err
		}
	}

	// VoteMultiplier (uint8)
	_, skip = excludes["VoteMultiplier"]
	if !skip {
		if err := read(buf, &action.VoteMultiplier); err != nil {
			return 0, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := read(buf, &action.ReferendumProposal); err != nil {
			return 0, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := read(buf, &action.InitiativeProposal); err != nil {
			return 0, err
		}
	}

	// AssetModificationGovernance (bool)
	_, skip = excludes["AssetModificationGovernance"]
	if !skip {
		if err := read(buf, &action.AssetModificationGovernance); err != nil {
			return 0, err
		}
	}

	// TokenQty (uint64)
	_, skip = excludes["TokenQty"]
	if !skip {
		if err := read(buf, &action.TokenQty); err != nil {
			return 0, err
		}
	}

	// AssetPayload ([]byte)
	_, skip = excludes["AssetPayload"]
	if !skip {
		var err error
		action.AssetPayload, err = ReadVarBin(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// AssetRevision (uint32)
	_, skip = excludes["AssetRevision"]
	if !skip {
		if err := read(buf, &action.AssetRevision); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action AssetCreation) PayloadMessage() (PayloadMessage, error) {
	p := AssetTypeMapping(action.AssetType)
	if p == nil {
		return nil, fmt.Errorf("Undefined asset type : %s", action.AssetType)
	}

	if _, err := p.Write(action.AssetPayload); err != nil {
		return nil, err
	}

	return p, nil
}

func (action AssetCreation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetAuthFlags:%#+v", action.AssetAuthFlags))
	vals = append(vals, fmt.Sprintf("TransfersPermitted:%#+v", action.TransfersPermitted))
	vals = append(vals, fmt.Sprintf("TradeRestrictions:%#+v", action.TradeRestrictions))
	vals = append(vals, fmt.Sprintf("EnforcementOrdersPermitted:%#+v", action.EnforcementOrdersPermitted))
	vals = append(vals, fmt.Sprintf("VoteMultiplier:%v", action.VoteMultiplier))
	vals = append(vals, fmt.Sprintf("ReferendumProposal:%#+v", action.ReferendumProposal))
	vals = append(vals, fmt.Sprintf("InitiativeProposal:%#+v", action.InitiativeProposal))
	vals = append(vals, fmt.Sprintf("AssetModificationGovernance:%#+v", action.AssetModificationGovernance))
	vals = append(vals, fmt.Sprintf("TokenQty:%v", action.TokenQty))
	vals = append(vals, fmt.Sprintf("AssetPayload:%#x", action.AssetPayload))
	vals = append(vals, fmt.Sprintf("AssetRevision:%v", action.AssetRevision))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// AssetModification Asset Modification Action - Token Dilutions, Call
// Backs/Revocations, burning etc.
type AssetModification struct {
	Header        Header      `json:"header,omitempty"`         // Common header data for all actions
	AssetType     string      `json:"asset_type,omitempty"`     // eg. Share - Common
	AssetCode     AssetCode   `json:"asset_code,omitempty"`     // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	AssetRevision uint32      `json:"asset_revision,omitempty"` // Counter. (Subfield cannot be manually changed by Asset Modification Action.  Only SC can increment by 1 with each AC action. SC will reject AM actions where the wrong asset revision has been selected.
	Modifications []Amendment `json:"modifications,omitempty"`  //
	RefTxID       TxId        `json:"ref_tx_id,omitempty"`      // Tx-ID of the associated Result action (governance) that permitted the modifications.
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AssetRevision (uint32)
	_, skip = excludes["AssetRevision"]
	if !skip {
		if err := write(buf, action.AssetRevision); err != nil {
			return nil, err
		}
	}

	// Modifications ([]Amendment)
	_, skip = excludes["Modifications"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Modifications)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Modifications {
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
	_, skip = excludes["RefTxID"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetRevision (uint32)
	_, skip = excludes["AssetRevision"]
	if !skip {
		if err := read(buf, &action.AssetRevision); err != nil {
			return 0, err
		}
	}

	// Modifications ([]Amendment)
	_, skip = excludes["Modifications"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Modifications = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Modifications = append(action.Modifications, newValue)
		}
	}

	// RefTxID (TxId)
	_, skip = excludes["RefTxID"]
	if !skip {
		if err := action.RefTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action AssetModification) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action AssetModification) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("AssetRevision:%v", action.AssetRevision))
	vals = append(vals, fmt.Sprintf("Modifications:%#+v", action.Modifications))
	vals = append(vals, fmt.Sprintf("RefTxID:%#+v", action.RefTxID))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractOffer The Contract Offer action allows the Issuer to tell the
// smart contract what they want the details (labels, data, T&C's, etc.) of
// the Contract to be on-chain in a public and immutable way. The Contract
// Offer action 'initializes' a generic smart contract that has been spun
// up by either the Smart Contract Operator or the Issuer. This on-chain
// action allows for the positive response from the smart contract with
// either a Contract Formation Action or a Rejection Action.
type ContractOffer struct {
	Header                     Header         `json:"header,omitempty"`                        // Common header data for all actions
	ContractName               string         `json:"contract_name,omitempty"`                 // Can be any unique identifying string, including human readable names for branding/vanity purposes.   [Contract identifier (instance) is the bitcoin public key hash address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address.]. Smart contracts can be branded and specialized to suit any terms and conditions.
	ContractFileType           uint8          `json:"contract_file_type,omitempty"`            // 1 - SHA-256 Hash, 2 - Markdown
	ContractFile               []byte         `json:"contract_file,omitempty"`                 // SHA-256 hash of the contract file or markdown data for contract file specific to the smart contract and relevant Assets.  Legal and technical information. (eg. pdf)
	SupportingDocsFileType     uint8          `json:"supporting_docs_file_type,omitempty"`     // 1 - 7z
	SupportingDocs             string         `json:"supporting_docs,omitempty"`               //
	GoverningLaw               string         `json:"governing_law,omitempty"`                 // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction               string         `json:"jurisdiction,omitempty"`                  // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	ContractExpiration         Timestamp      `json:"contract_expiration,omitempty"`           // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate total smart contract running costs for the entire life of the contract. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI                string         `json:"contract_uri,omitempty"`                  // Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on-chain information or even a public address (DNS).
	IssuerName                 string         `json:"issuer_name,omitempty"`                   // Length 0-255 bytes. 0 is not valid.Issuing entity (company, organization, individual).  Can be any unique identifying string, including human readable names for branding/vanity purposes.
	IssuerType                 byte           `json:"issuer_type,omitempty"`                   // P - Public Company Limited by Shares, C - Private Company Limited by Shares, I - Individual, L - Limited Partnership, U -Unlimited Partnership, T - Sole Proprietorship, S - Statutory Company, O - Non-Profit Organization, N - Nation State, G - Government Agency, U - Unit Trust, D - Discretionary Trust.  Found in 'Entities' (Specification/Resources).
	IssuerLEI                  string         `json:"issuer_lei,omitempty"`                    // Null is valid. A Legal Entity Identifier (or LEI) is an international identifier made up of a 20-character identifier that identifies distinct legal entities that engage in financial transactions. It is defined by ISO 17442.[1] Natural persons are not required to have an LEI; they’re eligible to have one issued, however, but only if they act in an independent business capacity.[2] The LEI is a global standard, designed to be non-proprietary data that is freely accessible to all.[3] As of December 2018, over 1,300,000 legal entities from more than 200 countries have now been issued with LEIs.
	IssuerLogoURL              string         `json:"issuer_logo_url,omitempty"`               // The URL of the Issuers logo.
	ContractOperatorIncluded   bool           `json:"contract_operator_included,omitempty"`    // If true, then the second input is a contract operator. If false, then all additional inputs are just funding and "includes" fields are skipped in serialization.
	ContractOperatorID         string         `json:"contract_operator_id,omitempty"`          // Length 0-255 bytes. 0 is valid. Smart Contract Operator identifier. Can be any unique identifying string, including human readable names for branding/vanity purposes. Can also be null or the Issuer.
	OperatorLEI                string         `json:"operator_lei,omitempty"`                  // Null is valid. A Legal Entity Identifier (or LEI) is an international identifier made up of a 20-character identifier that identifies distinct legal entities that engage in financial transactions. It is defined by ISO 17442.[1] Natural persons are not required to have an LEI; they’re eligible to have one issued, however, but only if they act in an independent business capacity.[2] The LEI is a global standard, designed to be non-proprietary data that is freely accessible to all.[3] As of December 2018, over 1,300,000 legal entities from more than 200 countries have now been issued with LEIs.
	ContractAuthFlags          [16]byte       `json:"contract_auth_flags,omitempty"`           // Authorization Flags aka Terms and Conditions that the smart contract can enforce.  Other terms and conditions that are out of the smart contract's control are listed in the actual Contract File.
	ActionFee                  []Fee          `json:"action_fee,omitempty"`                    // The tokens available for payment of action fees. One is required per action and must be included in the action request transaction. i.e. Bitcoin amount or an AUD backed token and amount.
	VotingSystems              []VotingSystem `json:"voting_systems,omitempty"`                // A list of voting systems.
	RestrictedQtyAssets        uint64         `json:"restricted_qty_assets,omitempty"`         // Number of Assets (non-fungible) permitted on this contract. 0 if unlimited which will display an infinity symbol in UI
	ReferendumProposal         bool           `json:"referendum_proposal,omitempty"`           // A Referendum is permitted for Proposals (outside of smart contract scope).
	InitiativeProposal         bool           `json:"initiative_proposal,omitempty"`           // An initiative is permitted for Proposals (outside of smart contract scope).
	Registries                 []Registry     `json:"registries,omitempty"`                    // A list Registries
	IssuerAddressIncluded      bool           `json:"issuer_address_included,omitempty"`       // Physical/mailing address. False means the "includes" fields are skipped in serialization.
	UnitNumber                 string         `json:"unit_number,omitempty"`                   // Issuer Address Details (eg. HQ)
	BuildingNumber             string         `json:"building_number,omitempty"`               //
	Street                     string         `json:"street,omitempty"`                        //
	SuburbCity                 string         `json:"suburb_city,omitempty"`                   //
	TerritoryStateProvinceCode string         `json:"territory_state_province_code,omitempty"` //
	CountryCode                string         `json:"country_code,omitempty"`                  //
	PostalZIPCode              string         `json:"postal_zip_code,omitempty"`               //
	EmailAddress               string         `json:"email_address,omitempty"`                 // Length 0-255 bytes. 0 is valid (no EmailAddress). Address for text-based communication: eg. email address, Bitcoin address
	PhoneNumber                string         `json:"phone_number,omitempty"`                  // Length 0-50 bytes. 0 is valid (no Phone subfield).Phone Number for Entity.
	KeyRoles                   []KeyRole      `json:"key_roles,omitempty"`                     // A list of Key Roles.
	NotableRoles               []NotableRole  `json:"notable_roles,omitempty"`                 // A list of Notable Roles.
}

// SetIssuerAddress Sets the issuer's mailing address on the contract..
func (action *ContractOffer) SetIssuerAddress(unit string, building string, street string, city string, state string, countryCode string, postalCode string) {
	action.UnitNumber = unit
	action.BuildingNumber = building
	action.Street = street
	action.SuburbCity = city
	action.TerritoryStateProvinceCode = state
	action.CountryCode = countryCode
	action.PostalZIPCode = postalCode
	action.IssuerAddressIncluded = true
}

// AddKeyRole Adds a key role to the contract.
func (action *ContractOffer) AddKeyRole(roleType uint8, name string) {
	action.KeyRoles = append(action.KeyRoles, *NewKeyRole(roleType, name))
}

// AddNotableRole Adds a notable role to the contract.
func (action *ContractOffer) AddNotableRole(roleType uint8, name string) {
	action.NotableRoles = append(action.NotableRoles, *NewNotableRole(roleType, name))
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
	excludes := make(map[string]bool)
	var skip bool

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := write(buf, action.ContractFileType); err != nil {
			return nil, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		if err := WriteVarBin(buf, action.ContractFile, 32); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		if err := WriteVarChar(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// IssuerName (string)
	_, skip = excludes["IssuerName"]
	if !skip {
		if err := WriteVarChar(buf, action.IssuerName, 8); err != nil {
			return nil, err
		}
	}

	// IssuerType (byte)
	_, skip = excludes["IssuerType"]
	if !skip {
		if err := write(buf, action.IssuerType); err != nil {
			return nil, err
		}
	}

	// IssuerLEI (string)
	_, skip = excludes["IssuerLEI"]
	if !skip {
		if err := WriteFixedChar(buf, action.IssuerLEI, 20); err != nil {
			return nil, err
		}
	}

	// IssuerLogoURL (string)
	_, skip = excludes["IssuerLogoURL"]
	if !skip {
		if err := WriteVarChar(buf, action.IssuerLogoURL, 8); err != nil {
			return nil, err
		}
	}

	// ContractOperatorIncluded (bool)
	_, skip = excludes["ContractOperatorIncluded"]
	if !skip {
		if err := write(buf, action.ContractOperatorIncluded); err != nil {
			return nil, err
		}
		if !action.ContractOperatorIncluded {
			excludes["ContractOperatorID"] = true
			excludes["OperatorLEI"] = true
		}
	}

	// ContractOperatorID (string)
	_, skip = excludes["ContractOperatorID"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractOperatorID, 8); err != nil {
			return nil, err
		}
	}

	// OperatorLEI (string)
	_, skip = excludes["OperatorLEI"]
	if !skip {
		if err := WriteFixedChar(buf, action.OperatorLEI, 20); err != nil {
			return nil, err
		}
	}

	// ContractAuthFlags ([16]byte)
	_, skip = excludes["ContractAuthFlags"]
	if !skip {
		if err := write(buf, action.ContractAuthFlags); err != nil {
			return nil, err
		}
	}

	// ActionFee ([]Fee)
	_, skip = excludes["ActionFee"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.ActionFee)), 8, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ActionFee {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// VotingSystems ([]VotingSystem)
	_, skip = excludes["VotingSystems"]
	if !skip {
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
	_, skip = excludes["RestrictedQtyAssets"]
	if !skip {
		if err := write(buf, action.RestrictedQtyAssets); err != nil {
			return nil, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := write(buf, action.ReferendumProposal); err != nil {
			return nil, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := write(buf, action.InitiativeProposal); err != nil {
			return nil, err
		}
	}

	// Registries ([]Registry)
	_, skip = excludes["Registries"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Registries)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Registries {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// IssuerAddressIncluded (bool)
	_, skip = excludes["IssuerAddressIncluded"]
	if !skip {
		if err := write(buf, action.IssuerAddressIncluded); err != nil {
			return nil, err
		}
		if !action.IssuerAddressIncluded {
			excludes["UnitNumber"] = true
			excludes["BuildingNumber"] = true
			excludes["Street"] = true
			excludes["SuburbCity"] = true
			excludes["TerritoryStateProvinceCode"] = true
			excludes["CountryCode"] = true
			excludes["PostalZIPCode"] = true
		}
	}

	// UnitNumber (string)
	_, skip = excludes["UnitNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.UnitNumber, 8); err != nil {
			return nil, err
		}
	}

	// BuildingNumber (string)
	_, skip = excludes["BuildingNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.BuildingNumber, 8); err != nil {
			return nil, err
		}
	}

	// Street (string)
	_, skip = excludes["Street"]
	if !skip {
		if err := WriteVarChar(buf, action.Street, 16); err != nil {
			return nil, err
		}
	}

	// SuburbCity (string)
	_, skip = excludes["SuburbCity"]
	if !skip {
		if err := WriteVarChar(buf, action.SuburbCity, 8); err != nil {
			return nil, err
		}
	}

	// TerritoryStateProvinceCode (string)
	_, skip = excludes["TerritoryStateProvinceCode"]
	if !skip {
		if err := WriteFixedChar(buf, action.TerritoryStateProvinceCode, 5); err != nil {
			return nil, err
		}
	}

	// CountryCode (string)
	_, skip = excludes["CountryCode"]
	if !skip {
		if err := WriteFixedChar(buf, action.CountryCode, 3); err != nil {
			return nil, err
		}
	}

	// PostalZIPCode (string)
	_, skip = excludes["PostalZIPCode"]
	if !skip {
		if err := WriteVarChar(buf, action.PostalZIPCode, 8); err != nil {
			return nil, err
		}
	}

	// EmailAddress (string)
	_, skip = excludes["EmailAddress"]
	if !skip {
		if err := WriteVarChar(buf, action.EmailAddress, 8); err != nil {
			return nil, err
		}
	}

	// PhoneNumber (string)
	_, skip = excludes["PhoneNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.PhoneNumber, 8); err != nil {
			return nil, err
		}
	}

	// KeyRoles ([]KeyRole)
	_, skip = excludes["KeyRoles"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.KeyRoles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.KeyRoles {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// NotableRoles ([]NotableRole)
	_, skip = excludes["NotableRoles"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.NotableRoles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.NotableRoles {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := read(buf, &action.ContractFileType); err != nil {
			return 0, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		var err error
		action.ContractFile, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		var err error
		action.SupportingDocs, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// IssuerName (string)
	_, skip = excludes["IssuerName"]
	if !skip {
		var err error
		action.IssuerName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// IssuerType (byte)
	_, skip = excludes["IssuerType"]
	if !skip {
		if err := read(buf, &action.IssuerType); err != nil {
			return 0, err
		}
	}

	// IssuerLEI (string)
	_, skip = excludes["IssuerLEI"]
	if !skip {
		var err error
		action.IssuerLEI, err = ReadFixedChar(buf, 20)
		if err != nil {
			return 0, err
		}
	}

	// IssuerLogoURL (string)
	_, skip = excludes["IssuerLogoURL"]
	if !skip {
		var err error
		action.IssuerLogoURL, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractOperatorIncluded (bool)
	_, skip = excludes["ContractOperatorIncluded"]
	if !skip {
		if err := read(buf, &action.ContractOperatorIncluded); err != nil {
			return 0, err
		}
		if !action.ContractOperatorIncluded {
			excludes["ContractOperatorID"] = true
			excludes["OperatorLEI"] = true
		}
	}

	// ContractOperatorID (string)
	_, skip = excludes["ContractOperatorID"]
	if !skip {
		var err error
		action.ContractOperatorID, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// OperatorLEI (string)
	_, skip = excludes["OperatorLEI"]
	if !skip {
		var err error
		action.OperatorLEI, err = ReadFixedChar(buf, 20)
		if err != nil {
			return 0, err
		}
	}

	// ContractAuthFlags ([16]byte)
	_, skip = excludes["ContractAuthFlags"]
	if !skip {
		if err := read(buf, &action.ContractAuthFlags); err != nil {
			return 0, err
		}
	}

	// ActionFee ([]Fee)
	_, skip = excludes["ActionFee"]
	if !skip {
		size, err := ReadVariableSize(buf, 8, 8)
		if err != nil {
			return 0, err
		}
		action.ActionFee = make([]Fee, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Fee
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ActionFee = append(action.ActionFee, newValue)
		}
	}

	// VotingSystems ([]VotingSystem)
	_, skip = excludes["VotingSystems"]
	if !skip {
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
	_, skip = excludes["RestrictedQtyAssets"]
	if !skip {
		if err := read(buf, &action.RestrictedQtyAssets); err != nil {
			return 0, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := read(buf, &action.ReferendumProposal); err != nil {
			return 0, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := read(buf, &action.InitiativeProposal); err != nil {
			return 0, err
		}
	}

	// Registries ([]Registry)
	_, skip = excludes["Registries"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Registries = make([]Registry, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Registry
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Registries = append(action.Registries, newValue)
		}
	}

	// IssuerAddressIncluded (bool)
	_, skip = excludes["IssuerAddressIncluded"]
	if !skip {
		if err := read(buf, &action.IssuerAddressIncluded); err != nil {
			return 0, err
		}
		if !action.IssuerAddressIncluded {
			excludes["UnitNumber"] = true
			excludes["BuildingNumber"] = true
			excludes["Street"] = true
			excludes["SuburbCity"] = true
			excludes["TerritoryStateProvinceCode"] = true
			excludes["CountryCode"] = true
			excludes["PostalZIPCode"] = true
		}
	}

	// UnitNumber (string)
	_, skip = excludes["UnitNumber"]
	if !skip {
		var err error
		action.UnitNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// BuildingNumber (string)
	_, skip = excludes["BuildingNumber"]
	if !skip {
		var err error
		action.BuildingNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Street (string)
	_, skip = excludes["Street"]
	if !skip {
		var err error
		action.Street, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// SuburbCity (string)
	_, skip = excludes["SuburbCity"]
	if !skip {
		var err error
		action.SuburbCity, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// TerritoryStateProvinceCode (string)
	_, skip = excludes["TerritoryStateProvinceCode"]
	if !skip {
		var err error
		action.TerritoryStateProvinceCode, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// CountryCode (string)
	_, skip = excludes["CountryCode"]
	if !skip {
		var err error
		action.CountryCode, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// PostalZIPCode (string)
	_, skip = excludes["PostalZIPCode"]
	if !skip {
		var err error
		action.PostalZIPCode, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// EmailAddress (string)
	_, skip = excludes["EmailAddress"]
	if !skip {
		var err error
		action.EmailAddress, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// PhoneNumber (string)
	_, skip = excludes["PhoneNumber"]
	if !skip {
		var err error
		action.PhoneNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// KeyRoles ([]KeyRole)
	_, skip = excludes["KeyRoles"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.KeyRoles = make([]KeyRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue KeyRole
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.KeyRoles = append(action.KeyRoles, newValue)
		}
	}

	// NotableRoles ([]NotableRole)
	_, skip = excludes["NotableRoles"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.NotableRoles = make([]NotableRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue NotableRole
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.NotableRoles = append(action.NotableRoles, newValue)
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action ContractOffer) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action ContractOffer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("ContractFileType:%v", action.ContractFileType))
	vals = append(vals, fmt.Sprintf("ContractFile:%#x", action.ContractFile))
	vals = append(vals, fmt.Sprintf("SupportingDocsFileType:%v", action.SupportingDocsFileType))
	vals = append(vals, fmt.Sprintf("SupportingDocs:%#+v", action.SupportingDocs))
	vals = append(vals, fmt.Sprintf("GoverningLaw:%#+v", action.GoverningLaw))
	vals = append(vals, fmt.Sprintf("Jurisdiction:%#+v", action.Jurisdiction))
	vals = append(vals, fmt.Sprintf("ContractExpiration:%#+v", action.ContractExpiration))
	vals = append(vals, fmt.Sprintf("ContractURI:%#+v", action.ContractURI))
	vals = append(vals, fmt.Sprintf("IssuerName:%#+v", action.IssuerName))
	vals = append(vals, fmt.Sprintf("IssuerType:%#+v", action.IssuerType))
	vals = append(vals, fmt.Sprintf("IssuerLEI:%#+v", action.IssuerLEI))
	vals = append(vals, fmt.Sprintf("IssuerLogoURL:%#+v", action.IssuerLogoURL))
	vals = append(vals, fmt.Sprintf("ContractOperatorIncluded:%#+v", action.ContractOperatorIncluded))
	vals = append(vals, fmt.Sprintf("ContractOperatorID:%#+v", action.ContractOperatorID))
	vals = append(vals, fmt.Sprintf("OperatorLEI:%#+v", action.OperatorLEI))
	vals = append(vals, fmt.Sprintf("ContractAuthFlags:%#+v", action.ContractAuthFlags))
	vals = append(vals, fmt.Sprintf("ActionFee:%#+v", action.ActionFee))
	vals = append(vals, fmt.Sprintf("VotingSystems:%#+v", action.VotingSystems))
	vals = append(vals, fmt.Sprintf("RestrictedQtyAssets:%v", action.RestrictedQtyAssets))
	vals = append(vals, fmt.Sprintf("ReferendumProposal:%#+v", action.ReferendumProposal))
	vals = append(vals, fmt.Sprintf("InitiativeProposal:%#+v", action.InitiativeProposal))
	vals = append(vals, fmt.Sprintf("Registries:%#+v", action.Registries))
	vals = append(vals, fmt.Sprintf("IssuerAddressIncluded:%#+v", action.IssuerAddressIncluded))
	vals = append(vals, fmt.Sprintf("UnitNumber:%#+v", action.UnitNumber))
	vals = append(vals, fmt.Sprintf("BuildingNumber:%#+v", action.BuildingNumber))
	vals = append(vals, fmt.Sprintf("Street:%#+v", action.Street))
	vals = append(vals, fmt.Sprintf("SuburbCity:%#+v", action.SuburbCity))
	vals = append(vals, fmt.Sprintf("TerritoryStateProvinceCode:%#+v", action.TerritoryStateProvinceCode))
	vals = append(vals, fmt.Sprintf("CountryCode:%#+v", action.CountryCode))
	vals = append(vals, fmt.Sprintf("PostalZIPCode:%#+v", action.PostalZIPCode))
	vals = append(vals, fmt.Sprintf("EmailAddress:%#+v", action.EmailAddress))
	vals = append(vals, fmt.Sprintf("PhoneNumber:%#+v", action.PhoneNumber))
	vals = append(vals, fmt.Sprintf("KeyRoles:%#+v", action.KeyRoles))
	vals = append(vals, fmt.Sprintf("NotableRoles:%#+v", action.NotableRoles))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractFormation This txn is created by the Contract (smart
// contract/off-chain agent/token contract) upon receipt of a valid
// Contract Offer Action from the issuer. The Smart Contract will execute
// on a server controlled by the Issuer. or a Smart Contract Operator on
// their behalf.
type ContractFormation struct {
	Header                     Header         `json:"header,omitempty"`                        // Common header data for all actions
	ContractName               string         `json:"contract_name,omitempty"`                 // Can be any unique identifying string, including human readable names for branding/vanity purposes.   [Contract identifier (instance) is the bitcoin public key hash address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address.]. Smart contracts can be branded and specialized to suit any terms and conditions.
	ContractFileType           uint8          `json:"contract_file_type,omitempty"`            // 1 - SHA-256 Hash, 2 - Markdown file
	ContractFile               []byte         `json:"contract_file,omitempty"`                 // SHA-256 hash of the contract file or markdown data for contract file specific to the smart contract and relevant Assets.  Legal and technical information. (eg. pdf)
	SupportingDocsFileType     uint8          `json:"supporting_docs_file_type,omitempty"`     // 1 - 7z
	SupportingDocs             string         `json:"supporting_docs,omitempty"`               //
	GoverningLaw               string         `json:"governing_law,omitempty"`                 // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction               string         `json:"jurisdiction,omitempty"`                  // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	ContractExpiration         Timestamp      `json:"contract_expiration,omitempty"`           // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate smart contract running costs. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI                string         `json:"contract_uri,omitempty"`                  // Length 0-255 bytes.  0 is valid. Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on chain information or even a public address (DNS).
	IssuerName                 string         `json:"issuer_name,omitempty"`                   // Length 0-255 bytes. 0 is not valid. Issuing entity (company, organization, individual).  Can be any unique identifying string, including human readable names for branding/vanity purposes.
	IssuerType                 byte           `json:"issuer_type,omitempty"`                   // P - Public Company Limited by Shares, C - Private Company Limited by Shares, I - Individual, L - Limited Partnership, U -Unlimited Partnership, T - Sole Proprietorship, S - Statutory Company, O - Non-Profit Organization, N - Nation State, G - Government Agency, U - Unit Trust, D - Discretionary Trust.  Found in 'Entities' (Specification/Resources).
	IssuerLEI                  string         `json:"issuer_lei,omitempty"`                    // Null is valid. A Legal Entity Identifier (or LEI) is an international identifier made up of a 20-character identifier that identifies distinct legal entities that engage in financial transactions. It is defined by ISO 17442.[1] Natural persons are not required to have an LEI; they’re eligible to have one issued, however, but only if they act in an independent business capacity.[2] The LEI is a global standard, designed to be non-proprietary data that is freely accessible to all.[3] As of December 2018, over 1,300,000 legal entities from more than 200 countries have now been issued with LEIs.
	IssuerLogoURL              string         `json:"issuer_logo_url,omitempty"`               // The URL of the Issuers logo.
	ContractOperatorIncluded   bool           `json:"contract_operator_included,omitempty"`    // If true, then the second input is a contract operator. If false, then all additional inputs are just funding and "includes" fields are skipped in serialization.
	ContractOperatorID         string         `json:"contract_operator_id,omitempty"`          // Length 0-255 bytes. 0 is valid. Smart Contract Operator identifier. Can be any unique identifying string, including human readable names for branding/vanity purposes. Can also be null or the Issuer.
	OperatorLEI                string         `json:"operator_lei,omitempty"`                  // Null is valid. A Legal Entity Identifier (or LEI) is an international identifier made up of a 20-character identifier that identifies distinct legal entities that engage in financial transactions. It is defined by ISO 17442.[1] Natural persons are not required to have an LEI; they’re eligible to have one issued, however, but only if they act in an independent business capacity.[2] The LEI is a global standard, designed to be non-proprietary data that is freely accessible to all.[3] As of December 2018, over 1,300,000 legal entities from more than 200 countries have now been issued with LEIs.
	ContractAuthFlags          [16]byte       `json:"contract_auth_flags,omitempty"`           // Authorization Flags aka Terms and Conditions that the smart contract can enforce.  Other terms and conditions that are out of the smart contract's control are listed in the actual Contract File.
	ActionFee                  []Fee          `json:"action_fee,omitempty"`                    // The tokens available for payment of action fees. One is required per action and must be included in the action request transaction. i.e. Bitcoin amount or an AUD backed token and amount.
	VotingSystems              []VotingSystem `json:"voting_systems,omitempty"`                // A list voting systems.
	RestrictedQtyAssets        uint64         `json:"restricted_qty_assets,omitempty"`         // Number of Assets (non-fungible) permitted on this contract. 0 if unlimited which will display an infinity symbol in UI
	ReferendumProposal         bool           `json:"referendum_proposal,omitempty"`           // A Referendum is permitted for Contract-Wide Proposals (outside of smart contract scope).
	InitiativeProposal         bool           `json:"initiative_proposal,omitempty"`           // An initiative is permitted for Contract-Wide Proposals (outside of smart contract scope).
	Registries                 []Registry     `json:"registries,omitempty"`                    // A list Registries
	IssuerAddressIncluded      bool           `json:"issuer_address_included,omitempty"`       // Physical/mailing address. False means the "includes" fields are skipped in serialization.
	UnitNumber                 string         `json:"unit_number,omitempty"`                   // Issuer Address Details (eg. HQ)
	BuildingNumber             string         `json:"building_number,omitempty"`               //
	Street                     string         `json:"street,omitempty"`                        //
	SuburbCity                 string         `json:"suburb_city,omitempty"`                   //
	TerritoryStateProvinceCode string         `json:"territory_state_province_code,omitempty"` //
	CountryCode                string         `json:"country_code,omitempty"`                  //
	PostalZIPCode              string         `json:"postal_zip_code,omitempty"`               //
	EmailAddress               string         `json:"email_address,omitempty"`                 // Address for text-based communication: eg. email address, Bitcoin address
	PhoneNumber                string         `json:"phone_number,omitempty"`                  // Phone Number for Entity. Max acceptable size: 50.
	KeyRoles                   []KeyRole      `json:"key_roles,omitempty"`                     // A list of Key Roles.
	NotableRoles               []NotableRole  `json:"notable_roles,omitempty"`                 // A list of Notable Roles.
	ContractRevision           uint32         `json:"contract_revision,omitempty"`             // Counter. Cannot be manually changed by issuer.  Can only be incremented by 1 by SC when CF action is published.
	Timestamp                  Timestamp      `json:"timestamp,omitempty"`                     // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := write(buf, action.ContractFileType); err != nil {
			return nil, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		if err := WriteVarBin(buf, action.ContractFile, 32); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		if err := WriteVarChar(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// IssuerName (string)
	_, skip = excludes["IssuerName"]
	if !skip {
		if err := WriteVarChar(buf, action.IssuerName, 8); err != nil {
			return nil, err
		}
	}

	// IssuerType (byte)
	_, skip = excludes["IssuerType"]
	if !skip {
		if err := write(buf, action.IssuerType); err != nil {
			return nil, err
		}
	}

	// IssuerLEI (string)
	_, skip = excludes["IssuerLEI"]
	if !skip {
		if err := WriteFixedChar(buf, action.IssuerLEI, 20); err != nil {
			return nil, err
		}
	}

	// IssuerLogoURL (string)
	_, skip = excludes["IssuerLogoURL"]
	if !skip {
		if err := WriteVarChar(buf, action.IssuerLogoURL, 8); err != nil {
			return nil, err
		}
	}

	// ContractOperatorIncluded (bool)
	_, skip = excludes["ContractOperatorIncluded"]
	if !skip {
		if err := write(buf, action.ContractOperatorIncluded); err != nil {
			return nil, err
		}
		if !action.ContractOperatorIncluded {
			excludes["ContractOperatorID"] = true
			excludes["OperatorLEI"] = true
		}
	}

	// ContractOperatorID (string)
	_, skip = excludes["ContractOperatorID"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractOperatorID, 8); err != nil {
			return nil, err
		}
	}

	// OperatorLEI (string)
	_, skip = excludes["OperatorLEI"]
	if !skip {
		if err := WriteFixedChar(buf, action.OperatorLEI, 20); err != nil {
			return nil, err
		}
	}

	// ContractAuthFlags ([16]byte)
	_, skip = excludes["ContractAuthFlags"]
	if !skip {
		if err := write(buf, action.ContractAuthFlags); err != nil {
			return nil, err
		}
	}

	// ActionFee ([]Fee)
	_, skip = excludes["ActionFee"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.ActionFee)), 8, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ActionFee {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// VotingSystems ([]VotingSystem)
	_, skip = excludes["VotingSystems"]
	if !skip {
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
	_, skip = excludes["RestrictedQtyAssets"]
	if !skip {
		if err := write(buf, action.RestrictedQtyAssets); err != nil {
			return nil, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := write(buf, action.ReferendumProposal); err != nil {
			return nil, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := write(buf, action.InitiativeProposal); err != nil {
			return nil, err
		}
	}

	// Registries ([]Registry)
	_, skip = excludes["Registries"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Registries)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Registries {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// IssuerAddressIncluded (bool)
	_, skip = excludes["IssuerAddressIncluded"]
	if !skip {
		if err := write(buf, action.IssuerAddressIncluded); err != nil {
			return nil, err
		}
		if !action.IssuerAddressIncluded {
			excludes["UnitNumber"] = true
			excludes["BuildingNumber"] = true
			excludes["Street"] = true
			excludes["SuburbCity"] = true
			excludes["TerritoryStateProvinceCode"] = true
			excludes["CountryCode"] = true
			excludes["PostalZIPCode"] = true
		}
	}

	// UnitNumber (string)
	_, skip = excludes["UnitNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.UnitNumber, 8); err != nil {
			return nil, err
		}
	}

	// BuildingNumber (string)
	_, skip = excludes["BuildingNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.BuildingNumber, 8); err != nil {
			return nil, err
		}
	}

	// Street (string)
	_, skip = excludes["Street"]
	if !skip {
		if err := WriteVarChar(buf, action.Street, 16); err != nil {
			return nil, err
		}
	}

	// SuburbCity (string)
	_, skip = excludes["SuburbCity"]
	if !skip {
		if err := WriteVarChar(buf, action.SuburbCity, 8); err != nil {
			return nil, err
		}
	}

	// TerritoryStateProvinceCode (string)
	_, skip = excludes["TerritoryStateProvinceCode"]
	if !skip {
		if err := WriteFixedChar(buf, action.TerritoryStateProvinceCode, 5); err != nil {
			return nil, err
		}
	}

	// CountryCode (string)
	_, skip = excludes["CountryCode"]
	if !skip {
		if err := WriteFixedChar(buf, action.CountryCode, 3); err != nil {
			return nil, err
		}
	}

	// PostalZIPCode (string)
	_, skip = excludes["PostalZIPCode"]
	if !skip {
		if err := WriteVarChar(buf, action.PostalZIPCode, 8); err != nil {
			return nil, err
		}
	}

	// EmailAddress (string)
	_, skip = excludes["EmailAddress"]
	if !skip {
		if err := WriteVarChar(buf, action.EmailAddress, 8); err != nil {
			return nil, err
		}
	}

	// PhoneNumber (string)
	_, skip = excludes["PhoneNumber"]
	if !skip {
		if err := WriteVarChar(buf, action.PhoneNumber, 8); err != nil {
			return nil, err
		}
	}

	// KeyRoles ([]KeyRole)
	_, skip = excludes["KeyRoles"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.KeyRoles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.KeyRoles {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// NotableRoles ([]NotableRole)
	_, skip = excludes["NotableRoles"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.NotableRoles)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.NotableRoles {
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
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := read(buf, &action.ContractFileType); err != nil {
			return 0, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		var err error
		action.ContractFile, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		var err error
		action.SupportingDocs, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// IssuerName (string)
	_, skip = excludes["IssuerName"]
	if !skip {
		var err error
		action.IssuerName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// IssuerType (byte)
	_, skip = excludes["IssuerType"]
	if !skip {
		if err := read(buf, &action.IssuerType); err != nil {
			return 0, err
		}
	}

	// IssuerLEI (string)
	_, skip = excludes["IssuerLEI"]
	if !skip {
		var err error
		action.IssuerLEI, err = ReadFixedChar(buf, 20)
		if err != nil {
			return 0, err
		}
	}

	// IssuerLogoURL (string)
	_, skip = excludes["IssuerLogoURL"]
	if !skip {
		var err error
		action.IssuerLogoURL, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractOperatorIncluded (bool)
	_, skip = excludes["ContractOperatorIncluded"]
	if !skip {
		if err := read(buf, &action.ContractOperatorIncluded); err != nil {
			return 0, err
		}
		if !action.ContractOperatorIncluded {
			excludes["ContractOperatorID"] = true
			excludes["OperatorLEI"] = true
		}
	}

	// ContractOperatorID (string)
	_, skip = excludes["ContractOperatorID"]
	if !skip {
		var err error
		action.ContractOperatorID, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// OperatorLEI (string)
	_, skip = excludes["OperatorLEI"]
	if !skip {
		var err error
		action.OperatorLEI, err = ReadFixedChar(buf, 20)
		if err != nil {
			return 0, err
		}
	}

	// ContractAuthFlags ([16]byte)
	_, skip = excludes["ContractAuthFlags"]
	if !skip {
		if err := read(buf, &action.ContractAuthFlags); err != nil {
			return 0, err
		}
	}

	// ActionFee ([]Fee)
	_, skip = excludes["ActionFee"]
	if !skip {
		size, err := ReadVariableSize(buf, 8, 8)
		if err != nil {
			return 0, err
		}
		action.ActionFee = make([]Fee, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Fee
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ActionFee = append(action.ActionFee, newValue)
		}
	}

	// VotingSystems ([]VotingSystem)
	_, skip = excludes["VotingSystems"]
	if !skip {
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
	_, skip = excludes["RestrictedQtyAssets"]
	if !skip {
		if err := read(buf, &action.RestrictedQtyAssets); err != nil {
			return 0, err
		}
	}

	// ReferendumProposal (bool)
	_, skip = excludes["ReferendumProposal"]
	if !skip {
		if err := read(buf, &action.ReferendumProposal); err != nil {
			return 0, err
		}
	}

	// InitiativeProposal (bool)
	_, skip = excludes["InitiativeProposal"]
	if !skip {
		if err := read(buf, &action.InitiativeProposal); err != nil {
			return 0, err
		}
	}

	// Registries ([]Registry)
	_, skip = excludes["Registries"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.Registries = make([]Registry, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Registry
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Registries = append(action.Registries, newValue)
		}
	}

	// IssuerAddressIncluded (bool)
	_, skip = excludes["IssuerAddressIncluded"]
	if !skip {
		if err := read(buf, &action.IssuerAddressIncluded); err != nil {
			return 0, err
		}
		if !action.IssuerAddressIncluded {
			excludes["UnitNumber"] = true
			excludes["BuildingNumber"] = true
			excludes["Street"] = true
			excludes["SuburbCity"] = true
			excludes["TerritoryStateProvinceCode"] = true
			excludes["CountryCode"] = true
			excludes["PostalZIPCode"] = true
		}
	}

	// UnitNumber (string)
	_, skip = excludes["UnitNumber"]
	if !skip {
		var err error
		action.UnitNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// BuildingNumber (string)
	_, skip = excludes["BuildingNumber"]
	if !skip {
		var err error
		action.BuildingNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Street (string)
	_, skip = excludes["Street"]
	if !skip {
		var err error
		action.Street, err = ReadVarChar(buf, 16)
		if err != nil {
			return 0, err
		}
	}

	// SuburbCity (string)
	_, skip = excludes["SuburbCity"]
	if !skip {
		var err error
		action.SuburbCity, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// TerritoryStateProvinceCode (string)
	_, skip = excludes["TerritoryStateProvinceCode"]
	if !skip {
		var err error
		action.TerritoryStateProvinceCode, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// CountryCode (string)
	_, skip = excludes["CountryCode"]
	if !skip {
		var err error
		action.CountryCode, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// PostalZIPCode (string)
	_, skip = excludes["PostalZIPCode"]
	if !skip {
		var err error
		action.PostalZIPCode, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// EmailAddress (string)
	_, skip = excludes["EmailAddress"]
	if !skip {
		var err error
		action.EmailAddress, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// PhoneNumber (string)
	_, skip = excludes["PhoneNumber"]
	if !skip {
		var err error
		action.PhoneNumber, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// KeyRoles ([]KeyRole)
	_, skip = excludes["KeyRoles"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.KeyRoles = make([]KeyRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue KeyRole
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.KeyRoles = append(action.KeyRoles, newValue)
		}
	}

	// NotableRoles ([]NotableRole)
	_, skip = excludes["NotableRoles"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.NotableRoles = make([]NotableRole, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue NotableRole
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.NotableRoles = append(action.NotableRoles, newValue)
		}
	}

	// ContractRevision (uint32)
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action ContractFormation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action ContractFormation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("ContractFileType:%v", action.ContractFileType))
	vals = append(vals, fmt.Sprintf("ContractFile:%#x", action.ContractFile))
	vals = append(vals, fmt.Sprintf("SupportingDocsFileType:%v", action.SupportingDocsFileType))
	vals = append(vals, fmt.Sprintf("SupportingDocs:%#+v", action.SupportingDocs))
	vals = append(vals, fmt.Sprintf("GoverningLaw:%#+v", action.GoverningLaw))
	vals = append(vals, fmt.Sprintf("Jurisdiction:%#+v", action.Jurisdiction))
	vals = append(vals, fmt.Sprintf("ContractExpiration:%#+v", action.ContractExpiration))
	vals = append(vals, fmt.Sprintf("ContractURI:%#+v", action.ContractURI))
	vals = append(vals, fmt.Sprintf("IssuerName:%#+v", action.IssuerName))
	vals = append(vals, fmt.Sprintf("IssuerType:%#+v", action.IssuerType))
	vals = append(vals, fmt.Sprintf("IssuerLEI:%#+v", action.IssuerLEI))
	vals = append(vals, fmt.Sprintf("IssuerLogoURL:%#+v", action.IssuerLogoURL))
	vals = append(vals, fmt.Sprintf("ContractOperatorIncluded:%#+v", action.ContractOperatorIncluded))
	vals = append(vals, fmt.Sprintf("ContractOperatorID:%#+v", action.ContractOperatorID))
	vals = append(vals, fmt.Sprintf("OperatorLEI:%#+v", action.OperatorLEI))
	vals = append(vals, fmt.Sprintf("ContractAuthFlags:%#+v", action.ContractAuthFlags))
	vals = append(vals, fmt.Sprintf("ActionFee:%#+v", action.ActionFee))
	vals = append(vals, fmt.Sprintf("VotingSystems:%#+v", action.VotingSystems))
	vals = append(vals, fmt.Sprintf("RestrictedQtyAssets:%v", action.RestrictedQtyAssets))
	vals = append(vals, fmt.Sprintf("ReferendumProposal:%#+v", action.ReferendumProposal))
	vals = append(vals, fmt.Sprintf("InitiativeProposal:%#+v", action.InitiativeProposal))
	vals = append(vals, fmt.Sprintf("Registries:%#+v", action.Registries))
	vals = append(vals, fmt.Sprintf("IssuerAddressIncluded:%#+v", action.IssuerAddressIncluded))
	vals = append(vals, fmt.Sprintf("UnitNumber:%#+v", action.UnitNumber))
	vals = append(vals, fmt.Sprintf("BuildingNumber:%#+v", action.BuildingNumber))
	vals = append(vals, fmt.Sprintf("Street:%#+v", action.Street))
	vals = append(vals, fmt.Sprintf("SuburbCity:%#+v", action.SuburbCity))
	vals = append(vals, fmt.Sprintf("TerritoryStateProvinceCode:%#+v", action.TerritoryStateProvinceCode))
	vals = append(vals, fmt.Sprintf("CountryCode:%#+v", action.CountryCode))
	vals = append(vals, fmt.Sprintf("PostalZIPCode:%#+v", action.PostalZIPCode))
	vals = append(vals, fmt.Sprintf("EmailAddress:%#+v", action.EmailAddress))
	vals = append(vals, fmt.Sprintf("PhoneNumber:%#+v", action.PhoneNumber))
	vals = append(vals, fmt.Sprintf("KeyRoles:%#+v", action.KeyRoles))
	vals = append(vals, fmt.Sprintf("NotableRoles:%#+v", action.NotableRoles))
	vals = append(vals, fmt.Sprintf("ContractRevision:%v", action.ContractRevision))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// ContractAmendment Contract Amendment Action - the issuer can initiate an
// amendment to the contract establishment metadata. The ability to make an
// amendment to the contract is restricted by the Authorization Flag set on
// the current revision of Contract Formation action.
type ContractAmendment struct {
	Header                Header      `json:"header,omitempty"`                  // Common header data for all actions
	ChangeIssuerAddress   bool        `json:"change_issuer_address,omitempty"`   // 1 - Yes, 0 - No.  Used to change the issuer address.  The new issuer address must be in the input[1] position.
	ChangeOperatorAddress bool        `json:"change_operator_address,omitempty"` // 1 - Yes, 0 - No.  Used to change the smart contract operator address.  The new operator address must be in the input[1] position.
	ContractRevision      uint32      `json:"contract_revision,omitempty"`       // Counter 0 to (2^32)-1
	Amendments            []Amendment `json:"amendments,omitempty"`              //
	RefTxID               TxId        `json:"ref_tx_id,omitempty"`               // Tx-ID of the associated Result action (governance) that permitted the modifications.
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
	excludes := make(map[string]bool)
	var skip bool

	// ChangeIssuerAddress (bool)
	_, skip = excludes["ChangeIssuerAddress"]
	if !skip {
		if err := write(buf, action.ChangeIssuerAddress); err != nil {
			return nil, err
		}
	}

	// ChangeOperatorAddress (bool)
	_, skip = excludes["ChangeOperatorAddress"]
	if !skip {
		if err := write(buf, action.ChangeOperatorAddress); err != nil {
			return nil, err
		}
	}

	// ContractRevision (uint32)
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// Amendments ([]Amendment)
	_, skip = excludes["Amendments"]
	if !skip {
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
	_, skip = excludes["RefTxID"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// ChangeIssuerAddress (bool)
	_, skip = excludes["ChangeIssuerAddress"]
	if !skip {
		if err := read(buf, &action.ChangeIssuerAddress); err != nil {
			return 0, err
		}
	}

	// ChangeOperatorAddress (bool)
	_, skip = excludes["ChangeOperatorAddress"]
	if !skip {
		if err := read(buf, &action.ChangeOperatorAddress); err != nil {
			return 0, err
		}
	}

	// ContractRevision (uint32)
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// Amendments ([]Amendment)
	_, skip = excludes["Amendments"]
	if !skip {
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
	_, skip = excludes["RefTxID"]
	if !skip {
		if err := action.RefTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action ContractAmendment) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action ContractAmendment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("ChangeIssuerAddress:%#+v", action.ChangeIssuerAddress))
	vals = append(vals, fmt.Sprintf("ChangeOperatorAddress:%#+v", action.ChangeOperatorAddress))
	vals = append(vals, fmt.Sprintf("ContractRevision:%v", action.ContractRevision))
	vals = append(vals, fmt.Sprintf("Amendments:%#+v", action.Amendments))
	vals = append(vals, fmt.Sprintf("RefTxID:%#+v", action.RefTxID))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// StaticContractFormation Static Contract Formation Action
type StaticContractFormation struct {
	Header                 Header       `json:"header,omitempty"`                    // Common header data for all actions
	ContractName           string       `json:"contract_name,omitempty"`             // Length 0-255 bytes. Can be any unique identifying string, including human readable names for branding/vanity purposes.   [Contract identifier (instance) is the bitcoin public address. If the Public Address is lost, then the issuer will have to reissue the entire contract, Asset definition and tokens with the new public address.]. Smart contracts can be branded and specialized to suit any terms and conditions.
	ContractType           string       `json:"contract_type,omitempty"`             //
	ContractCode           ContractCode `json:"contract_code,omitempty"`             // 32 randomly generated bytes.  Each Contract Code should be unique.  The Contract ID will be human facing and will be the Contract Code, with a checksum, encoded in base58 and prefixed by 'CON'. Contract ID = CON + base58(ContractCode + checksum).  Eg. Contract ID = 'CON18RDoKK7Ed5zid2FkKVy7q3rULr4tgfjr4'
	ContractFileType       uint8        `json:"contract_file_type,omitempty"`        // 1 - SHA-256 Hash, 2 - Markdown file
	ContractFile           []byte       `json:"contract_file,omitempty"`             // SHA-256 hash of the contract file or markdown data for contract file specific to the smart contract and relevant Assets.  Legal and technical information. (eg. pdf)
	SupportingDocsFileType uint8        `json:"supporting_docs_file_type,omitempty"` // 1 - 7z
	SupportingDocs         string       `json:"supporting_docs,omitempty"`           //
	ContractRevision       uint32       `json:"contract_revision,omitempty"`         // Counter 0 to (2^32)-1
	GoverningLaw           string       `json:"governing_law,omitempty"`             // 5 Letter Code to Identify which governing law the contract will adhere to.  Disputes are to be settled by this law in the jurisdiction specified below. Private dispute resolution organizations can be used as well.  A custom code just needs to be defined.
	Jurisdiction           string       `json:"jurisdiction,omitempty"`              // Legal proceedings/arbitration will take place using the specified Governing Law in this location.
	EffectiveDate          Timestamp    `json:"effective_date,omitempty"`            // Start date of the contract.
	ContractExpiration     Timestamp    `json:"contract_expiration,omitempty"`       // All actions related to the contract will cease to work after this timestamp. The smart contract will stop running.  This will allow many token use cases to be able to calculate smart contract running costs. Eg. an issuer is creating tickets for an event on the 5th of June 2018.  The smart contract will facilitate exchange and send transactions up until the 6th of June.  Wallets can use this to forget tokens that are no longer valid - or at least store them in an 'Expired' folder.
	ContractURI            string       `json:"contract_uri,omitempty"`              // Length 0-255 bytes. Points to an information page that also has a copy of the Contract.  Anyone can go to the website to have a look at the price/token, information about the Issuer (company), information about the Asset, legal information, etc.  There will also be a way for Token Owners to vote on this page and contact details with the Issuer/tokenized companies. Could be a IPv6/IPv4, an IPFS address (hash) or txn-id for on chain information or even a public address (DNS).
	PrevRevTxID            TxId         `json:"prev_rev_tx_id,omitempty"`            // The Tx-ID of the previous contract revision.
	Entities               []Entity     `json:"entities,omitempty"`                  //
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
	excludes := make(map[string]bool)
	var skip bool

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractName, 8); err != nil {
			return nil, err
		}
	}

	// ContractType (string)
	_, skip = excludes["ContractType"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractType, 8); err != nil {
			return nil, err
		}
	}

	// ContractCode (ContractCode)
	_, skip = excludes["ContractCode"]
	if !skip {
		b, err := action.ContractCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := write(buf, action.ContractFileType); err != nil {
			return nil, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		if err := WriteVarBin(buf, action.ContractFile, 32); err != nil {
			return nil, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := write(buf, action.SupportingDocsFileType); err != nil {
			return nil, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		if err := WriteVarChar(buf, action.SupportingDocs, 32); err != nil {
			return nil, err
		}
	}

	// ContractRevision (uint32)
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := write(buf, action.ContractRevision); err != nil {
			return nil, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		if err := WriteFixedChar(buf, action.GoverningLaw, 5); err != nil {
			return nil, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		if err := WriteFixedChar(buf, action.Jurisdiction, 5); err != nil {
			return nil, err
		}
	}

	// EffectiveDate (Timestamp)
	_, skip = excludes["EffectiveDate"]
	if !skip {
		b, err := action.EffectiveDate.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		b, err := action.ContractExpiration.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		if err := WriteVarChar(buf, action.ContractURI, 8); err != nil {
			return nil, err
		}
	}

	// PrevRevTxID (TxId)
	_, skip = excludes["PrevRevTxID"]
	if !skip {
		b, err := action.PrevRevTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Entities ([]Entity)
	_, skip = excludes["Entities"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractName (string)
	_, skip = excludes["ContractName"]
	if !skip {
		var err error
		action.ContractName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractType (string)
	_, skip = excludes["ContractType"]
	if !skip {
		var err error
		action.ContractType, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// ContractCode (ContractCode)
	_, skip = excludes["ContractCode"]
	if !skip {
		if err := action.ContractCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractFileType (uint8)
	_, skip = excludes["ContractFileType"]
	if !skip {
		if err := read(buf, &action.ContractFileType); err != nil {
			return 0, err
		}
	}

	// ContractFile ([]byte)
	_, skip = excludes["ContractFile"]
	if !skip {
		var err error
		action.ContractFile, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// SupportingDocsFileType (uint8)
	_, skip = excludes["SupportingDocsFileType"]
	if !skip {
		if err := read(buf, &action.SupportingDocsFileType); err != nil {
			return 0, err
		}
	}

	// SupportingDocs (string)
	_, skip = excludes["SupportingDocs"]
	if !skip {
		var err error
		action.SupportingDocs, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ContractRevision (uint32)
	_, skip = excludes["ContractRevision"]
	if !skip {
		if err := read(buf, &action.ContractRevision); err != nil {
			return 0, err
		}
	}

	// GoverningLaw (string)
	_, skip = excludes["GoverningLaw"]
	if !skip {
		var err error
		action.GoverningLaw, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// Jurisdiction (string)
	_, skip = excludes["Jurisdiction"]
	if !skip {
		var err error
		action.Jurisdiction, err = ReadFixedChar(buf, 5)
		if err != nil {
			return 0, err
		}
	}

	// EffectiveDate (Timestamp)
	_, skip = excludes["EffectiveDate"]
	if !skip {
		if err := action.EffectiveDate.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractExpiration (Timestamp)
	_, skip = excludes["ContractExpiration"]
	if !skip {
		if err := action.ContractExpiration.Write(buf); err != nil {
			return 0, err
		}
	}

	// ContractURI (string)
	_, skip = excludes["ContractURI"]
	if !skip {
		var err error
		action.ContractURI, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// PrevRevTxID (TxId)
	_, skip = excludes["PrevRevTxID"]
	if !skip {
		if err := action.PrevRevTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Entities ([]Entity)
	_, skip = excludes["Entities"]
	if !skip {
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

// PayloadMessage returns the PayloadMessage, if any.
func (action StaticContractFormation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action StaticContractFormation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("ContractName:%#+v", action.ContractName))
	vals = append(vals, fmt.Sprintf("ContractType:%#+v", action.ContractType))
	vals = append(vals, fmt.Sprintf("ContractCode:%#+v", action.ContractCode))
	vals = append(vals, fmt.Sprintf("ContractFileType:%v", action.ContractFileType))
	vals = append(vals, fmt.Sprintf("ContractFile:%#x", action.ContractFile))
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

// Order Order Action - Issuer to signal to the smart contract that the
// tokens that a particular public address(es) owns are to be confiscated,
// frozen, thawed or reconciled.
type Order struct {
	Header                 Header          `json:"header,omitempty"`                    // Common header data for all actions
	AssetType              string          `json:"asset_type,omitempty"`                // eg. Share, Bond, Ticket
	AssetCode              AssetCode       `json:"asset_code,omitempty"`                // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	ComplianceAction       byte            `json:"compliance_action,omitempty"`         // Freeze (F), Thaw (T), Confiscate (C), Reconciliation (R)
	TargetAddresses        []TargetAddress `json:"target_addresses,omitempty"`          //
	DepositAddress         PublicKeyHash   `json:"deposit_address,omitempty"`           // The public address for confiscated tokens to be deposited in.  Null for Freeze, Thaw, actions. For Reconciliation actions the deposit address is who receives bitcoin.
	AuthorityName          string          `json:"authority_name,omitempty"`            // Length 0-255 bytes. Enforcement Authority Name (eg. Issuer, Queensland Police Service, Tokenized, etc.)
	SigAlgoAddressList     uint8           `json:"sig_algo_address_list,omitempty"`     // 0 = No Registry-signed Message, 1 = ECDSA+secp256k1
	AuthorityPublicKey     string          `json:"authority_public_key,omitempty"`      // Length 0-255 bytes. Public Key associated with the Enforcement Authority
	OrderSignature         string          `json:"order_signature,omitempty"`           // Length 0-255 bytes. Signature for a message that lists out the target addresses and deposit address. Signature of (Contract Address, Asset Code, Compliance Action, Supporting Evidence Hash, Time Out Expiration, TargetAddress1, TargetAddress1Qty, TargetAddressX, TargetAddressXQty,...,DepositAddress)
	SupportingEvidenceTxId TxId            `json:"supporting_evidence_tx_id,omitempty"` // SHA-256: warrant, court order, etc.
	RefTxnID               TxId            `json:"ref_txn_id,omitempty"`                // The settlement action that was dropped from the network.  Not applicable for Freeze, Thaw, and Confiscation orders.  Only applicable for reconcilliation actions.  No subfield when F, T, R is selected as the Compliance Action subfield.
	FreezePeriod           Timestamp       `json:"freeze_period,omitempty"`             // Used for a 'time out'.  Tokens are automatically unfrozen after the expiration timestamp without requiring a Thaw Action. Null value for Thaw, Confiscation and Reconciallitaion orders.
	Message                string          `json:"message,omitempty"`                   //
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ComplianceAction (byte)
	_, skip = excludes["ComplianceAction"]
	if !skip {
		if err := write(buf, action.ComplianceAction); err != nil {
			return nil, err
		}
	}

	// TargetAddresses ([]TargetAddress)
	_, skip = excludes["TargetAddresses"]
	if !skip {
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

	// DepositAddress (PublicKeyHash)
	_, skip = excludes["DepositAddress"]
	if !skip {
		b, err := action.DepositAddress.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// AuthorityName (string)
	_, skip = excludes["AuthorityName"]
	if !skip {
		if err := WriteVarChar(buf, action.AuthorityName, 8); err != nil {
			return nil, err
		}
	}

	// SigAlgoAddressList (uint8)
	_, skip = excludes["SigAlgoAddressList"]
	if !skip {
		if err := write(buf, action.SigAlgoAddressList); err != nil {
			return nil, err
		}
	}

	// AuthorityPublicKey (string)
	_, skip = excludes["AuthorityPublicKey"]
	if !skip {
		if err := WriteVarChar(buf, action.AuthorityPublicKey, 8); err != nil {
			return nil, err
		}
	}

	// OrderSignature (string)
	_, skip = excludes["OrderSignature"]
	if !skip {
		if err := WriteVarChar(buf, action.OrderSignature, 8); err != nil {
			return nil, err
		}
	}

	// SupportingEvidenceTxId (TxId)
	_, skip = excludes["SupportingEvidenceTxId"]
	if !skip {
		b, err := action.SupportingEvidenceTxId.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// RefTxnID (TxId)
	_, skip = excludes["RefTxnID"]
	if !skip {
		b, err := action.RefTxnID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// FreezePeriod (Timestamp)
	_, skip = excludes["FreezePeriod"]
	if !skip {
		b, err := action.FreezePeriod.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Order from the byte slice
func (action *Order) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// ComplianceAction (byte)
	_, skip = excludes["ComplianceAction"]
	if !skip {
		if err := read(buf, &action.ComplianceAction); err != nil {
			return 0, err
		}
	}

	// TargetAddresses ([]TargetAddress)
	_, skip = excludes["TargetAddresses"]
	if !skip {
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

	// DepositAddress (PublicKeyHash)
	_, skip = excludes["DepositAddress"]
	if !skip {
		if err := action.DepositAddress.Write(buf); err != nil {
			return 0, err
		}
	}

	// AuthorityName (string)
	_, skip = excludes["AuthorityName"]
	if !skip {
		var err error
		action.AuthorityName, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SigAlgoAddressList (uint8)
	_, skip = excludes["SigAlgoAddressList"]
	if !skip {
		if err := read(buf, &action.SigAlgoAddressList); err != nil {
			return 0, err
		}
	}

	// AuthorityPublicKey (string)
	_, skip = excludes["AuthorityPublicKey"]
	if !skip {
		var err error
		action.AuthorityPublicKey, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// OrderSignature (string)
	_, skip = excludes["OrderSignature"]
	if !skip {
		var err error
		action.OrderSignature, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// SupportingEvidenceTxId (TxId)
	_, skip = excludes["SupportingEvidenceTxId"]
	if !skip {
		if err := action.SupportingEvidenceTxId.Write(buf); err != nil {
			return 0, err
		}
	}

	// RefTxnID (TxId)
	_, skip = excludes["RefTxnID"]
	if !skip {
		if err := action.RefTxnID.Write(buf); err != nil {
			return 0, err
		}
	}

	// FreezePeriod (Timestamp)
	_, skip = excludes["FreezePeriod"]
	if !skip {
		if err := action.FreezePeriod.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Order) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Order) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("ComplianceAction:%#+v", action.ComplianceAction))
	vals = append(vals, fmt.Sprintf("TargetAddresses:%#+v", action.TargetAddresses))
	vals = append(vals, fmt.Sprintf("DepositAddress:%#+v", action.DepositAddress))
	vals = append(vals, fmt.Sprintf("AuthorityName:%#+v", action.AuthorityName))
	vals = append(vals, fmt.Sprintf("SigAlgoAddressList:%v", action.SigAlgoAddressList))
	vals = append(vals, fmt.Sprintf("AuthorityPublicKey:%#+v", action.AuthorityPublicKey))
	vals = append(vals, fmt.Sprintf("OrderSignature:%#+v", action.OrderSignature))
	vals = append(vals, fmt.Sprintf("SupportingEvidenceTxId:%#+v", action.SupportingEvidenceTxId))
	vals = append(vals, fmt.Sprintf("RefTxnID:%#+v", action.RefTxnID))
	vals = append(vals, fmt.Sprintf("FreezePeriod:%#+v", action.FreezePeriod))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Freeze Freeze Action - To be used to comply with
// contractual/legal/issuer requirements. The target public address(es)
// will be marked as frozen. However the Freeze action publishes this fact
// to the public blockchain for transparency. The Contract will not respond
// to any actions requested by the frozen address.
type Freeze struct {
	Header    Header          `json:"header,omitempty"`    // Common header data for all actions
	Addresses []PublicKeyHash `json:"addresses,omitempty"` // Addresses holding tokens to be frozen.
	Timestamp Timestamp       `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Addresses)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Addresses {
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
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Addresses = make([]PublicKeyHash, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue PublicKeyHash
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Addresses = append(action.Addresses, newValue)
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Freeze) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Freeze) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Addresses:%#+v", action.Addresses))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Thaw Thaw Action - to be used to comply with contractual obligations or
// legal requirements. The Alleged Offender's tokens will be unfrozen to
// allow them to resume normal exchange and governance activities.
type Thaw struct {
	Header    Header          `json:"header,omitempty"`    // Common header data for all actions
	Addresses []PublicKeyHash `json:"addresses,omitempty"` // Addresses holding tokens to be thawed.
	RefTxID   TxId            `json:"ref_tx_id,omitempty"` // The related freeze action.
	Timestamp Timestamp       `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Addresses)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Addresses {
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
	_, skip = excludes["RefTxID"]
	if !skip {
		b, err := action.RefTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Addresses = make([]PublicKeyHash, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue PublicKeyHash
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Addresses = append(action.Addresses, newValue)
		}
	}

	// RefTxID (TxId)
	_, skip = excludes["RefTxID"]
	if !skip {
		if err := action.RefTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Thaw) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Thaw) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Addresses:%#+v", action.Addresses))
	vals = append(vals, fmt.Sprintf("RefTxID:%#+v", action.RefTxID))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Confiscation Confiscation Action - to be used to comply with contractual
// obligations, legal and/or issuer requirements.
type Confiscation struct {
	Header     Header          `json:"header,omitempty"`      // Common header data for all actions
	Addresses  []PublicKeyHash `json:"addresses,omitempty"`   // Addresses holding tokens to be confiscated.
	DepositQty uint64          `json:"deposit_qty,omitempty"` // Custodian's token balance after confiscation.
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
	excludes := make(map[string]bool)
	var skip bool

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Addresses)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Addresses {
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
	_, skip = excludes["DepositQty"]
	if !skip {
		if err := write(buf, action.DepositQty); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Addresses = make([]PublicKeyHash, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue PublicKeyHash
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Addresses = append(action.Addresses, newValue)
		}
	}

	// DepositQty (uint64)
	_, skip = excludes["DepositQty"]
	if !skip {
		if err := read(buf, &action.DepositQty); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Confiscation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Confiscation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Addresses:%#+v", action.Addresses))
	vals = append(vals, fmt.Sprintf("DepositQty:%v", action.DepositQty))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Reconciliation Reconciliation Action - to be used at the direction of
// the issuer to fix record keeping errors with bitcoin and token balances.
type Reconciliation struct {
	Header    Header          `json:"header,omitempty"`    // Common header data for all actions
	Addresses []PublicKeyHash `json:"addresses,omitempty"` // Addresses holding tokens to be reconciled.
	Timestamp Timestamp       `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.Addresses)), 16, 8); err != nil {
			return nil, err
		}
		for _, value := range action.Addresses {
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
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Addresses ([]PublicKeyHash)
	_, skip = excludes["Addresses"]
	if !skip {
		size, err := ReadVariableSize(buf, 16, 8)
		if err != nil {
			return 0, err
		}
		action.Addresses = make([]PublicKeyHash, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue PublicKeyHash
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.Addresses = append(action.Addresses, newValue)
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Reconciliation) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Reconciliation) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Addresses:%#+v", action.Addresses))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Initiative Initiative Action - Allows Token Owners to propose an
// Initiative (aka Initiative/Shareholder vote). A significant cost -
// specified in the Contract Formation - can be attached to this action to
// reduce spam, as the resulting vote will be put to all token owners.
type Initiative struct {
	Header               Header      `json:"header,omitempty"`                 // Common header data for all actions
	AssetType            string      `json:"asset_type,omitempty"`             // eg. Share, Bond, Ticket
	AssetCode            AssetCode   `json:"asset_code,omitempty"`             // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	VoteSystem           uint8       `json:"vote_system,omitempty"`            // X for Vote System X. (1-255, 0 is not valid.)
	Proposal             bool        `json:"proposal,omitempty"`               // 1 for a Proposal, 0 for an initiative that is requesting changes to specific subfields for modification. If this field is true, the subfields should be empty.  The smart contract cannot interpret the results of a vote when Proposal = 1.  All meaning is interpreted by the token owners and smart contract simply facilates the record keeping.  When Proposal = 0, the smart contract always assumes the first choice is a 'yes', or 'pass', if the threshold is met, and will process the proposed changes accordingly.
	ProposedChanges      []Amendment `json:"proposed_changes,omitempty"`       // Each element contains details of which fields to modify, or delete. Because the number of fields in a Contract and Asset is dynamic due to some fields being able to be repeated, the index value of the field needs to be calculated against the Contract or Asset the changes are to apply to. In the event of a Vote being created from this Initiative, the changes will be applied to the version of the Contract or Asset at that time.
	VoteOptions          string      `json:"vote_options,omitempty"`           // Length 1-255 bytes. 0 is not valid. Each byte allows for a different vote option.  Typical votes will likely be multiple choice or Y/N. Vote instances are identified by the Tx-ID. AB000000000 would be chosen for Y/N (binary) type votes.
	VoteMax              uint8       `json:"vote_max,omitempty"`               // Range: 1-X. How many selections can a voter make in a Ballot Cast.  1 is selected for Y/N (binary)
	ProposalDescription  string      `json:"proposal_description,omitempty"`   // Length restricted by the Bitcoin protocol. 0 is valid. Description or details of the vote
	ProposalDocumentHash [32]byte    `json:"proposal_document_hash,omitempty"` // SHA256 Hash of the proposal document to be distributed to voters.
	VoteCutOffTimestamp  Timestamp   `json:"vote_cut_off_timestamp,omitempty"` // Ballot casts after this timestamp will not be included. The vote has finished.
}

// Type returns the type identifer for this message.
func (action Initiative) Type() string {
	return CodeInitiative
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Initiative) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Initiative) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// VoteSystem (uint8)
	_, skip = excludes["VoteSystem"]
	if !skip {
		if err := write(buf, action.VoteSystem); err != nil {
			return nil, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := write(buf, action.Proposal); err != nil {
			return nil, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.ProposedChanges)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ProposedChanges {
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
	_, skip = excludes["VoteOptions"]
	if !skip {
		if err := WriteVarChar(buf, action.VoteOptions, 8); err != nil {
			return nil, err
		}
	}

	// VoteMax (uint8)
	_, skip = excludes["VoteMax"]
	if !skip {
		if err := write(buf, action.VoteMax); err != nil {
			return nil, err
		}
	}

	// ProposalDescription (string)
	_, skip = excludes["ProposalDescription"]
	if !skip {
		if err := WriteVarChar(buf, action.ProposalDescription, 32); err != nil {
			return nil, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	_, skip = excludes["ProposalDocumentHash"]
	if !skip {
		if err := write(buf, action.ProposalDocumentHash); err != nil {
			return nil, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	_, skip = excludes["VoteCutOffTimestamp"]
	if !skip {
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

// write populates the fields in Initiative from the byte slice
func (action *Initiative) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// VoteSystem (uint8)
	_, skip = excludes["VoteSystem"]
	if !skip {
		if err := read(buf, &action.VoteSystem); err != nil {
			return 0, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := read(buf, &action.Proposal); err != nil {
			return 0, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.ProposedChanges = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ProposedChanges = append(action.ProposedChanges, newValue)
		}
	}

	// VoteOptions (string)
	_, skip = excludes["VoteOptions"]
	if !skip {
		var err error
		action.VoteOptions, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// VoteMax (uint8)
	_, skip = excludes["VoteMax"]
	if !skip {
		if err := read(buf, &action.VoteMax); err != nil {
			return 0, err
		}
	}

	// ProposalDescription (string)
	_, skip = excludes["ProposalDescription"]
	if !skip {
		var err error
		action.ProposalDescription, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	_, skip = excludes["ProposalDocumentHash"]
	if !skip {
		if err := read(buf, &action.ProposalDocumentHash); err != nil {
			return 0, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	_, skip = excludes["VoteCutOffTimestamp"]
	if !skip {
		if err := action.VoteCutOffTimestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Initiative) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Initiative) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("VoteSystem:%v", action.VoteSystem))
	vals = append(vals, fmt.Sprintf("Proposal:%#+v", action.Proposal))
	vals = append(vals, fmt.Sprintf("ProposedChanges:%#+v", action.ProposedChanges))
	vals = append(vals, fmt.Sprintf("VoteOptions:%#+v", action.VoteOptions))
	vals = append(vals, fmt.Sprintf("VoteMax:%v", action.VoteMax))
	vals = append(vals, fmt.Sprintf("ProposalDescription:%#+v", action.ProposalDescription))
	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:%#+v", action.ProposalDocumentHash))
	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%#+v", action.VoteCutOffTimestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Referendum Referendum Action - Issuer instructs the Contract to Initiate
// a Token Owner Vote. Usually used for contract amendments, organizational
// governance, etc.
type Referendum struct {
	Header               Header      `json:"header,omitempty"`                 // Common header data for all actions
	AssetSpecificVote    bool        `json:"asset_specific_vote,omitempty"`    // 1 - Yes, 0 - No.  No Asset Type/AssetCode subfields for N - No.
	AssetType            string      `json:"asset_type,omitempty"`             // eg. Share, Bond, Ticket
	AssetCode            AssetCode   `json:"asset_code,omitempty"`             // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	VoteSystem           uint8       `json:"vote_system,omitempty"`            // X for Vote System X. (1-255, 0 is not valid.)
	Proposal             bool        `json:"proposal,omitempty"`               // 1 for a Proposal, 0 for an initiative that is requesting changes to specific subfields for modification. If this field is true, the subfields should be empty.  The smart contract cannot interpret the results of a vote when Proposal = 1.  All meaning is interpreted by the token owners and smart contract simply facilates the record keeping.  When Proposal = 0, the smart contract always assumes the first choice is a 'yes', or 'pass', if the threshold is met, and will process the proposed changes accordingly.
	ProposedChanges      []Amendment `json:"proposed_changes,omitempty"`       // Each element contains details of which fields to modify, or delete. Because the number of fields in a Contract and Asset is dynamic due to some fields being able to be repeated, the index value of the field needs to be calculated against the Contract or Asset the changes are to apply to. In the event of a Vote being created from this Initiative, the changes will be applied to the version of the Contract or Asset at that time.
	VoteOptions          string      `json:"vote_options,omitempty"`           // Length 1-255 bytes. 0 is not valid. Each byte allows for a different vote option.  Typical votes will likely be multiple choice or Y/N. Vote instances are identified by the Tx-ID. AB000000000 would be chosen for Y/N (binary) type votes. Only applicable if Proposal Type is set to P for Proposal.  All other Proposal Types will be binary.  Pass/Fail.
	VoteMax              uint8       `json:"vote_max,omitempty"`               // Range: 1-15. How many selections can a voter make in a Ballot Cast.  1 is selected for Y/N (binary)
	ProposalDescription  string      `json:"proposal_description,omitempty"`   // Length restricted by the Bitcoin protocol. 0 is valid. Description of the vote.
	ProposalDocumentHash [32]byte    `json:"proposal_document_hash,omitempty"` // SHA256 Hash of the proposal document to be distributed to voters
	VoteCutOffTimestamp  Timestamp   `json:"vote_cut_off_timestamp,omitempty"` // Ballot casts after this timestamp will not be included. The vote has finished.
}

// Type returns the type identifer for this message.
func (action Referendum) Type() string {
	return CodeReferendum
}

// Read implements the io.Reader interface, writing the receiver to the
// []byte.
func (action *Referendum) read(b []byte) (int, error) {
	data, err := action.serialize()

	if err != nil {
		return 0, err
	}

	copy(b, data)

	return len(b), nil
}

// serialize returns the full OP_RETURN payload bytes.
func (action *Referendum) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	excludes := make(map[string]bool)
	var skip bool

	// AssetSpecificVote (bool)
	_, skip = excludes["AssetSpecificVote"]
	if !skip {
		if err := write(buf, action.AssetSpecificVote); err != nil {
			return nil, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// VoteSystem (uint8)
	_, skip = excludes["VoteSystem"]
	if !skip {
		if err := write(buf, action.VoteSystem); err != nil {
			return nil, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := write(buf, action.Proposal); err != nil {
			return nil, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.ProposedChanges)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ProposedChanges {
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
	_, skip = excludes["VoteOptions"]
	if !skip {
		if err := WriteVarChar(buf, action.VoteOptions, 8); err != nil {
			return nil, err
		}
	}

	// VoteMax (uint8)
	_, skip = excludes["VoteMax"]
	if !skip {
		if err := write(buf, action.VoteMax); err != nil {
			return nil, err
		}
	}

	// ProposalDescription (string)
	_, skip = excludes["ProposalDescription"]
	if !skip {
		if err := WriteVarChar(buf, action.ProposalDescription, 32); err != nil {
			return nil, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	_, skip = excludes["ProposalDocumentHash"]
	if !skip {
		if err := write(buf, action.ProposalDocumentHash); err != nil {
			return nil, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	_, skip = excludes["VoteCutOffTimestamp"]
	if !skip {
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

// write populates the fields in Referendum from the byte slice
func (action *Referendum) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetSpecificVote (bool)
	_, skip = excludes["AssetSpecificVote"]
	if !skip {
		if err := read(buf, &action.AssetSpecificVote); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// VoteSystem (uint8)
	_, skip = excludes["VoteSystem"]
	if !skip {
		if err := read(buf, &action.VoteSystem); err != nil {
			return 0, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := read(buf, &action.Proposal); err != nil {
			return 0, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.ProposedChanges = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ProposedChanges = append(action.ProposedChanges, newValue)
		}
	}

	// VoteOptions (string)
	_, skip = excludes["VoteOptions"]
	if !skip {
		var err error
		action.VoteOptions, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// VoteMax (uint8)
	_, skip = excludes["VoteMax"]
	if !skip {
		if err := read(buf, &action.VoteMax); err != nil {
			return 0, err
		}
	}

	// ProposalDescription (string)
	_, skip = excludes["ProposalDescription"]
	if !skip {
		var err error
		action.ProposalDescription, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// ProposalDocumentHash ([32]byte)
	_, skip = excludes["ProposalDocumentHash"]
	if !skip {
		if err := read(buf, &action.ProposalDocumentHash); err != nil {
			return 0, err
		}
	}

	// VoteCutOffTimestamp (Timestamp)
	_, skip = excludes["VoteCutOffTimestamp"]
	if !skip {
		if err := action.VoteCutOffTimestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Referendum) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Referendum) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetSpecificVote:%#+v", action.AssetSpecificVote))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("VoteSystem:%v", action.VoteSystem))
	vals = append(vals, fmt.Sprintf("Proposal:%#+v", action.Proposal))
	vals = append(vals, fmt.Sprintf("ProposedChanges:%#+v", action.ProposedChanges))
	vals = append(vals, fmt.Sprintf("VoteOptions:%#+v", action.VoteOptions))
	vals = append(vals, fmt.Sprintf("VoteMax:%v", action.VoteMax))
	vals = append(vals, fmt.Sprintf("ProposalDescription:%#+v", action.ProposalDescription))
	vals = append(vals, fmt.Sprintf("ProposalDocumentHash:%#+v", action.ProposalDocumentHash))
	vals = append(vals, fmt.Sprintf("VoteCutOffTimestamp:%#+v", action.VoteCutOffTimestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Vote Vote Action - A vote is created by the Contract in response to a
// valid Referendum (Issuer) or Initiative (User) Action.
type Vote struct {
	Header    Header    `json:"header,omitempty"`    // Common header data for all actions
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
	excludes := make(map[string]bool)
	var skip bool

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Vote) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Vote) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCast Ballot Cast Action - Used by Token Owners to cast their
// ballot (vote) on proposals raised by the Issuer (Referendum) or other
// token holders (Initiative). 1 Vote per token unless a vote multiplier is
// specified in the relevant Asset Definition action.
type BallotCast struct {
	Header    Header    `json:"header,omitempty"`     // Common header data for all actions
	AssetType string    `json:"asset_type,omitempty"` // eg. Share, Bond, Ticket
	AssetCode AssetCode `json:"asset_code,omitempty"` // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	VoteTxID  TxId      `json:"vote_tx_id,omitempty"` // Tx ID of the Vote the Ballot Cast is being made for.
	Vote      string    `json:"vote,omitempty"`       // Length 1-255 bytes. 0 is not valid. Accept, Reject, Abstain, Spoiled, Multiple Choice, or Preference List. 15 options total. Order of preference.  1st position = 1st choice. 2nd position = 2nd choice, etc.  A is always Accept and B is always reject in a Y/N votes.
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// VoteTxID (TxId)
	_, skip = excludes["VoteTxID"]
	if !skip {
		b, err := action.VoteTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Vote (string)
	_, skip = excludes["Vote"]
	if !skip {
		if err := WriteVarChar(buf, action.Vote, 8); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in BallotCast from the byte slice
func (action *BallotCast) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// VoteTxID (TxId)
	_, skip = excludes["VoteTxID"]
	if !skip {
		if err := action.VoteTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// Vote (string)
	_, skip = excludes["Vote"]
	if !skip {
		var err error
		action.Vote, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action BallotCast) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action BallotCast) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("VoteTxID:%#+v", action.VoteTxID))
	vals = append(vals, fmt.Sprintf("Vote:%#+v", action.Vote))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// BallotCounted Ballot Counted Action - The smart contract will respond to
// a Ballot Cast action with a Ballot Counted action if the Ballot Cast is
// valid. If the Ballot Cast is not valid, then the smart contract will
// respond with a Rejection Action.
type BallotCounted struct {
	Header    Header    `json:"header,omitempty"`    // Common header data for all actions
	Timestamp Timestamp `json:"timestamp,omitempty"` // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action BallotCounted) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action BallotCounted) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Result Result Action - Once a vote has been completed the results are
// published.
type Result struct {
	Header           Header      `json:"header,omitempty"`             // Common header data for all actions
	AssetType        string      `json:"asset_type,omitempty"`         // eg. Share, Bond, Ticket
	AssetCode        AssetCode   `json:"asset_code,omitempty"`         // 32 randomly generated bytes.  Each Asset Code should be unique.  However, an Asset Code is always linked to a Contract that is identified by the public address of the Contract wallet. The Asset Type + Asset Code = Asset Code.  An Asset Code is a human readable identifier that can be used in a similar way to a Bitcoin (BSV) address.
	Proposal         bool        `json:"proposal,omitempty"`           // 1 for a Proposal, 0 for an initiative that is requesting changes to specific subfields for modification. If this field is true, the subfields should be empty.  The smart contract cannot interpret the results of a vote when Proposal = 1.  All meaning is interpreted by the token owners and smart contract simply facilates the record keeping.  When Proposal = 0, the smart contract always assumes the first choice is a 'yes', or 'pass', if the threshold is met, and will process the proposed changes accordingly.
	ProposedChanges  []Amendment `json:"proposed_changes,omitempty"`   // Each element contains details of which fields to modify, or delete. Because the number of fields in a Contract and Asset is dynamic due to some fields being able to be repeated, the index value of the field needs to be calculated against the Contract or Asset the changes are to apply to. In the event of a Vote being created from this Initiative, the changes will be applied to the version of the Contract or Asset at that time.
	VoteTxID         TxId        `json:"vote_tx_id,omitempty"`         // Link to the Vote Action txn.
	VoteOptionsCount uint8       `json:"vote_options_count,omitempty"` // Number of Vote Options to follow.
	OptionXTally     uint64      `json:"option_x_tally,omitempty"`     // Number of valid votes counted for Option X
	Result           string      `json:"result,omitempty"`             // Length 1-255 bytes. 0 is not valid. The Option with the most votes. In the event of a draw for 1st place, all winning options are listed.
	Timestamp        Timestamp   `json:"timestamp,omitempty"`          // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		if err := WriteFixedChar(buf, action.AssetType, 3); err != nil {
			return nil, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		b, err := action.AssetCode.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := write(buf, action.Proposal); err != nil {
			return nil, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.ProposedChanges)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.ProposedChanges {
			b, err := value.Serialize()
			if err != nil {
				return nil, err
			}

			if err := write(buf, b); err != nil {
				return nil, err
			}
		}
	}

	// VoteTxID (TxId)
	_, skip = excludes["VoteTxID"]
	if !skip {
		b, err := action.VoteTxID.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// VoteOptionsCount (uint8)
	_, skip = excludes["VoteOptionsCount"]
	if !skip {
		if err := write(buf, action.VoteOptionsCount); err != nil {
			return nil, err
		}
	}

	// OptionXTally (uint64)
	_, skip = excludes["OptionXTally"]
	if !skip {
		if err := write(buf, action.OptionXTally); err != nil {
			return nil, err
		}
	}

	// Result (string)
	_, skip = excludes["Result"]
	if !skip {
		if err := WriteVarChar(buf, action.Result, 8); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AssetType (string)
	_, skip = excludes["AssetType"]
	if !skip {
		var err error
		action.AssetType, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// AssetCode (AssetCode)
	_, skip = excludes["AssetCode"]
	if !skip {
		if err := action.AssetCode.Write(buf); err != nil {
			return 0, err
		}
	}

	// Proposal (bool)
	_, skip = excludes["Proposal"]
	if !skip {
		if err := read(buf, &action.Proposal); err != nil {
			return 0, err
		}
	}

	// ProposedChanges ([]Amendment)
	_, skip = excludes["ProposedChanges"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.ProposedChanges = make([]Amendment, 0, size)
		for i := uint64(0); i < size; i++ {
			var newValue Amendment
			if err := newValue.Write(buf); err != nil {
				return 0, err
			}

			action.ProposedChanges = append(action.ProposedChanges, newValue)
		}
	}

	// VoteTxID (TxId)
	_, skip = excludes["VoteTxID"]
	if !skip {
		if err := action.VoteTxID.Write(buf); err != nil {
			return 0, err
		}
	}

	// VoteOptionsCount (uint8)
	_, skip = excludes["VoteOptionsCount"]
	if !skip {
		if err := read(buf, &action.VoteOptionsCount); err != nil {
			return 0, err
		}
	}

	// OptionXTally (uint64)
	_, skip = excludes["OptionXTally"]
	if !skip {
		if err := read(buf, &action.OptionXTally); err != nil {
			return 0, err
		}
	}

	// Result (string)
	_, skip = excludes["Result"]
	if !skip {
		var err error
		action.Result, err = ReadVarChar(buf, 8)
		if err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Result) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Result) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AssetType:%#+v", action.AssetType))
	vals = append(vals, fmt.Sprintf("AssetCode:%#+v", action.AssetCode))
	vals = append(vals, fmt.Sprintf("Proposal:%#+v", action.Proposal))
	vals = append(vals, fmt.Sprintf("ProposedChanges:%#+v", action.ProposedChanges))
	vals = append(vals, fmt.Sprintf("VoteTxID:%#+v", action.VoteTxID))
	vals = append(vals, fmt.Sprintf("VoteOptionsCount:%v", action.VoteOptionsCount))
	vals = append(vals, fmt.Sprintf("OptionXTally:%v", action.OptionXTally))
	vals = append(vals, fmt.Sprintf("Result:%#+v", action.Result))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Message Message Action - the message action is a general purpose
// communication action. 'Twitter/SMS' for Issuers/Investors/Users. The
// message txn can also be used for passing partially signed txns on-chain,
// establishing private communication channels and EDI (receipting,
// invoices, PO, and private offers/bids). The messages are broken down by
// type for easy filtering in the a user's wallet. The Message Types are
// listed in the Message Types table.
type Message struct {
	Header         Header   `json:"header,omitempty"`          // Common header data for all actions
	AddressIndexes []uint16 `json:"address_indexes,omitempty"` // Associates the message to a particular output by the index.
	MessageType    string   `json:"message_type,omitempty"`    // Potential for up to 65,535 different message types
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
	excludes := make(map[string]bool)
	var skip bool

	// AddressIndexes ([]uint16)
	_, skip = excludes["AddressIndexes"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.AddressIndexes)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.AddressIndexes {
			if err := write(buf, value); err != nil {
				return nil, err
			}
		}
	}

	// MessageType (string)
	_, skip = excludes["MessageType"]
	if !skip {
		if err := WriteFixedChar(buf, action.MessageType, 4); err != nil {
			return nil, err
		}
	}

	// MessagePayload ([]byte)
	_, skip = excludes["MessagePayload"]
	if !skip {
		if err := WriteVarBin(buf, action.MessagePayload, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Message from the byte slice
func (action *Message) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// AddressIndexes ([]uint16)
	_, skip = excludes["AddressIndexes"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.AddressIndexes = make([]uint16, size, size)
		if err := read(buf, &action.AddressIndexes); err != nil {
			return 0, err
		}
	}

	// MessageType (string)
	_, skip = excludes["MessageType"]
	if !skip {
		var err error
		action.MessageType, err = ReadFixedChar(buf, 4)
		if err != nil {
			return 0, err
		}
	}

	// MessagePayload ([]byte)
	_, skip = excludes["MessagePayload"]
	if !skip {
		var err error
		action.MessagePayload, err = ReadVarBin(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Message) PayloadMessage() (PayloadMessage, error) {
	p := MessageTypeMapping(action.MessageType)
	if p == nil {
		return nil, fmt.Errorf("Undefined message type : %s", action.MessageType)
	}

	if _, err := p.Write(action.MessagePayload); err != nil {
		return nil, err
	}

	return p, nil
}

func (action Message) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("AddressIndexes:%v", action.AddressIndexes))
	vals = append(vals, fmt.Sprintf("MessageType:%#+v", action.MessageType))
	vals = append(vals, fmt.Sprintf("MessagePayload:%#x", action.MessagePayload))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Rejection Rejection Action - used to reject request actions that do not
// comply with the Contract. If money is to be returned to a User then it
// is used in lieu of the Settlement Action to properly account for token
// balances. All Issuer/User request Actions must be responded to by the
// Contract with an Action. The only exception to this rule is when there
// is not enough fees in the first Action for the Contract response action
// to remain revenue neutral. If not enough fees are attached to pay for
// the Contract response then the Contract will not respond.
type Rejection struct {
	Header                Header    `json:"header,omitempty"`                  // Common header data for all actions
	QtyReceivingAddresses uint8     `json:"qty_receiving_addresses,omitempty"` // 0-255 Message Receiving Addresses
	AddressIndexes        []uint16  `json:"address_indexes,omitempty"`         // Associates the message to a particular output by the index.
	RejectionType         uint8     `json:"rejection_type,omitempty"`          // Classifies the rejection by a type.
	MessagePayload        string    `json:"message_payload,omitempty"`         // Length 0-65,535 bytes. Message that explains the reasoning for a rejection, if needed.  Most rejection types will be captured by the Rejection Type Subfield.
	Timestamp             Timestamp `json:"timestamp,omitempty"`               // Timestamp in nanoseconds of when the smart contract created the action.
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
	excludes := make(map[string]bool)
	var skip bool

	// QtyReceivingAddresses (uint8)
	_, skip = excludes["QtyReceivingAddresses"]
	if !skip {
		if err := write(buf, action.QtyReceivingAddresses); err != nil {
			return nil, err
		}
	}

	// AddressIndexes ([]uint16)
	_, skip = excludes["AddressIndexes"]
	if !skip {
		if err := WriteVariableSize(buf, uint64(len(action.AddressIndexes)), 0, 8); err != nil {
			return nil, err
		}
		for _, value := range action.AddressIndexes {
			if err := write(buf, value); err != nil {
				return nil, err
			}
		}
	}

	// RejectionType (uint8)
	_, skip = excludes["RejectionType"]
	if !skip {
		if err := write(buf, action.RejectionType); err != nil {
			return nil, err
		}
	}

	// MessagePayload (string)
	_, skip = excludes["MessagePayload"]
	if !skip {
		if err := WriteVarChar(buf, action.MessagePayload, 32); err != nil {
			return nil, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// QtyReceivingAddresses (uint8)
	_, skip = excludes["QtyReceivingAddresses"]
	if !skip {
		if err := read(buf, &action.QtyReceivingAddresses); err != nil {
			return 0, err
		}
	}

	// AddressIndexes ([]uint16)
	_, skip = excludes["AddressIndexes"]
	if !skip {
		size, err := ReadVariableSize(buf, 0, 8)
		if err != nil {
			return 0, err
		}
		action.AddressIndexes = make([]uint16, size, size)
		if err := read(buf, &action.AddressIndexes); err != nil {
			return 0, err
		}
	}

	// RejectionType (uint8)
	_, skip = excludes["RejectionType"]
	if !skip {
		if err := read(buf, &action.RejectionType); err != nil {
			return 0, err
		}
	}

	// MessagePayload (string)
	_, skip = excludes["MessagePayload"]
	if !skip {
		var err error
		action.MessagePayload, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	// Timestamp (Timestamp)
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Rejection) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Rejection) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("QtyReceivingAddresses:%v", action.QtyReceivingAddresses))
	vals = append(vals, fmt.Sprintf("AddressIndexes:%v", action.AddressIndexes))
	vals = append(vals, fmt.Sprintf("RejectionType:%v", action.RejectionType))
	vals = append(vals, fmt.Sprintf("MessagePayload:%#+v", action.MessagePayload))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Establishment Establishment Action - Establishes an on-chain register.
type Establishment struct {
	Header  Header `json:"header,omitempty"`  // Common header data for all actions
	Message string `json:"message,omitempty"` //
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
	excludes := make(map[string]bool)
	var skip bool

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Establishment from the byte slice
func (action *Establishment) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Establishment) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Establishment) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Addition Addition Action - Adds an entry to the Register.
type Addition struct {
	Header  Header `json:"header,omitempty"`  // Common header data for all actions
	Message string `json:"message,omitempty"` //
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
	excludes := make(map[string]bool)
	var skip bool

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Addition from the byte slice
func (action *Addition) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Addition) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Addition) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Alteration Alteration Action - A register entry/record can be altered.
type Alteration struct {
	Header  Header `json:"header,omitempty"`  // Common header data for all actions
	Message string `json:"message,omitempty"` //
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
	excludes := make(map[string]bool)
	var skip bool

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Alteration from the byte slice
func (action *Alteration) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Alteration) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Alteration) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Message:%#+v", action.Message))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Removal Removal Action - Removes an entry/record from the Register.
type Removal struct {
	Header  Header `json:"header,omitempty"`  // Common header data for all actions
	Message string `json:"message,omitempty"` //
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
	excludes := make(map[string]bool)
	var skip bool

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		if err := WriteVarChar(buf, action.Message, 32); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// write populates the fields in Removal from the byte slice
func (action *Removal) write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Message (string)
	_, skip = excludes["Message"]
	if !skip {
		var err error
		action.Message, err = ReadVarChar(buf, 32)
		if err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Removal) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Removal) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
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
	Header              Header          `json:"header,omitempty"`                // Common header data for all actions
	Assets              []AssetTransfer `json:"assets,omitempty"`                // The Assets involved in the Transfer Action.
	OfferExpiry         Timestamp       `json:"offer_expiry,omitempty"`          // This prevents any party from holding on to the partially signed message as a form of an option.  Eg. the exchange at this price is valid for 30 mins.
	ExchangeFeeCurrency string          `json:"exchange_fee_currency,omitempty"` // BSV, USD, AUD, EUR, etc.
	ExchangeFeeVar      float32         `json:"exchange_fee_var,omitempty"`      // Percent of the value of the transaction
	ExchangeFeeFixed    float32         `json:"exchange_fee_fixed,omitempty"`    // Fixed fee
	ExchangeFeeAddress  PublicKeyHash   `json:"exchange_fee_address,omitempty"`  // Identifies the public address that the exchange fee should be paid to.
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
	excludes := make(map[string]bool)
	var skip bool

	// Assets ([]AssetTransfer)
	_, skip = excludes["Assets"]
	if !skip {
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
	_, skip = excludes["OfferExpiry"]
	if !skip {
		b, err := action.OfferExpiry.Serialize()
		if err != nil {
			return nil, err
		}

		if err := write(buf, b); err != nil {
			return nil, err
		}
	}

	// ExchangeFeeCurrency (string)
	_, skip = excludes["ExchangeFeeCurrency"]
	if !skip {
		if err := WriteFixedChar(buf, action.ExchangeFeeCurrency, 3); err != nil {
			return nil, err
		}
	}

	// ExchangeFeeVar (float32)
	_, skip = excludes["ExchangeFeeVar"]
	if !skip {
		if err := write(buf, action.ExchangeFeeVar); err != nil {
			return nil, err
		}
	}

	// ExchangeFeeFixed (float32)
	_, skip = excludes["ExchangeFeeFixed"]
	if !skip {
		if err := write(buf, action.ExchangeFeeFixed); err != nil {
			return nil, err
		}
	}

	// ExchangeFeeAddress (PublicKeyHash)
	_, skip = excludes["ExchangeFeeAddress"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Assets ([]AssetTransfer)
	_, skip = excludes["Assets"]
	if !skip {
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
	_, skip = excludes["OfferExpiry"]
	if !skip {
		if err := action.OfferExpiry.Write(buf); err != nil {
			return 0, err
		}
	}

	// ExchangeFeeCurrency (string)
	_, skip = excludes["ExchangeFeeCurrency"]
	if !skip {
		var err error
		action.ExchangeFeeCurrency, err = ReadFixedChar(buf, 3)
		if err != nil {
			return 0, err
		}
	}

	// ExchangeFeeVar (float32)
	_, skip = excludes["ExchangeFeeVar"]
	if !skip {
		if err := read(buf, &action.ExchangeFeeVar); err != nil {
			return 0, err
		}
	}

	// ExchangeFeeFixed (float32)
	_, skip = excludes["ExchangeFeeFixed"]
	if !skip {
		if err := read(buf, &action.ExchangeFeeFixed); err != nil {
			return 0, err
		}
	}

	// ExchangeFeeAddress (PublicKeyHash)
	_, skip = excludes["ExchangeFeeAddress"]
	if !skip {
		if err := action.ExchangeFeeAddress.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Transfer) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Transfer) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Assets:%#+v", action.Assets))
	vals = append(vals, fmt.Sprintf("OfferExpiry:%#+v", action.OfferExpiry))
	vals = append(vals, fmt.Sprintf("ExchangeFeeCurrency:%#+v", action.ExchangeFeeCurrency))
	vals = append(vals, fmt.Sprintf("ExchangeFeeVar:%v", action.ExchangeFeeVar))
	vals = append(vals, fmt.Sprintf("ExchangeFeeFixed:%v", action.ExchangeFeeFixed))
	vals = append(vals, fmt.Sprintf("ExchangeFeeAddress:%#+v", action.ExchangeFeeAddress))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}

// Settlement Settlement Action - Settles the transfer request of bitcoins
// and tokens from transfer (T1) actions.
type Settlement struct {
	Header    Header            `json:"header,omitempty"`    // Common header data for all actions
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
	excludes := make(map[string]bool)
	var skip bool

	// Assets ([]AssetSettlement)
	_, skip = excludes["Assets"]
	if !skip {
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
	_, skip = excludes["Timestamp"]
	if !skip {
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
	excludes := make(map[string]bool)
	var skip bool
	// Header (Header)
	_, skip = excludes["Header"]
	if !skip {
		if err := action.Header.Write(buf); err != nil {
			return 0, err
		}
	}

	// Assets ([]AssetSettlement)
	_, skip = excludes["Assets"]
	if !skip {
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
	_, skip = excludes["Timestamp"]
	if !skip {
		if err := action.Timestamp.Write(buf); err != nil {
			return 0, err
		}
	}

	return len(b) - buf.Len(), nil
}

// PayloadMessage returns the PayloadMessage, if any.
func (action Settlement) PayloadMessage() (PayloadMessage, error) {
	return nil, nil
}

func (action Settlement) String() string {
	vals := []string{}

	vals = append(vals, fmt.Sprintf("Header:%#+v", action.Header))
	vals = append(vals, fmt.Sprintf("Assets:%#+v", action.Assets))
	vals = append(vals, fmt.Sprintf("Timestamp:%#+v", action.Timestamp))

	return fmt.Sprintf("{%s}", strings.Join(vals, " "))
}
