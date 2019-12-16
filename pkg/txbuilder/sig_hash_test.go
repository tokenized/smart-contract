package txbuilder

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/tokenized/smart-contract/pkg/wire"
)

func TestSigHash(t *testing.T) {
	tests := []struct {
		hex_transaction string
		script          string
		input_index     int
		value           uint64
		hashType        int
		signature_hash  string
	}{
		{"907c2bc503ade11cc3b04eb2918b6f547b0630ab569273824748c87ea14b0696526c66ba740200000004ab65ababfd1f9bdd4ef073c7afc4ae00da8a66f429c917a0081ad1e1dabce28d373eab81d8628de802000000096aab5253ab52000052ad042b5f25efb33beec9f3364e8a9139e8439d9d7e26529c3c30b6c3fd89f8684cfd68ea0200000009ab53526500636a52ab599ac2fe02a526ed040000000008535300516352515164370e010000000003006300ab2ec229", "", 2, 123, 1864164639, "ac37112171153d11b4ef51bde62599a5c4636cbe6b22afeb469de80a3d3c346a"},
		{"a0aa3126041621a6dea5b800141aa696daf28408959dfb2df96095db9fa425ad3f427f2f6103000000015360290e9c6063fa26912c2e7fb6a0ad80f1c5fea1771d42f12976092e7a85a4229fdb6e890000000001abc109f6e47688ac0e4682988785744602b8c87228fcef0695085edf19088af1a9db126e93000000000665516aac536affffffff8fe53e0806e12dfd05d67ac68f4768fdbe23fc48ace22a5aa8ba04c96d58e2750300000009ac51abac63ab5153650524aa680455ce7b000000000000499e50030000000008636a00ac526563ac5051ee030000000003abacabd2b6fe000000000003516563910fb6b5", "65", 0, 84626, -1391424484, "c2e5d07178ae69868741fb99710a9356009dbb7b280afb0ac0b0c8c957f5c0f0"},
		{"6e7e9d4b04ce17afa1e8546b627bb8d89a6a7fefd9d892ec8a192d79c2ceafc01694a6a7e7030000000953ac6a51006353636a33bced1544f797f08ceed02f108da22cd24c9e7809a446c61eb3895914508ac91f07053a01000000055163ab516affffffff11dc54eee8f9e4ff0bcf6b1a1a35b1cd10d63389571375501af7444073bcec3c02000000046aab53514a821f0ce3956e235f71e4c69d91abe1e93fb703bd33039ac567249ed339bf0ba0883ef300000000090063ab65000065ac654bec3cc504bcf499020000000005ab6a52abac64eb060100000000076a6a5351650053bbbc130100000000056a6aab53abd6e1380100000000026a51c4e509b8", "acab655151", 0, 943837271, 479279909, "cbfd22524e78bb72007600e8e5a5264f3bbebb3ced8eda07440b8434e42baeda"},
		{"73107cbd025c22ebc8c3e0a47b2a760739216a528de8d4dab5d45cbeb3051cebae73b01ca10200000007ab6353656a636affffffffe26816dffc670841e6a6c8c61c586da401df1261a330a6c6b3dd9f9a0789bc9e000000000800ac6552ac6aac51ffffffff0174a8f0010000000004ac52515100000000", "5163ac63635151ac", 1, 9351684, 1190874345, "76ea6155aad3a53b3a4b00aac404ec596185b6f199e605b4ca242f16aab28af7"},
		{"e93bbf6902be872933cb987fc26ba0f914fcfc2f6ce555258554dd9939d12032a8536c8802030000000453ac5353eabb6451e074e6fef9de211347d6a45900ea5aaf2636ef7967f565dce66fa451805c5cd10000000003525253ffffffff047dc3e6020000000007516565ac656aabec9eea010000000001633e46e600000000000015080a030000000001ab00000000", "5300ac6a53ab6a", 1, 1, -886562767, "bb5754484700a4ea35e9e09b58256fe6ac09d582f0cf222afe1bceb0c5531458"},
		{"50818f4c01b464538b1e7e7f5ae4ed96ad23c68c830e78da9a845bc19b5c3b0b20bb82e5e9030000000763526a63655352ffffffff023b3f9c040000000008630051516a6a5163a83caf01000000000553ab65510000000000", "6aac", 0, 756, 946795545, "097126c0342f2513255ebf77b4948ebaab25aa37b926b9f78fe70db55e5410ce"},
	}

	for _, tt := range tests {
		t.Run(tt.signature_hash, func(t *testing.T) {
			txData, err := hex.DecodeString(tt.hex_transaction)
			if err != nil {
				t.Fatalf("Failed to decode tx hex : %s\n", err)
			}

			var tx wire.MsgTx
			txBuf := bytes.NewBuffer(txData)
			if err := tx.Deserialize(txBuf); err != nil {
				t.Fatalf("Failed to deserialize tx : %s\n", err)
			}

			script, err := hex.DecodeString(tt.script)
			if err != nil {
				t.Fatalf("Failed to decode script hex : %s\n", err)
			}

			wantHash, err := hex.DecodeString(tt.signature_hash)
			if err != nil {
				t.Fatalf("Failed to decode signature hash hex : %s\n", err)
			}

			var hashCache SigHashCache
			gotHash, err := signatureHash(&tx, tt.input_index, script, tt.value,
				SigHashType(tt.hashType), &hashCache)
			if err != nil {
				t.Fatalf("Failed to generate signature hash : %s\n", err)
			}

			if !bytes.Equal(wantHash, gotHash) {
				t.Fatalf("Invalid Sig Hash\ngot:%x\nwant:%x", gotHash, wantHash)
			}
		})
	}
}
