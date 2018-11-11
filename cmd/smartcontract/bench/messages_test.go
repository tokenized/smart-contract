package bench

import (
	"testing"

	"github.com/tokenized/smart-contract/pkg/protocol"
)

var benchMessage protocol.OpReturnMessage

func BenchmarkParseTokenizedTX(b *testing.B) {
	tx := loadFixtureTX("ac7b6f91b458ea81d4bf3f8034ea737ee84f1db29893dbf8cf3f5c9753dd7bd0.txt")

	// run the function b.N times
	for n := 0; n < b.N; n++ {
		benchMessage = loadMessageFromTX(tx)
	}
}
