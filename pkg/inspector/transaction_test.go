package inspector

import (
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

func TestAddressesUnique(t *testing.T) {
	testArr := []struct {
		name string
		in   []bitcoin.Address
		want []bitcoin.Address
	}{
		{
			name: "one",
			in: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "two unique",
			in: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "two duplicate",
			in: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "2 x 2 duplicate",
			in: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "2 duplicates, 1 unique",
			in: []bitcoin.Address{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.Address{
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
