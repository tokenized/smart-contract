package inspector

import (
	"reflect"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
)

func TestAddressesUnique(t *testing.T) {
	testArr := []struct {
		name string
		in   []bitcoin.RawAddress
		want []bitcoin.RawAddress
	}{
		{
			name: "one",
			in: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "two unique",
			in: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "two duplicate",
			in: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
			want: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
			},
		},
		{
			name: "2 x 2 duplicate",
			in: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
		},
		{
			name: "2 duplicates, 1 unique",
			in: []bitcoin.RawAddress{
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1ERCtpGiBANVTHQk9guT6KpHiYcopTrCYu"),
				decodeAddress("1L8eJq8yAHsbByVvYVLbx4YEXZadRJHJWk"),
			},
			want: []bitcoin.RawAddress{
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
