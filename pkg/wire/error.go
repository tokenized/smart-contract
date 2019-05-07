// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

// MessageError describes an issue with a message.
// An example of some potential issues are messages from the wrong bitcoin
// network, invalid commands, mismatched checksums, and exceeding max payloads.
//
// This provides a mechanism for the caller to type assert the error to
// differentiate between general io errors such as io.EOF and issues that
// resulted from malformed messages.
type MessageError struct {
	Func        string // Function name
	Type        int
	Description string // Human readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e *MessageError) Error() string {
	result := ""
	if len(e.Func) > 0 {
		result += e.Func + " : "
	}
	typeName := messageErrorTypeName(e.Type)
	if len(typeName) > 0 {
		result += typeName
		if len(e.Description) > 0 {
			result += " : " + e.Description
		}
	} else {
		result += e.Description
	}
	return result
}

// messageError creates an error for the given function and description.
func messageError(f string, desc string) *MessageError {
	return &MessageError{Func: f, Type: MessageErrorUndefined, Description: desc}
}

// messageTypeError creates an error for the given function, type, and description.
func messageTypeError(f string, t int, desc string) *MessageError {
	return &MessageError{Func: f, Type: t, Description: desc}
}

const (
	MessageErrorUndefined        = 0
	MessageErrorConnectionClosed = 1
	MessageErrorWrongNetwork     = 2
	MessageErrorUnknownCommand   = 3
)

func messageErrorTypeName(t int) string {
	switch t {
	case MessageErrorUndefined:
		return ""
	case MessageErrorConnectionClosed:
		return "Connection Closed"
	case MessageErrorWrongNetwork:
		return "Wrong Network"
	case MessageErrorUnknownCommand:
		return "Unknown Command"
	default:
		return ""
	}
}
