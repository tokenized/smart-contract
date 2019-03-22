package txbuilder

import "fmt"

const (
	ErrorCodeInsufficientValue = 1
	ErrorCodeWrongPrivateKey   = 2
	ErrorCodeMissingPrivateKey = 3
)

func IsErrorCode(err error, code int) bool {
	er, ok := err.(*txBuilderError)
	if !ok {
		return false
	}
	return er.code == code
}

type txBuilderError struct {
	code    int
	message string
}

func (err *txBuilderError) Error() string {
	if len(err.message) == 0 {
		return errorCodeString(err.code)
	}
	return fmt.Sprintf("%s : %s", errorCodeString(err.code), err.message)
}

func errorCodeString(code int) string {
	switch code {
	case ErrorCodeInsufficientValue:
		return "Insufficient Value"
	case ErrorCodeWrongPrivateKey:
		return "Wrong Private Key"
	case ErrorCodeMissingPrivateKey:
		return "Missing Private Key"
	default:
		return "Unknown Error Code"
	}
}

func newError(code int, message string) *txBuilderError {
	result := txBuilderError{code: code, message: message}
	return &result
}
