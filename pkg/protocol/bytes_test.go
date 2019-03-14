package protocol

import (
	"bytes"
	"testing"
)

var tests = []struct {
	size   uint64
	script []byte
}{
	// Single byte pushes (op code is 8 bit integer representing size to push)
	{0, []byte{0}},
	{10, []byte{10}},
	{0x4b, []byte{0x4b}},

	// OP_PUSHDATA1 (push code 0x4c followed by 1 byte for size)
	{0x4c, []byte{0x4c, 0x4c}},
	{0x50, []byte{0x4c, 0x50}},

	// OP_PUSHDATA2 (push code 0x4d followed by 2 bytes for size)
	{0x1050, []byte{0x4d, 0x10, 0x50}},
	{0xff50, []byte{0x4d, 0xff, 0x50}},

	// OP_PUSHDATA4 (push code 0x4e followed by 4 bytes for size)
	{0x0010ff50, []byte{0x4e, 0x00, 0x10, 0xff, 0x50}},
	{0x00ffff50, []byte{0x4e, 0x00, 0xff, 0xff, 0x50}},
}

func TestPushDataScript(t *testing.T) {
	for i, test := range tests {
		result := PushDataScript(test.size)
		if !bytes.Equal(result, test.script) {
			t.Fatalf("Failed test %d :\nResult : %+v\nCorrect : %+v\n", i, result, test.script)
		}
	}
}

func TestParsePushDataScript(t *testing.T) {
	for i, test := range tests {
		buf := bytes.NewBuffer(test.script)
		result, err := ParsePushDataScript(buf)
		if err != nil {
			t.Fatalf("Failed test %d : %s", i, err)
		}
		if result != test.size {
			t.Fatalf("Failed test %d :\nResult : %d\nCorrect : %d\n", i, result, test.size)
		}
	}
}
