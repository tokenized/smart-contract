package txbuilder

import "testing"

func TestConvertBCHToSatoshis(t *testing.T) {
	tests := []struct {
		name string
		bch  float32
		want uint64
	}{
		{
			name: "1234",
			bch:  0.00001234,
			want: 1234,
		},
		{
			name: "1",
			bch:  0.00000001,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertBCHToSatoshis(tt.bch)

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
