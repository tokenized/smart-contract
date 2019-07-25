package bitcoin

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/tokenized/smart-contract/pkg/wire"
)

func TestPKH(t *testing.T) {
	tests := []struct {
		pkhText string
		net     wire.BitcoinNet
		want    string
		err     error
	}{
		{
			pkhText: "4974a24418c676add75fc291fccf3e2253ceb21d",
			net:     MainNet,
			want:    "17hQ3sDD7yPNZ6Yjx4vMbkw5a14MhBf4wm",
			err:     nil,
		},
		{
			pkhText: "9e9aa671ec2258da2929d5ab146739fb9b43163f",
			net:     MainNet,
			want:    "1FTd2bKpCdxZAdY2zYqSNdLV8KkSWeof6X",
			err:     nil,
		},
		{
			pkhText: "e38be25f3ce0f3a275f35b2af8c52da6e4f89948",
			net:     MainNet,
			want:    "1Mk9xsjREmns4jxvkGAWVkPamx9fbpPtg5",
			err:     nil,
		},
		{
			pkhText: "f0e92f72817f2e7f737cbd9e79b9ad7a2185b216",
			net:     TestNet,
			want:    "n3Ump6TLXhB8Nv1z74VTr3xSYZMWd3Vtzn",
			err:     nil,
		},
		{
			pkhText: "1c42836b0707bf8c783bc43a942193c29efd99a8",
			net:     TestNet,
			want:    "mi6NtrV3DuaPLPPrtCYPb4h65fnhowk3Vn",
			err:     nil,
		},
		{
			pkhText: "27d5f776795d1af8c39e261c7a3fd60cbefcb25b",
			net:     TestNet,
			want:    "mj9ayCVe4Nt2ca8G1aPtMDeXYGACr9WJRm",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pkhText, func(t *testing.T) {
			pkh, err := hex.DecodeString(tt.pkhText)
			if err != nil {
				t.Fatal(err)
			}

			address, err := NewAddressPKH(pkh, tt.net)
			if err != nil {
				t.Fatal(err)
			}

			addressText := address.String()
			if addressText != tt.want {
				t.Fatalf("PKH Address text invalid\ngot:%s\nwant:%s", addressText, tt.want)
			}

			a, err := DecodeAddressString(addressText)
			if err != nil {
				t.Fatal(err)
			}

			if a.Network() != tt.net {
				t.Fatal("PKH decoded wrong net")
			}

			apkh, ok := a.(*AddressPKH)
			if !ok {
				t.Fatal("PKH decoded wrong address type")
			}

			if !bytes.Equal(apkh.PKH(), pkh) {
				t.Fatalf("PKH decode invalid\ngot:%x\nwant:%x", apkh.PKH(), pkh)
			}

			// Locking script
			script := address.LockingScript()

			if len(script) != 25 {
				t.Fatalf("Invalid PKH locking script generated : %x", script)
			}

			scriptAddress, err := AddressFromLockingScript(script, tt.net)
			if err != nil {
				t.Fatalf("Failed to parse PKH locking script : %s", err.Error())
			}

			spkh, ok := scriptAddress.(*AddressPKH)
			if !ok {
				t.Fatal("PKH parse script wrong address type")
			}

			if !bytes.Equal(spkh.PKH(), pkh) {
				t.Fatalf("PKH parse script invalid\ngot:%x\nwant:%x", spkh.PKH(), pkh)
			}
		})
	}
}

func TestSH(t *testing.T) {
	tests := []struct {
		pkhText string
		net     wire.BitcoinNet
		want    string
		err     error
	}{
		{
			pkhText: "b5c127a49beb315d9a3a3c2c0b5380e80003618d",
			net:     MainNet,
			want:    "3JG3cu8qcFNUr9KzG8Ykbi83Wfy79hDDZy",
			err:     nil,
		},
		{
			pkhText: "a9d36e840b7bf6b8230d048ec115d191dc62ff0b",
			net:     MainNet,
			want:    "3HAyQVgVkEcVoScwX9tarNgFXzp5roiABD",
			err:     nil,
		},
		{
			pkhText: "611b51efc8bc5869a93bcb02076500efb0b335dd",
			net:     MainNet,
			want:    "3AYUCP9fhaeTZa1XmiTtN6agnsrsbjCyKv",
			err:     nil,
		},
		{
			pkhText: "b8a9bcb2cfbc4e318d5d1d1f718e95b8fac07da3",
			net:     TestNet,
			want:    "2NA5df7zpLBBhPNhr9BwgzHtzdso9JUz1GT",
			err:     nil,
		},
		{
			pkhText: "a45ae400072e28d7b4baa4b8c83d761518b23c53",
			net:     TestNet,
			want:    "2N8EFgwDJcbjL7VxpREZYYDXVHnf2e2DNqc",
			err:     nil,
		},
		{
			pkhText: "18e076de3b91562c27656a170de97c82f200b7c7",
			net:     TestNet,
			want:    "2MuWm6wdypGbqwnvWHDcKg2ehi8m9j2xmDj",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pkhText, func(t *testing.T) {
			sh, err := hex.DecodeString(tt.pkhText)
			if err != nil {
				t.Fatal(err)
			}

			address, err := NewAddressSH(sh, tt.net)
			if err != nil {
				t.Fatal(err)
			}

			addressText := address.String()
			if addressText != tt.want {
				t.Fatalf("SH Address text invalid\ngot:%s\nwant:%s", addressText, tt.want)
			}

			a, err := DecodeAddressString(addressText)
			if err != nil {
				t.Fatal(err)
			}

			if a.Network() != tt.net {
				t.Fatal("PKH decoded wrong net")
			}

			ash, ok := a.(*AddressSH)
			if !ok {
				t.Fatal("PKH decoded wrong address type")
			}

			if !bytes.Equal(ash.SH(), sh) {
				t.Fatalf("PKH decode invalid\ngot:%x\nwant:%x", ash.SH(), sh)
			}

			// Locking script
			script := address.LockingScript()

			if len(script) != 23 {
				t.Fatalf("Invalid SH locking script generated : %x", script)
			}

			scriptAddress, err := AddressFromLockingScript(script, tt.net)
			if err != nil {
				t.Fatalf("Failed to parse SH locking script : %s", err.Error())
			}

			spkh, ok := scriptAddress.(*AddressSH)
			if !ok {
				t.Fatal("SH parse script wrong address type")
			}

			if !bytes.Equal(spkh.SH(), sh) {
				t.Fatalf("SH parse script invalid\ngot:%x\nwant:%x", spkh.SH(), sh)
			}
		})
	}
}

// TODO func TestMultiPKH(t *testing.T) {
// TODO func TestRPuzzle(t *testing.T) {
