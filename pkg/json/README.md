
# Rationale

This `json` package is a fork of the standard golang `encoding/json`. The standard package encodes standard binary data as base64. This conflicts with most languages which encode standard binary data as hex. Since JSON messages are often used to communicate with external systems implemented in other languages, this causes issues. In order to improve compatibility between languages this package has been modified to encode standard binary data as hex.

In the file encode.go, in the function `encodeByteSlice`, the base64 encoding functions have been changed to use `encoding/hex`.

In the file decode.go, in the function `literalStore` within the `string` case, and under the `reflect.Slice` case, the base64 decoding functions have been changed to use `encoding/hex`.

Tests also had to be modified to account for these changes.
