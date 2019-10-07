package node

import (
	"fmt"

	"github.com/tokenized/specification/dist/golang/actions"
)

// IsErrorCode returns true if the error is a NodeError and it matches the corresponding protocol
//   rejection code.
func IsErrorCode(err error, code uint32) bool {
	er, ok := err.(*NodeError)
	if !ok {
		return false
	}
	return er.code == code
}

// ErrorCode returns the error code of the error and true if the error is a NodeError.
// Otherwise it returns 0xffffffff and false
func ErrorCode(err error) (uint32, bool) {
	er, ok := err.(*NodeError)
	if !ok {
		return 0xffffffff, false
	}
	return er.code, true
}

type NodeError struct {
	code    uint32
	message string
}

func (err *NodeError) Error() string {
	if len(err.message) == 0 {
		return errorCodeString(err.code)
	}
	return fmt.Sprintf("%s : %s", errorCodeString(err.code), err.message)
}

func errorCodeString(code uint32) string {
	data := actions.RejectionsData(code)
	if data != nil {
		return data.Name
	}
	return "Unknown Error Code"
}

func NewError(code uint32, message string) *NodeError {
	result := NodeError{code: code, message: message}
	return &result
}
