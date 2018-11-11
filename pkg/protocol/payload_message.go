package protocol

import "fmt"

// NewPayloadMessageFromCode returns the approriate PayloadMessage for the
// given code.
func NewPayloadMessageFromCode(code []byte) (PayloadMessage, error) {
	s := string(code)
	switch s {
	case CodeAssetTypeShareCommon:
		return NewAssetTypeShareCommon(), nil
	}

	return nil, fmt.Errorf("No asset type for code %s", code)
}
