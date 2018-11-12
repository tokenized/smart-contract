package protocol

import "encoding/json"

// The code in this file is auto-generated. Do not edit it by hand as it will
// be overwritten when code is regenerated.

// NewFormByCode returns a new ProtocolForm by code.
//
// An error will be returned if there is no matching Form.
func NewFormByCode(code string) (Form, error) {

	if code == CodeAssetDefinition {
		return &AssetDefinitionForm{}, nil
	}

	if code == CodeAssetCreation {
		return &AssetCreationForm{}, nil
	}

	if code == CodeAssetModification {
		return &AssetModificationForm{}, nil
	}

	if code == CodeContractOffer {
		return &ContractOfferForm{}, nil
	}

	if code == CodeContractFormation {
		return &ContractFormationForm{}, nil
	}

	if code == CodeContractAmendment {
		return &ContractAmendmentForm{}, nil
	}

	if code == CodeOrder {
		return &OrderForm{}, nil
	}

	if code == CodeFreeze {
		return &FreezeForm{}, nil
	}

	if code == CodeThaw {
		return &ThawForm{}, nil
	}

	if code == CodeConfiscation {
		return &ConfiscationForm{}, nil
	}

	if code == CodeReconciliation {
		return &ReconciliationForm{}, nil
	}

	if code == CodeInitiative {
		return &InitiativeForm{}, nil
	}

	if code == CodeReferendum {
		return &ReferendumForm{}, nil
	}

	if code == CodeVote {
		return &VoteForm{}, nil
	}

	if code == CodeBallotCast {
		return &BallotCastForm{}, nil
	}

	if code == CodeBallotCounted {
		return &BallotCountedForm{}, nil
	}

	if code == CodeResult {
		return &ResultForm{}, nil
	}

	if code == CodeMessage {
		return &MessageForm{}, nil
	}

	if code == CodeRejection {
		return &RejectionForm{}, nil
	}

	if code == CodeEstablishment {
		return &EstablishmentForm{}, nil
	}

	if code == CodeAddition {
		return &AdditionForm{}, nil
	}

	if code == CodeAlteration {
		return &AlterationForm{}, nil
	}

	if code == CodeRemoval {
		return &RemovalForm{}, nil
	}

	if code == CodeSend {
		return &SendForm{}, nil
	}

	if code == CodeExchange {
		return &ExchangeForm{}, nil
	}

	if code == CodeSwap {
		return &SwapForm{}, nil
	}

	if code == CodeSettlement {
		return &SettlementForm{}, nil
	}

	return nil, ErrUnknownMessage
}

// AssetDefinitionForm is the JSON friendly version of a AssetDefinition.
type AssetDefinitionForm struct {
	BaseForm

	Version             uint8           `json:"version,omitempty"`
	AssetType           string          `json:"asset_type,omitempty"`
	AssetID             string          `json:"asset_id,omitempty"`
	AuthorizationFlags  string          `json:"authorization_flags,omitempty"`
	VotingSystem        string          `json:"voting_system,omitempty"`
	VoteMultiplier      uint8           `json:"vote_multiplier,omitempty"`
	Qty                 uint64          `json:"qty,omitempty"`
	ContractFeeCurrency string          `json:"contract_fee_currency,omitempty"`
	ContractFeeVar      float32         `json:"contract_fee_var,omitempty"`
	ContractFeeFixed    float32         `json:"contract_fee_fixed,omitempty"`
	Payload             json.RawMessage `json:"payload,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *AssetDefinitionForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f AssetDefinitionForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f AssetDefinitionForm) PayloadForm() (PayloadForm, error) {
	pf, err := NewPayloadFormByCode(f.AssetType)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(f.Payload, pf); err != nil {
		return nil, err
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	f.Payload = b

	return pf, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f AssetDefinitionForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewAssetDefinition()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.VoteMultiplier = f.VoteMultiplier

	m.Qty = f.Qty

	m.ContractFeeCurrency, err = f.pad(f.ContractFeeCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.ContractFeeVar = f.ContractFeeVar

	m.ContractFeeFixed = f.ContractFeeFixed

	m.Payload = f.Payload

	pf, err := f.PayloadForm()
	if err != nil {
		return nil, err
	}

	if pf == nil {
		return &m, nil
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	m.Payload = b

	return &m, nil
}

// AssetCreationForm is the JSON friendly version of a AssetCreation.
type AssetCreationForm struct {
	BaseForm

	Version             uint8           `json:"version,omitempty"`
	AssetType           string          `json:"asset_type,omitempty"`
	AssetID             string          `json:"asset_id,omitempty"`
	AssetRevision       uint16          `json:"asset_revision,omitempty"`
	AuthorizationFlags  string          `json:"authorization_flags,omitempty"`
	VotingSystem        string          `json:"voting_system,omitempty"`
	VoteMultiplier      uint8           `json:"vote_multiplier,omitempty"`
	Qty                 uint64          `json:"qty,omitempty"`
	ContractFeeCurrency string          `json:"contract_fee_currency,omitempty"`
	ContractFeeVar      float32         `json:"contract_fee_var,omitempty"`
	ContractFeeFixed    float32         `json:"contract_fee_fixed,omitempty"`
	Payload             json.RawMessage `json:"payload,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *AssetCreationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f AssetCreationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f AssetCreationForm) PayloadForm() (PayloadForm, error) {
	pf, err := NewPayloadFormByCode(f.AssetType)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(f.Payload, pf); err != nil {
		return nil, err
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	f.Payload = b

	return pf, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f AssetCreationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewAssetCreation()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.AssetRevision = f.AssetRevision

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.VoteMultiplier = f.VoteMultiplier

	m.Qty = f.Qty

	m.ContractFeeCurrency, err = f.pad(f.ContractFeeCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.ContractFeeVar = f.ContractFeeVar

	m.ContractFeeFixed = f.ContractFeeFixed

	m.Payload = f.Payload

	pf, err := f.PayloadForm()
	if err != nil {
		return nil, err
	}

	if pf == nil {
		return &m, nil
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	m.Payload = b

	return &m, nil
}

// AssetModificationForm is the JSON friendly version of a AssetModification.
type AssetModificationForm struct {
	BaseForm

	Version             uint8           `json:"version,omitempty"`
	AssetType           string          `json:"asset_type,omitempty"`
	AssetID             string          `json:"asset_id,omitempty"`
	AssetRevision       uint16          `json:"asset_revision,omitempty"`
	AuthorizationFlags  string          `json:"authorization_flags,omitempty"`
	VotingSystem        string          `json:"voting_system,omitempty"`
	VoteMultiplier      uint8           `json:"vote_multiplier,omitempty"`
	Qty                 uint64          `json:"qty,omitempty"`
	ContractFeeCurrency string          `json:"contract_fee_currency,omitempty"`
	ContractFeeVar      float32         `json:"contract_fee_var,omitempty"`
	ContractFeeFixed    float32         `json:"contract_fee_fixed,omitempty"`
	Payload             json.RawMessage `json:"payload,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *AssetModificationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f AssetModificationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f AssetModificationForm) PayloadForm() (PayloadForm, error) {
	pf, err := NewPayloadFormByCode(f.AssetType)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(f.Payload, pf); err != nil {
		return nil, err
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	f.Payload = b

	return pf, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f AssetModificationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewAssetModification()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.AssetRevision = f.AssetRevision

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.VoteMultiplier = f.VoteMultiplier

	m.Qty = f.Qty

	m.ContractFeeCurrency, err = f.pad(f.ContractFeeCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.ContractFeeVar = f.ContractFeeVar

	m.ContractFeeFixed = f.ContractFeeFixed

	m.Payload = f.Payload

	pf, err := f.PayloadForm()
	if err != nil {
		return nil, err
	}

	if pf == nil {
		return &m, nil
	}

	b, err := pf.Bytes()
	if err != nil {
		return nil, err
	}

	m.Payload = b

	return &m, nil
}

// ContractOfferForm is the JSON friendly version of a ContractOffer.
type ContractOfferForm struct {
	BaseForm

	Version                     uint8   `json:"version,omitempty"`
	ContractName                string  `json:"contract_name,omitempty"`
	ContractFileHash            string  `json:"contract_file_hash,omitempty"`
	GoverningLaw                string  `json:"governing_law,omitempty"`
	Jurisdiction                string  `json:"jurisdiction,omitempty"`
	ContractExpiration          uint64  `json:"contract_expiration,omitempty"`
	URI                         string  `json:"uri,omitempty"`
	IssuerID                    string  `json:"issuer_id,omitempty"`
	IssuerType                  string  `json:"issuer_type,omitempty"`
	ContractOperatorID          string  `json:"contract_operator_id,omitempty"`
	AuthorizationFlags          string  `json:"authorization_flags,omitempty"`
	VotingSystem                string  `json:"voting_system,omitempty"`
	InitiativeThreshold         float32 `json:"initiative_threshold,omitempty"`
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency,omitempty"`
	RestrictedQty               uint64  `json:"restricted_qty,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ContractOfferForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ContractOfferForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ContractOfferForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ContractOfferForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewContractOffer()

	m.Version = f.Version

	m.ContractName, err = f.pad(f.ContractName, 32)
	if err != nil {
		return nil, err
	}

	m.ContractFileHash, err = f.pad(f.ContractFileHash, 32)
	if err != nil {
		return nil, err
	}

	m.GoverningLaw, err = f.pad(f.GoverningLaw, 5)
	if err != nil {
		return nil, err
	}

	m.Jurisdiction, err = f.pad(f.Jurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.ContractExpiration = f.ContractExpiration

	m.URI, err = f.pad(f.URI, 78)
	if err != nil {
		return nil, err
	}

	m.IssuerID, err = f.pad(f.IssuerID, 16)
	if err != nil {
		return nil, err
	}

	m.IssuerType, err = f.ensureByte(f.IssuerType)
	if err != nil {
		return nil, err
	}

	m.ContractOperatorID, err = f.pad(f.ContractOperatorID, 16)
	if err != nil {
		return nil, err
	}

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.InitiativeThreshold = f.InitiativeThreshold

	m.InitiativeThresholdCurrency, err = f.pad(f.InitiativeThresholdCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.RestrictedQty = f.RestrictedQty

	return &m, nil
}

// ContractFormationForm is the JSON friendly version of a ContractFormation.
type ContractFormationForm struct {
	BaseForm

	Version                     uint8   `json:"version,omitempty"`
	ContractName                string  `json:"contract_name,omitempty"`
	ContractFileHash            string  `json:"contract_file_hash,omitempty"`
	GoverningLaw                string  `json:"governing_law,omitempty"`
	Jurisdiction                string  `json:"jurisdiction,omitempty"`
	ContractExpiration          uint64  `json:"contract_expiration,omitempty"`
	URI                         string  `json:"uri,omitempty"`
	ContractRevision            uint16  `json:"contract_revision,omitempty"`
	IssuerID                    string  `json:"issuer_id,omitempty"`
	IssuerType                  string  `json:"issuer_type,omitempty"`
	ContractOperatorID          string  `json:"contract_operator_id,omitempty"`
	AuthorizationFlags          string  `json:"authorization_flags,omitempty"`
	VotingSystem                string  `json:"voting_system,omitempty"`
	InitiativeThreshold         float32 `json:"initiative_threshold,omitempty"`
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency,omitempty"`
	RestrictedQty               uint64  `json:"restricted_qty,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ContractFormationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ContractFormationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ContractFormationForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ContractFormationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewContractFormation()

	m.Version = f.Version

	m.ContractName, err = f.pad(f.ContractName, 32)
	if err != nil {
		return nil, err
	}

	m.ContractFileHash, err = f.pad(f.ContractFileHash, 32)
	if err != nil {
		return nil, err
	}

	m.GoverningLaw, err = f.pad(f.GoverningLaw, 5)
	if err != nil {
		return nil, err
	}

	m.Jurisdiction, err = f.pad(f.Jurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.ContractExpiration = f.ContractExpiration

	m.URI, err = f.pad(f.URI, 78)
	if err != nil {
		return nil, err
	}

	m.ContractRevision = f.ContractRevision

	m.IssuerID, err = f.pad(f.IssuerID, 16)
	if err != nil {
		return nil, err
	}

	m.IssuerType, err = f.ensureByte(f.IssuerType)
	if err != nil {
		return nil, err
	}

	m.ContractOperatorID, err = f.pad(f.ContractOperatorID, 16)
	if err != nil {
		return nil, err
	}

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.InitiativeThreshold = f.InitiativeThreshold

	m.InitiativeThresholdCurrency, err = f.pad(f.InitiativeThresholdCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.RestrictedQty = f.RestrictedQty

	return &m, nil
}

// ContractAmendmentForm is the JSON friendly version of a ContractAmendment.
type ContractAmendmentForm struct {
	BaseForm

	Version                     uint8   `json:"version,omitempty"`
	ContractName                string  `json:"contract_name,omitempty"`
	ContractFileHash            string  `json:"contract_file_hash,omitempty"`
	GoverningLaw                string  `json:"governing_law,omitempty"`
	Jurisdiction                string  `json:"jurisdiction,omitempty"`
	ContractExpiration          uint64  `json:"contract_expiration,omitempty"`
	URI                         string  `json:"uri,omitempty"`
	ContractRevision            uint16  `json:"contract_revision,omitempty"`
	IssuerID                    string  `json:"issuer_id,omitempty"`
	IssuerType                  string  `json:"issuer_type,omitempty"`
	ContractOperatorID          string  `json:"contract_operator_id,omitempty"`
	AuthorizationFlags          string  `json:"authorization_flags,omitempty"`
	VotingSystem                string  `json:"voting_system,omitempty"`
	InitiativeThreshold         float32 `json:"initiative_threshold,omitempty"`
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency,omitempty"`
	RestrictedQty               uint64  `json:"restricted_qty,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ContractAmendmentForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ContractAmendmentForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ContractAmendmentForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ContractAmendmentForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewContractAmendment()

	m.Version = f.Version

	m.ContractName, err = f.pad(f.ContractName, 32)
	if err != nil {
		return nil, err
	}

	m.ContractFileHash, err = f.pad(f.ContractFileHash, 32)
	if err != nil {
		return nil, err
	}

	m.GoverningLaw, err = f.pad(f.GoverningLaw, 5)
	if err != nil {
		return nil, err
	}

	m.Jurisdiction, err = f.pad(f.Jurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.ContractExpiration = f.ContractExpiration

	m.URI, err = f.pad(f.URI, 78)
	if err != nil {
		return nil, err
	}

	m.ContractRevision = f.ContractRevision

	m.IssuerID, err = f.pad(f.IssuerID, 16)
	if err != nil {
		return nil, err
	}

	m.IssuerType, err = f.ensureByte(f.IssuerType)
	if err != nil {
		return nil, err
	}

	m.ContractOperatorID, err = f.pad(f.ContractOperatorID, 16)
	if err != nil {
		return nil, err
	}

	m.AuthorizationFlags, err = f.pad(f.AuthorizationFlags, 2)
	if err != nil {
		return nil, err
	}

	m.VotingSystem, err = f.ensureByte(f.VotingSystem)
	if err != nil {
		return nil, err
	}

	m.InitiativeThreshold = f.InitiativeThreshold

	m.InitiativeThresholdCurrency, err = f.pad(f.InitiativeThresholdCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.RestrictedQty = f.RestrictedQty

	return &m, nil
}

// OrderForm is the JSON friendly version of a Order.
type OrderForm struct {
	BaseForm

	Version                uint8  `json:"version,omitempty"`
	AssetType              string `json:"asset_type,omitempty"`
	AssetID                string `json:"asset_id,omitempty"`
	ComplianceAction       string `json:"compliance_action,omitempty"`
	TargetAddress          string `json:"target_address,omitempty"`
	DepositAddress         string `json:"deposit_address,omitempty"`
	SupportingEvidenceHash string `json:"supporting_evidence_hash,omitempty"`
	Qty                    uint64 `json:"qty,omitempty"`
	Expiration             uint64 `json:"expiration,omitempty"`
	Message                string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *OrderForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f OrderForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f OrderForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f OrderForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewOrder()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.ComplianceAction, err = f.ensureByte(f.ComplianceAction)
	if err != nil {
		return nil, err
	}

	m.TargetAddress, err = f.pad(f.TargetAddress, 34)
	if err != nil {
		return nil, err
	}

	m.DepositAddress, err = f.pad(f.DepositAddress, 34)
	if err != nil {
		return nil, err
	}

	m.SupportingEvidenceHash, err = f.pad(f.SupportingEvidenceHash, 32)
	if err != nil {
		return nil, err
	}

	m.Qty = f.Qty

	m.Expiration = f.Expiration

	m.Message, err = f.pad(f.Message, 61)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// FreezeForm is the JSON friendly version of a Freeze.
type FreezeForm struct {
	BaseForm

	Version    uint8  `json:"version,omitempty"`
	AssetType  string `json:"asset_type,omitempty"`
	AssetID    string `json:"asset_id,omitempty"`
	Timestamp  uint64 `json:"timestamp,omitempty"`
	Qty        uint64 `json:"qty,omitempty"`
	Expiration uint64 `json:"expiration,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *FreezeForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f FreezeForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f FreezeForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f FreezeForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewFreeze()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Timestamp = f.Timestamp

	m.Qty = f.Qty

	m.Expiration = f.Expiration

	m.Message, err = f.pad(f.Message, 61)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ThawForm is the JSON friendly version of a Thaw.
type ThawForm struct {
	BaseForm

	Version   uint8  `json:"version,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	Timestamp uint64 `json:"timestamp,omitempty"`
	Qty       uint64 `json:"qty,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ThawForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ThawForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ThawForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ThawForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewThaw()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Timestamp = f.Timestamp

	m.Qty = f.Qty

	m.Message, err = f.pad(f.Message, 61)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ConfiscationForm is the JSON friendly version of a Confiscation.
type ConfiscationForm struct {
	BaseForm

	Version     uint8  `json:"version,omitempty"`
	AssetType   string `json:"asset_type,omitempty"`
	AssetID     string `json:"asset_id,omitempty"`
	Timestamp   uint64 `json:"timestamp,omitempty"`
	TargetsQty  uint64 `json:"targets_qty,omitempty"`
	DepositsQty uint64 `json:"deposits_qty,omitempty"`
	Message     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ConfiscationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ConfiscationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ConfiscationForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ConfiscationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewConfiscation()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Timestamp = f.Timestamp

	m.TargetsQty = f.TargetsQty

	m.DepositsQty = f.DepositsQty

	m.Message, err = f.pad(f.Message, 61)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ReconciliationForm is the JSON friendly version of a Reconciliation.
type ReconciliationForm struct {
	BaseForm

	Version          uint8  `json:"version,omitempty"`
	AssetType        string `json:"asset_type,omitempty"`
	AssetID          string `json:"asset_id,omitempty"`
	RefTxnID         string `json:"ref_txn_id,omitempty"`
	TargetAddressQty uint64 `json:"target_address_qty,omitempty"`
	Timestamp        uint64 `json:"timestamp,omitempty"`
	Message          string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ReconciliationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ReconciliationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ReconciliationForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ReconciliationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewReconciliation()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.RefTxnID, err = f.pad(f.RefTxnID, 32)
	if err != nil {
		return nil, err
	}

	m.TargetAddressQty = f.TargetAddressQty

	m.Timestamp = f.Timestamp

	m.Message, err = f.pad(f.Message, 61)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// InitiativeForm is the JSON friendly version of a Initiative.
type InitiativeForm struct {
	BaseForm

	Version              uint8  `json:"version,omitempty"`
	AssetType            string `json:"asset_type,omitempty"`
	AssetID              string `json:"asset_id,omitempty"`
	VoteType             string `json:"vote_type,omitempty"`
	VoteOptions          string `json:"vote_options,omitempty"`
	VoteMax              uint8  `json:"vote_max,omitempty"`
	VoteLogic            string `json:"vote_logic,omitempty"`
	ProposalDescription  string `json:"proposal_description,omitempty"`
	ProposalDocumentHash string `json:"proposal_document_hash,omitempty"`
	VoteCutOffTimestamp  uint64 `json:"vote_cut_off_timestamp,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *InitiativeForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f InitiativeForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f InitiativeForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f InitiativeForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewInitiative()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteType, err = f.ensureByte(f.VoteType)
	if err != nil {
		return nil, err
	}

	m.VoteOptions, err = f.pad(f.VoteOptions, 16)
	if err != nil {
		return nil, err
	}

	m.VoteMax = f.VoteMax

	m.VoteLogic, err = f.ensureByte(f.VoteLogic)
	if err != nil {
		return nil, err
	}

	m.ProposalDescription, err = f.pad(f.ProposalDescription, 82)
	if err != nil {
		return nil, err
	}

	m.ProposalDocumentHash, err = f.pad(f.ProposalDocumentHash, 32)
	if err != nil {
		return nil, err
	}

	m.VoteCutOffTimestamp = f.VoteCutOffTimestamp

	return &m, nil
}

// ReferendumForm is the JSON friendly version of a Referendum.
type ReferendumForm struct {
	BaseForm

	Version              uint8  `json:"version,omitempty"`
	AssetType            string `json:"asset_type,omitempty"`
	AssetID              string `json:"asset_id,omitempty"`
	VoteType             string `json:"vote_type,omitempty"`
	VoteOptions          string `json:"vote_options,omitempty"`
	VoteMax              uint8  `json:"vote_max,omitempty"`
	VoteLogic            string `json:"vote_logic,omitempty"`
	ProposalDescription  string `json:"proposal_description,omitempty"`
	ProposalDocumentHash string `json:"proposal_document_hash,omitempty"`
	VoteCutOffTimestamp  uint64 `json:"vote_cut_off_timestamp,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ReferendumForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ReferendumForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ReferendumForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ReferendumForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewReferendum()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteType, err = f.ensureByte(f.VoteType)
	if err != nil {
		return nil, err
	}

	m.VoteOptions, err = f.pad(f.VoteOptions, 16)
	if err != nil {
		return nil, err
	}

	m.VoteMax = f.VoteMax

	m.VoteLogic, err = f.ensureByte(f.VoteLogic)
	if err != nil {
		return nil, err
	}

	m.ProposalDescription, err = f.pad(f.ProposalDescription, 82)
	if err != nil {
		return nil, err
	}

	m.ProposalDocumentHash, err = f.pad(f.ProposalDocumentHash, 32)
	if err != nil {
		return nil, err
	}

	m.VoteCutOffTimestamp = f.VoteCutOffTimestamp

	return &m, nil
}

// VoteForm is the JSON friendly version of a Vote.
type VoteForm struct {
	BaseForm

	Version              uint8  `json:"version,omitempty"`
	AssetType            string `json:"asset_type,omitempty"`
	AssetID              string `json:"asset_id,omitempty"`
	VoteType             string `json:"vote_type,omitempty"`
	VoteOptions          string `json:"vote_options,omitempty"`
	VoteMax              uint8  `json:"vote_max,omitempty"`
	VoteLogic            string `json:"vote_logic,omitempty"`
	ProposalDescription  string `json:"proposal_description,omitempty"`
	ProposalDocumentHash string `json:"proposal_document_hash,omitempty"`
	VoteCutOffTimestamp  uint64 `json:"vote_cut_off_timestamp,omitempty"`
	Timestamp            uint64 `json:"timestamp,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *VoteForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f VoteForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f VoteForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f VoteForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewVote()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteType, err = f.ensureByte(f.VoteType)
	if err != nil {
		return nil, err
	}

	m.VoteOptions, err = f.pad(f.VoteOptions, 16)
	if err != nil {
		return nil, err
	}

	m.VoteMax = f.VoteMax

	m.VoteLogic, err = f.ensureByte(f.VoteLogic)
	if err != nil {
		return nil, err
	}

	m.ProposalDescription, err = f.pad(f.ProposalDescription, 82)
	if err != nil {
		return nil, err
	}

	m.ProposalDocumentHash, err = f.pad(f.ProposalDocumentHash, 32)
	if err != nil {
		return nil, err
	}

	m.VoteCutOffTimestamp = f.VoteCutOffTimestamp

	m.Timestamp = f.Timestamp

	return &m, nil
}

// BallotCastForm is the JSON friendly version of a BallotCast.
type BallotCastForm struct {
	BaseForm

	Version   uint8  `json:"version,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	VoteTxnID string `json:"vote_txn_id,omitempty"`
	Vote      string `json:"vote,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *BallotCastForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f BallotCastForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f BallotCastForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f BallotCastForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewBallotCast()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteTxnID, err = f.pad(f.VoteTxnID, 32)
	if err != nil {
		return nil, err
	}

	m.Vote, err = f.pad(f.Vote, 16)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// BallotCountedForm is the JSON friendly version of a BallotCounted.
type BallotCountedForm struct {
	BaseForm

	Version   uint8  `json:"version,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	VoteTxnID string `json:"vote_txn_id,omitempty"`
	Vote      string `json:"vote,omitempty"`
	Timestamp uint64 `json:"timestamp,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *BallotCountedForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f BallotCountedForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f BallotCountedForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f BallotCountedForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewBallotCounted()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteTxnID, err = f.pad(f.VoteTxnID, 32)
	if err != nil {
		return nil, err
	}

	m.Vote, err = f.pad(f.Vote, 16)
	if err != nil {
		return nil, err
	}

	m.Timestamp = f.Timestamp

	return &m, nil
}

// ResultForm is the JSON friendly version of a Result.
type ResultForm struct {
	BaseForm

	Version       uint8  `json:"version,omitempty"`
	AssetType     string `json:"asset_type,omitempty"`
	AssetID       string `json:"asset_id,omitempty"`
	VoteType      string `json:"vote_type,omitempty"`
	VoteTxnID     string `json:"vote_txn_id,omitempty"`
	Timestamp     uint64 `json:"timestamp,omitempty"`
	Option1Tally  uint64 `json:"option1_tally,omitempty"`
	Option2Tally  uint64 `json:"option2_tally,omitempty"`
	Option3Tally  uint64 `json:"option3_tally,omitempty"`
	Option4Tally  uint64 `json:"option4_tally,omitempty"`
	Option5Tally  uint64 `json:"option5_tally,omitempty"`
	Option6Tally  uint64 `json:"option6_tally,omitempty"`
	Option7Tally  uint64 `json:"option7_tally,omitempty"`
	Option8Tally  uint64 `json:"option8_tally,omitempty"`
	Option9Tally  uint64 `json:"option9_tally,omitempty"`
	Option10Tally uint64 `json:"option10_tally,omitempty"`
	Option11Tally uint64 `json:"option11_tally,omitempty"`
	Option12Tally uint64 `json:"option12_tally,omitempty"`
	Option13Tally uint64 `json:"option13_tally,omitempty"`
	Option14Tally uint64 `json:"option14_tally,omitempty"`
	Option15Tally uint64 `json:"option15_tally,omitempty"`
	Result        string `json:"result,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ResultForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ResultForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ResultForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ResultForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewResult()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.VoteType, err = f.ensureByte(f.VoteType)
	if err != nil {
		return nil, err
	}

	m.VoteTxnID, err = f.pad(f.VoteTxnID, 32)
	if err != nil {
		return nil, err
	}

	m.Timestamp = f.Timestamp

	m.Option1Tally = f.Option1Tally

	m.Option2Tally = f.Option2Tally

	m.Option3Tally = f.Option3Tally

	m.Option4Tally = f.Option4Tally

	m.Option5Tally = f.Option5Tally

	m.Option6Tally = f.Option6Tally

	m.Option7Tally = f.Option7Tally

	m.Option8Tally = f.Option8Tally

	m.Option9Tally = f.Option9Tally

	m.Option10Tally = f.Option10Tally

	m.Option11Tally = f.Option11Tally

	m.Option12Tally = f.Option12Tally

	m.Option13Tally = f.Option13Tally

	m.Option14Tally = f.Option14Tally

	m.Option15Tally = f.Option15Tally

	m.Result, err = f.pad(f.Result, 16)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// MessageForm is the JSON friendly version of a Message.
type MessageForm struct {
	BaseForm

	Version     uint8  `json:"version,omitempty"`
	Timestamp   uint64 `json:"timestamp,omitempty"`
	MessageType string `json:"message_type,omitempty"`
	Message     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *MessageForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f MessageForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f MessageForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f MessageForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewMessage()

	m.Version = f.Version

	m.Timestamp = f.Timestamp

	m.MessageType, err = f.pad(f.MessageType, 2)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 203)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// RejectionForm is the JSON friendly version of a Rejection.
type RejectionForm struct {
	BaseForm

	Version       uint8  `json:"version,omitempty"`
	Timestamp     uint64 `json:"timestamp,omitempty"`
	AssetType     string `json:"asset_type,omitempty"`
	AssetID       string `json:"asset_id,omitempty"`
	RejectionType string `json:"rejection_type,omitempty"`
	Message       string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *RejectionForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f RejectionForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f RejectionForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f RejectionForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewRejection()

	m.Version = f.Version

	m.Timestamp = f.Timestamp

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.RejectionType, err = f.ensureByte(f.RejectionType)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 169)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// EstablishmentForm is the JSON friendly version of a Establishment.
type EstablishmentForm struct {
	BaseForm

	Version                     uint8  `json:"version,omitempty"`
	Registrar                   string `json:"registrar,omitempty"`
	RegisterType                string `json:"register_type,omitempty"`
	KYCJurisdiction             string `json:"kyc_jurisdiction,omitempty"`
	DOB                         uint64 `json:"dob,omitempty"`
	CountryOfResidence          string `json:"country_of_residence,omitempty"`
	SupportingDocumentationHash string `json:"supporting_documentation_hash,omitempty"`
	Message                     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *EstablishmentForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f EstablishmentForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f EstablishmentForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f EstablishmentForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewEstablishment()

	m.Version = f.Version

	m.Registrar, err = f.pad(f.Registrar, 16)
	if err != nil {
		return nil, err
	}

	m.RegisterType, err = f.ensureByte(f.RegisterType)
	if err != nil {
		return nil, err
	}

	m.KYCJurisdiction, err = f.pad(f.KYCJurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.DOB = f.DOB

	m.CountryOfResidence, err = f.pad(f.CountryOfResidence, 3)
	if err != nil {
		return nil, err
	}

	m.SupportingDocumentationHash, err = f.pad(f.SupportingDocumentationHash, 32)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 148)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// AdditionForm is the JSON friendly version of a Addition.
type AdditionForm struct {
	BaseForm

	Version                     uint8  `json:"version,omitempty"`
	Sublist                     string `json:"sublist,omitempty"`
	KYC                         string `json:"kyc,omitempty"`
	KYCJurisdiction             string `json:"kyc_jurisdiction,omitempty"`
	DOB                         uint64 `json:"dob,omitempty"`
	CountryOfResidence          string `json:"country_of_residence,omitempty"`
	SupportingDocumentationHash string `json:"supporting_documentation_hash,omitempty"`
	Message                     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *AdditionForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f AdditionForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f AdditionForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f AdditionForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewAddition()

	m.Version = f.Version

	m.Sublist, err = f.pad(f.Sublist, 4)
	if err != nil {
		return nil, err
	}

	m.KYC, err = f.ensureByte(f.KYC)
	if err != nil {
		return nil, err
	}

	m.KYCJurisdiction, err = f.pad(f.KYCJurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.DOB = f.DOB

	m.CountryOfResidence, err = f.pad(f.CountryOfResidence, 3)
	if err != nil {
		return nil, err
	}

	m.SupportingDocumentationHash, err = f.pad(f.SupportingDocumentationHash, 32)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 148)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// AlterationForm is the JSON friendly version of a Alteration.
type AlterationForm struct {
	BaseForm

	Version                     uint8  `json:"version,omitempty"`
	Sublist                     string `json:"sublist,omitempty"`
	KYC                         string `json:"kyc,omitempty"`
	KYCJurisdiction             string `json:"kyc_jurisdiction,omitempty"`
	DOB                         uint64 `json:"dob,omitempty"`
	CountryOfResidence          string `json:"country_of_residence,omitempty"`
	SupportingDocumentationHash string `json:"supporting_documentation_hash,omitempty"`
	Message                     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *AlterationForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f AlterationForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f AlterationForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f AlterationForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewAlteration()

	m.Version = f.Version

	m.Sublist, err = f.pad(f.Sublist, 4)
	if err != nil {
		return nil, err
	}

	m.KYC, err = f.ensureByte(f.KYC)
	if err != nil {
		return nil, err
	}

	m.KYCJurisdiction, err = f.pad(f.KYCJurisdiction, 5)
	if err != nil {
		return nil, err
	}

	m.DOB = f.DOB

	m.CountryOfResidence, err = f.pad(f.CountryOfResidence, 3)
	if err != nil {
		return nil, err
	}

	m.SupportingDocumentationHash, err = f.pad(f.SupportingDocumentationHash, 32)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 160)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// RemovalForm is the JSON friendly version of a Removal.
type RemovalForm struct {
	BaseForm

	Version                     uint8  `json:"version,omitempty"`
	SupportingDocumentationHash string `json:"supporting_documentation_hash,omitempty"`
	Message                     string `json:"message,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *RemovalForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f RemovalForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f RemovalForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f RemovalForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewRemoval()

	m.Version = f.Version

	m.SupportingDocumentationHash, err = f.pad(f.SupportingDocumentationHash, 32)
	if err != nil {
		return nil, err
	}

	m.Message, err = f.pad(f.Message, 181)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// SendForm is the JSON friendly version of a Send.
type SendForm struct {
	BaseForm

	Version   uint8  `json:"version,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	TokenQty  uint64 `json:"token_qty,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *SendForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f SendForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f SendForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f SendForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewSend()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.TokenQty = f.TokenQty

	return &m, nil
}

// ExchangeForm is the JSON friendly version of a Exchange.
type ExchangeForm struct {
	BaseForm

	Version             uint8   `json:"version,omitempty"`
	Party1AssetType     string  `json:"party1_asset_type,omitempty"`
	Party1AssetID       string  `json:"party1_asset_id,omitempty"`
	Party1TokenQty      uint64  `json:"party1_token_qty,omitempty"`
	OfferValidUntil     uint64  `json:"offer_valid_until,omitempty"`
	ExchangeFeeCurrency string  `json:"exchange_fee_currency,omitempty"`
	ExchangeFeeVar      float32 `json:"exchange_fee_var,omitempty"`
	ExchangeFeeFixed    float32 `json:"exchange_fee_fixed,omitempty"`
	ExchangeFeeAddress  string  `json:"exchange_fee_address,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *ExchangeForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f ExchangeForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f ExchangeForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f ExchangeForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewExchange()

	m.Version = f.Version

	m.Party1AssetType, err = f.pad(f.Party1AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.Party1AssetID, err = f.pad(f.Party1AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Party1TokenQty = f.Party1TokenQty

	m.OfferValidUntil = f.OfferValidUntil

	m.ExchangeFeeCurrency, err = f.pad(f.ExchangeFeeCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.ExchangeFeeVar = f.ExchangeFeeVar

	m.ExchangeFeeFixed = f.ExchangeFeeFixed

	m.ExchangeFeeAddress, err = f.pad(f.ExchangeFeeAddress, 34)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// SwapForm is the JSON friendly version of a Swap.
type SwapForm struct {
	BaseForm

	Version             uint8   `json:"version,omitempty"`
	Party1AssetType     string  `json:"party1_asset_type,omitempty"`
	Party1AssetID       string  `json:"party1_asset_id,omitempty"`
	Party1TokenQty      uint64  `json:"party1_token_qty,omitempty"`
	OfferValidUntil     uint64  `json:"offer_valid_until,omitempty"`
	Party2AssetType     string  `json:"party2_asset_type,omitempty"`
	Party2AssetID       string  `json:"party2_asset_id,omitempty"`
	Party2TokenQty      uint64  `json:"party2_token_qty,omitempty"`
	ExchangeFeeCurrency string  `json:"exchange_fee_currency,omitempty"`
	ExchangeFeeVar      float32 `json:"exchange_fee_var,omitempty"`
	ExchangeFeeFixed    float32 `json:"exchange_fee_fixed,omitempty"`
	ExchangeFeeAddress  string  `json:"exchange_fee_address,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *SwapForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f SwapForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f SwapForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f SwapForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewSwap()

	m.Version = f.Version

	m.Party1AssetType, err = f.pad(f.Party1AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.Party1AssetID, err = f.pad(f.Party1AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Party1TokenQty = f.Party1TokenQty

	m.OfferValidUntil = f.OfferValidUntil

	m.Party2AssetType, err = f.pad(f.Party2AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.Party2AssetID, err = f.pad(f.Party2AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Party2TokenQty = f.Party2TokenQty

	m.ExchangeFeeCurrency, err = f.pad(f.ExchangeFeeCurrency, 3)
	if err != nil {
		return nil, err
	}

	m.ExchangeFeeVar = f.ExchangeFeeVar

	m.ExchangeFeeFixed = f.ExchangeFeeFixed

	m.ExchangeFeeAddress, err = f.pad(f.ExchangeFeeAddress, 34)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// SettlementForm is the JSON friendly version of a Settlement.
type SettlementForm struct {
	BaseForm

	Version        uint8  `json:"version,omitempty"`
	AssetType      string `json:"asset_type,omitempty"`
	AssetID        string `json:"asset_id,omitempty"`
	Party1TokenQty uint64 `json:"party1_token_qty,omitempty"`
	Party2TokenQty uint64 `json:"party2_token_qty,omitempty"`
	Timestamp      uint64 `json:"timestamp,omitempty"`
}

// Write implements the io.Writer interface, writing the data in []byte to
// the receiver.
func (f *SettlementForm) Write(b []byte) (n int, err error) {
	if err := json.Unmarshal(b, f); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Validate returns an error if validations fails.
func (f SettlementForm) Validate() error {
	return nil
}

// PayloadForm returns a PayloadForm for this Form, or nil if this Form has
// no matching PayloadForm.
func (f SettlementForm) PayloadForm() (PayloadForm, error) {
	return nil, nil
}

// BuildMessage returns an OpReturnMessage from the form.
func (f SettlementForm) BuildMessage() (OpReturnMessage, error) {
	var err error

	m := NewSettlement()

	m.Version = f.Version

	m.AssetType, err = f.pad(f.AssetType, 3)
	if err != nil {
		return nil, err
	}

	m.AssetID, err = f.pad(f.AssetID, 32)
	if err != nil {
		return nil, err
	}

	m.Party1TokenQty = f.Party1TokenQty

	m.Party2TokenQty = f.Party2TokenQty

	m.Timestamp = f.Timestamp

	return &m, nil
}
