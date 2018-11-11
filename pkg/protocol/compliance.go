package protocol

const (
	// ComplianceActionFreeze indicates a Freeze Order
	ComplianceActionFreeze = byte('F')

	// ComplianceActionThaw indicates a Thaw Order
	ComplianceActionThaw = byte('T')

	// ComplianceActionConfiscation indicates a Confiscation Order
	ComplianceActionConfiscation = byte('C')
)
