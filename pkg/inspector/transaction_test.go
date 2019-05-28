package inspector

import (
	"reflect"
	"testing"

	"github.com/btcsuite/btcutil"
)

func TestAddressesUnique(t *testing.T) {
	testArr := []struct {
		name string
		in   []btcutil.Address
		want []btcutil.Address
	}{
		{
			name: "one",
			in: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "two unique",
			in: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "two duplicate",
			in: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "2 x 2 duplicate",
			in: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "2 duplicates, 1 unique",
			in: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []btcutil.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
	}

	for _, tt := range testArr {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueAddresses(tt.in)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}
