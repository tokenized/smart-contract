package txbuilder

import (
	"encoding/hex"
	"testing"
)

func TestBuildUnsignedHardCoded(t *testing.T) {
	b, err := buildUnsignedHardCoded()
	if err != nil {
		t.Fatal(err)
	}

	got := hex.EncodeToString(b)

	want := "0200000002072591a23c39f55750fb17333f1df700c4efde47c1bdff63f4e5ef822c9c08cd0100000000ffffffff905b38da9a969072f18d9024d7b16eb4b255b005edacfadac0510487ce576fe70100000000ffffffff0122020000000000001976a9145389667241a2bb9a8665a463531171e50a7f058588ac00000000"

	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", string(got), want)
	}
}
