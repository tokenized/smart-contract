package txbuilder

import (
	"reflect"
	"testing"

	"github.com/btcsuite/btcutil"
)

func TestAddressSet_Set(t *testing.T) {
	tests := []struct {
		name      string
		addresses []btcutil.Address
		want      []btcutil.Address
	}{
		{
			name: "one address",
			addresses: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
			},
			want: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
			},
		},
		{
			name: "one address repeated",
			addresses: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
			},
			want: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
			},
		},
		{
			name: "one address repeated, with 2nd address",
			addresses: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
				decodeAddress("1L9Vr7BCEeczDtSJiX3fHLG5VVQgHtB22o"),
			},
			want: []btcutil.Address{
				decodeAddress("18H59cUZMAPRhp74xoeE6LXingw3Wxr3VG"),
				decodeAddress("1L9Vr7BCEeczDtSJiX3fHLG5VVQgHtB22o"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := AddressSet{
				Addresses: tt.addresses,
			}

			got := set.Set()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
