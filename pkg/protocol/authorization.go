package protocol

import (
	"bytes"

	"github.com/dgryski/go-bitstream"
	"github.com/pkg/errors"
)

// Permission represents the permissions assigned to a specific field.
type Permission struct {
	Permitted     bool
	Referendum    bool
	Initiative    bool
	VotingSystems []bool
}

// ReadAuthFlags reads raw auth flag data into an array of permission structs.
func ReadAuthFlags(authFlags []byte, fields, votingSystems int) ([]Permission, error) {
	result := make([]Permission, 0)
	buf := bytes.NewBuffer(authFlags)
	stream := bitstream.NewReader(buf)
	for i := 0; i < fields; i++ {
		var newPermission Permission

		// Permitted
		bit, err := stream.ReadBit()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read auth flags")
		}
		newPermission.Permitted = bool(bit)

		// Referendum
		bit, err = stream.ReadBit()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read auth flags")
		}
		newPermission.Referendum = bool(bit)

		// Initiative
		bit, err = stream.ReadBit()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read auth flags")
		}
		newPermission.Initiative = bool(bit)

		// Voting Systems
		if newPermission.Referendum || newPermission.Initiative {
			for j := 0; j < votingSystems; j++ {
				bit, err = stream.ReadBit()
				if err != nil {
					return nil, errors.Wrap(err, "Failed to read auth flags")
				}
				newPermission.VotingSystems = append(newPermission.VotingSystems, bool(bit))
			}
		}

		result = append(result, newPermission)
	}

	return result, nil
}

// WriteAuthFlags writes an array of permission structs into raw auth flag data.
func WriteAuthFlags(permissions []Permission) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	stream := bitstream.NewWriter(&buf)

	for _, permission := range permissions {
		// Permitted
		err = stream.WriteBit(bitstream.Bit(permission.Permitted))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to write auth flags")
		}

		// Referendum
		err = stream.WriteBit(bitstream.Bit(permission.Referendum))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to write auth flags")
		}

		// Initiative
		err = stream.WriteBit(bitstream.Bit(permission.Initiative))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to write auth flags")
		}

		// Voting Systems
		if permission.Referendum || permission.Initiative {
			for _, votingSystem := range permission.VotingSystems {
				err = stream.WriteBit(bitstream.Bit(votingSystem))
				if err != nil {
					return nil, errors.Wrap(err, "Failed to write auth flags")
				}
			}
		}
	}

	err = stream.Flush(bitstream.Zero)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to write auth flags")
	}
	return buf.Bytes(), nil
}
