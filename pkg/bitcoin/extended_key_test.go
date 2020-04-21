package bitcoin

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	bip32 "github.com/tyler-smith/go-bip32"
)

func TestExtendedKeyVsBIP0032(t *testing.T) {
	tests := []struct {
		name   string
		seed   string
		path   []uint32
		oldKey string
		oldPub string
	}{
		{
			name:   "BIP-0032 Test Vector 1.1",
			seed:   "fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			path:   nil, // m
			oldKey: "xprv9s21ZrQH143K31xYSDQpPDxsXRTUcvj2iNHm5NUtrGiGG5e2DtALGdso3pGz6ssrdK4PFmM8NSpSBHNqPqm55Qn3LqFtT2emdEXVYsCzC2U",
			oldPub: "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB",
		},
		{
			name:   "BIP-0032 Test Vector 1.2",
			seed:   "fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			path:   []uint32{0}, // m/0
			oldKey: "xprv9vHkqa6EV4sPZHYqZznhT2NPtPCjKuDKGY38FBWLvgaDx45zo9WQRUT3dKYnjwih2yJD9mkrocEZXo1ex8G81dwSM1fwqWpWkeS3v86pgKt",
			oldPub: "xpub69H7F5d8KSRgmmdJg2KhpAK8SR3DjMwAdkxj3ZuxV27CprR9LgpeyGmXUbC6wb7ERfvrnKZjXoUmmDznezpbZb7ap6r1D3tgFxHmwMkQTPH",
		},
		{
			name:   "BIP-0032 Test Vector 1.3",
			seed:   "fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542",
			path:   []uint32{0, Hardened + 2147483647}, // m/0/2147483647'
			oldKey: "xprv9wSp6B7kry3Vj9m1zSnLvN3xH8RdsPP1Mh7fAaR7aRLcQMKTR2vidYEeEg2mUCTAwCd6vnxVrcjfy2kRgVsFawNzmjuHc2YmYRmagcEPdU9",
			oldPub: "xpub6ASAVgeehLbnwdqV6UKMHVzgqAG8Gr6riv3Fxxpj8ksbH9ebxaEyBLZ85ySDhKiLDBrQSARLq1uNRts8RuJiHjaDMBU4Zn9h8LZNnBC5y4a",
		},
		{
			name:   "BIP-0032 Test Vector 2.1",
			seed:   "000102030405060708090a0b0c0d0e0f",
			path:   nil, // m
			oldKey: "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi",
			oldPub: "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8",
		},
		{
			name:   "BIP-0032 Test Vector 2.2",
			seed:   "000102030405060708090a0b0c0d0e0f",
			path:   []uint32{Hardened}, // m/0'
			oldKey: "xprv9uHRZZhk6KAJC1avXpDAp4MDc3sQKNxDiPvvkX8Br5ngLNv1TxvUxt4cV1rGL5hj6KCesnDYUhd7oWgT11eZG7XnxHrnYeSvkzY7d2bhkJ7",
			oldPub: "xpub68Gmy5EdvgibQVfPdqkBBCHxA5htiqg55crXYuXoQRKfDBFA1WEjWgP6LHhwBZeNK1VTsfTFUHCdrfp1bgwQ9xv5ski8PX9rL2dZXvgGDnw",
		},
		{
			name:   "BIP-0032 Test Vector 2.3",
			seed:   "000102030405060708090a0b0c0d0e0f",
			path:   []uint32{Hardened, 1}, // m/0'/1
			oldKey: "xprv9wTYmMFdV23N2TdNG573QoEsfRrWKQgWeibmLntzniatZvR9BmLnvSxqu53Kw1UmYPxLgboyZQaXwTCg8MSY3H2EU4pWcQDnRnrVA1xe8fs",
			oldPub: "xpub6ASuArnXKPbfEwhqN6e3mwBcDTgzisQN1wXN9BJcM47sSikHjJf3UFHKkNAWbWMiGj7Wf5uMash7SyYq527Hqck2AxYysAA7xmALppuCkwQ",
		},
		{
			name:   "BIP-0032 Test Vector 2.4",
			seed:   "000102030405060708090a0b0c0d0e0f",
			path:   []uint32{Hardened, 1, Hardened + 2}, // m/0'/1/2'
			oldKey: "xprv9z4pot5VBttmtdRTWfWQmoH1taj2axGVzFqSb8C9xaxKymcFzXBDptWmT7FwuEzG3ryjH4ktypQSAewRiNMjANTtpgP4mLTj34bhnZX7UiM",
			oldPub: "xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5",
		},
		{
			name:   "BIP-0032 Test Vector 3.1",
			seed:   "4b381541583be4423346c643850da4b320e46a87ae3d2a4e6da11eba819cd4acba45d239319ac14f863b8d5ab5a0d0c64d2e8a1e7d1457df2e5a3c51c73235be",
			path:   nil, // m
			oldKey: "xprv9s21ZrQH143K25QhxbucbDDuQ4naNntJRi4KUfWT7xo4EKsHt2QJDu7KXp1A3u7Bi1j8ph3EGsZ9Xvz9dGuVrtHHs7pXeTzjuxBrCmmhgC6",
			oldPub: "xpub661MyMwAqRbcEZVB4dScxMAdx6d4nFc9nvyvH3v4gJL378CSRZiYmhRoP7mBy6gSPSCYk6SzXPTf3ND1cZAceL7SfJ1Z3GC8vBgp2epUt13",
		},
		{
			name:   "BIP-0032 Test Vector 3.2",
			seed:   "4b381541583be4423346c643850da4b320e46a87ae3d2a4e6da11eba819cd4acba45d239319ac14f863b8d5ab5a0d0c64d2e8a1e7d1457df2e5a3c51c73235be",
			path:   []uint32{Hardened}, // m/0'
			oldKey: "xprv9uPDJpEQgRQfDcW7BkF7eTya6RPxXeJCqCJGHuCJ4GiRVLzkTXBAJMu2qaMWPrS7AANYqdq6vcBcBUdJCVVFceUvJFjaPdGZ2y9WACViL4L",
			oldPub: "xpub68NZiKmJWnxxS6aaHmn81bvJeTESw724CRDs6HbuccFQN9Ku14VQrADWgqbhhTHBaohPX4CjNLf9fq9MYo6oDaPPLPxSb7gwQN3ih19Zm4Y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed, err := hex.DecodeString(tt.seed)
			if err != nil {
				t.Fatalf("Failed decode key hex : %s", err)
			}

			xkey, err := LoadMasterExtendedKey(seed)
			if err != nil {
				t.Fatalf("Failed to load key : %s", err)
			}
			t.Logf("Master Key : %s", xkey.String())

			t.Logf("Deriving child %v", tt.path)
			child, err := xkey.ChildKeyForPath(tt.path)
			if err != nil {
				t.Fatalf("Failed to generate child key : %s", err)
			}
			t.Logf("Key : %s", child.String())

			if child.KeyValue[0] != 0 {
				t.Fatalf("Child key not private")
			}

			pubKey := child.ExtendedPublicKey()
			t.Logf("Public Key : %s", pubKey.String())

			oldKey, err := bip32.B58Deserialize(tt.oldKey)
			if err != nil {
				t.Fatalf("Failed to load old key : %s", err)
			}
			// t.Logf("Old Key : %s", oldKey.String())

			if !bytes.Equal(child.ChainCode[:], oldKey.ChainCode) {
				t.Fatalf("Old xprv chain doesn't match :\n  got  : %x\n  want : %x\n", child.ChainCode[:],
					oldKey.ChainCode)
			}
			if !bytes.Equal(child.KeyValue[1:], oldKey.Key) {
				t.Fatalf("Old xprv key doesn't match :\n  got  : %x\n  want : %x\n", child.KeyValue[1:],
					oldKey.Key)
			}

			oldPub, err := bip32.B58Deserialize(tt.oldPub)
			if err != nil {
				t.Fatalf("Failed to load old key : %s", err)
			}
			// t.Logf("Old Pub : %s", oldPub.String())

			if !bytes.Equal(pubKey.ChainCode[:], oldPub.ChainCode) {
				t.Fatalf("Old xpub chain doesn't match :\n  got  : %x\n  want : %x\n", pubKey.ChainCode[:],
					oldPub.ChainCode)
			}
			if !bytes.Equal(pubKey.KeyValue[:], oldPub.Key) {
				t.Fatalf("Old xpub key doesn't match :\n  got  : %x\n  want : %x\n", pubKey.KeyValue[:],
					oldPub.Key)
			}
		})
	}
}

func TestExtendedKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		path     []uint32
		childKey string
		childPub string
	}{
		{
			name:     "BIP-0032 Test Vector 1.1",
			key:      "bitcoin-xkey:01004000000000000000000060499f801b896d83179a4374aeb7822aaeaceaa0db1f85ee3e904c4defbd9689004b03d6fc340455b363f51020ad3ecca4f0850280cf436c70c727923f6db46c3e338cdbb2",
			path:     nil, // m
			childKey: "bitcoin-xkey:01004000000000000000000060499f801b896d83179a4374aeb7822aaeaceaa0db1f85ee3e904c4defbd9689004b03d6fc340455b363f51020ad3ecca4f0850280cf436c70c727923f6db46c3e338cdbb2",
			childPub: "bitcoin-xkey:01004000000000000000000060499f801b896d83179a4374aeb7822aaeaceaa0db1f85ee3e904c4defbd968903cbcaa9c98c877a26977d00825c956a238e8dddfbd322cce4f74b0b5bd6ace4a7bb9aa916",
		},
		{
			name:     "BIP-0032 Test Vector 1.2",
			key:      "bitcoin-xkey:01004000000000000000000060499f801b896d83179a4374aeb7822aaeaceaa0db1f85ee3e904c4defbd9689004b03d6fc340455b363f51020ad3ecca4f0850280cf436c70c727923f6db46c3e338cdbb2",
			path:     []uint32{0}, // m/0
			childKey: "bitcoin-xkey:01004001bd16bee500000000f0909affaa7ee7abe5dd4e100598d4dc53cd709d5a5c2cac40e7412f232f7c9c00abe74a98f6c7eabee0428f53798f0ab8aa1bd37873999041703c742f15ac7e1e2a9f8415",
			childPub: "bitcoin-xkey:01004001bd16bee500000000f0909affaa7ee7abe5dd4e100598d4dc53cd709d5a5c2cac40e7412f232f7c9c02fc9e5af0ac8d9b3cecfe2a888e2117ba3d089d8585886c9c826b6b22a98d12ea39f50829",
		},
		{
			name:     "BIP-0032 Test Vector 1.3",
			key:      "bitcoin-xkey:01004000000000000000000060499f801b896d83179a4374aeb7822aaeaceaa0db1f85ee3e904c4defbd9689004b03d6fc340455b363f51020ad3ecca4f0850280cf436c70c727923f6db46c3e338cdbb2",
			path:     []uint32{0, Hardened + 2147483647}, // m/0/2147483647'
			childKey: "bitcoin-xkey:010040025a61ff8effffffffbe17a268474a6bb9c61e1d720cf6215e2a88c5406c4aee7b38547f585c9a37d900877c779ad9687164e9c2f4f0f4ff0340814392330693ce95a58fe18fd52e6e93abd89116",
			childPub: "bitcoin-xkey:010040025a61ff8effffffffbe17a268474a6bb9c61e1d720cf6215e2a88c5406c4aee7b38547f585c9a37d903c01e7425647bdefa82b12d9bad5e3e6865bee0502694b94ca58b666abc0a5c3b70dcf396",
		},
		{
			name:     "BIP-0032 Test Vector 2.1",
			key:      "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d50800e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b350871a164",
			path:     nil, // m
			childKey: "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d50800e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b350871a164",
			childPub: "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d5080339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2adda4816",
		},
		{
			name:     "BIP-0032 Test Vector 2.2",
			key:      "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d50800e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b350871a164",
			path:     []uint32{Hardened}, // m/0'
			childKey: "bitcoin-xkey:010040013442193e0000008047fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae623614100edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afeaaa97fe7e",
			childPub: "bitcoin-xkey:010040013442193e0000008047fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae6236141035a784662a4a20a65bf6aab9ae98a6c068a81c52e4b032c0fb5400c706cfccc5628287c70",
		},
		{
			name:     "BIP-0032 Test Vector 2.3",
			key:      "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d50800e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b350871a164",
			path:     []uint32{Hardened, 1}, // m/0'/1
			childKey: "bitcoin-xkey:010040025c1bd648010000002a7857631386ba23dacac34180dd1983734e444fdbf774041578e9b6adb37c19003c6cb8d0f6a264c91ea8b5030fadaa8e538b020f0a387421a12de9319dc933681c3609a5",
			childPub: "bitcoin-xkey:010040025c1bd648010000002a7857631386ba23dacac34180dd1983734e444fdbf774041578e9b6adb37c1903501e454bf00751f24b1b489aa925215d66af2234e3891c3b21a52bedb3cd711c1c8e635d",
		},
		{
			name:     "BIP-0032 Test Vector 2.4",
			key:      "bitcoin-xkey:010040000000000000000000873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d50800e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b350871a164",
			path:     []uint32{Hardened, 1, Hardened + 2}, // m/0'/1/2'
			childKey: "bitcoin-xkey:01004003bef5a2f90200008004466b9cc8e161e966409ca52986c584f07e9dc81f735db683c3ff6ec7b1503f00cbce0d719ecf7431d88e6a89fa1483e02e35092af60c042b1df2ff59fa424dcad3ea183e",
			childPub: "bitcoin-xkey:01004003bef5a2f90200008004466b9cc8e161e966409ca52986c584f07e9dc81f735db683c3ff6ec7b1503f0357bfe1e341d01c69fe5654309956cbea516822fba8a601743a012a7896ee8dc261f7d205",
		},
		{
			name:     "BIP-0032 Test Vector 3.1",
			key:      "bitcoin-xkey:01004000000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f0000ddb80b067e0d4993197fe10f2657a844a384589847602d56f0c629c81aae32a98c740e",
			path:     nil, // m
			childKey: "bitcoin-xkey:01004000000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f0000ddb80b067e0d4993197fe10f2657a844a384589847602d56f0c629c81aae32a98c740e",
			childPub: "bitcoin-xkey:01004000000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f03683af1ba5743bdfc798cf814efeeab2735ec52d95eced528e692b8e34c4e5669a2d08c85",
		},
		{
			name:     "BIP-0032 Test Vector 3.2",
			key:      "bitcoin-xkey:01004000000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f0000ddb80b067e0d4993197fe10f2657a844a384589847602d56f0c629c81aae32a98c740e",
			path:     []uint32{Hardened}, // m/0'
			childKey: "bitcoin-xkey:0100400141d63b5000000080e5fea12a97b927fc9dc3d2cb0d1ea1cf50aa5a1fdc1f933e8906bb38df3377bd00491f7a2eebc7b57028e0d3faa0acda02e75c33b03c48fb288c41e2ea44e1daefad592edc",
			childPub: "bitcoin-xkey:0100400141d63b5000000080e5fea12a97b927fc9dc3d2cb0d1ea1cf50aa5a1fdc1f933e8906bb38df3377bd026557fdda1d5d43d79611f784780471f086d58e8126b8c40acb82272a7712e7f2818b85ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xkey, err := ExtendedKeyFromStr(tt.key)
			if err != nil {
				t.Fatalf("Failed decode key : %s", err)
			}
			// t.Logf("Master Key : %s", xkey.String())

			child := xkey
			for i, index := range tt.path {
				t.Logf("Deriving child %d", index)
				child, err = child.ChildKey(index)
				if err != nil {
					t.Fatalf("Failed to derive child : %s", err)
				}
				if child.Depth != byte(i+1) {
					t.Fatalf("Incorrect child depth : got %d, want %d", child.Depth, i+1)
				}
				if child.Index != index {
					t.Fatalf("Incorrect child index : got %d, want %d", child.Index, index)
				}
			}
			// t.Logf("Key : %s", child.String())

			if child.KeyValue[0] != 0 {
				t.Fatalf("Child key not private")
			}

			pubKey := child.ExtendedPublicKey()
			// t.Logf("Public Key : %s", pubKey.String())

			wantKey, err := ExtendedKeyFromStr(tt.childKey)
			if err != nil {
				t.Fatalf("Failed to load child key : %s", err)
			}
			// t.Logf("Old Key : %s", wantKey.String())

			if !bytes.Equal(child.ChainCode[:], wantKey.ChainCode[:]) {
				t.Fatalf("Child private chain doesn't match :\n  got  : %x\n  want : %x\n", child.ChainCode[:],
					wantKey.ChainCode[:])
			}
			if !bytes.Equal(child.KeyValue[:], wantKey.KeyValue[:]) {
				t.Fatalf("Child private key doesn't match :\n  got  : %x\n  want : %x\n", child.KeyValue[:],
					wantKey.KeyValue[:])
			}

			wantPubKey, err := ExtendedKeyFromStr(tt.childPub)
			if err != nil {
				t.Fatalf("Failed to load child pub key : %s", err)
			}
			// t.Logf("Old Pub : %s", wantPubKey.String())

			if !bytes.Equal(pubKey.ChainCode[:], wantPubKey.ChainCode[:]) {
				t.Fatalf("Child public chain doesn't match :\n  got  : %x\n  want : %x\n", pubKey.ChainCode[:],
					wantPubKey.ChainCode[:])
			}
			if !bytes.Equal(pubKey.KeyValue[:], wantPubKey.KeyValue[:]) {
				t.Fatalf("Child public key doesn't match :\n  got  : %x\n  want : %x\n", pubKey.KeyValue[:],
					wantPubKey.KeyValue[:])
			}

			b58 := child.String58()
			b58Key, err := ExtendedKeyFromStr58(b58)
			if err != nil {
				t.Fatalf("Failed to decode base 58 xkey : %s", err)
			}
			if !b58Key.Equal(child) {
				t.Fatalf("Base 58 key not equal :\ngot :\n%+v\nwant :\n%+v", b58Key, child)
			}

			// Verify same child key generated from private and public keys
			privGrandChild, err := child.ChildKey(1000)
			if err != nil {
				t.Fatalf("Failed to generate private grand child : %s", err)
			}

			pubGrandChild, err := child.ExtendedPublicKey().ChildKey(1000)
			if err != nil {
				t.Fatalf("Failed to generate public grand child : %s", err)
			}

			if !privGrandChild.ExtendedPublicKey().Equal(pubGrandChild) {
				t.Fatalf("Grand children not equal :\n  got  : %x\n  want : %x",
					privGrandChild.PublicKey().Bytes(), pubGrandChild.PublicKey().Bytes())
			}
		})
	}
}

func TestExtendedKeys(t *testing.T) {
	tests := []struct {
		key  string
		keys []string
	}{
		{
			key: "bitcoin-xkeys:01004103025a61ff8effffffffbe17a268474a6bb9c61e1d720cf6215e2a88c5406c4aee7b38547f585c9a37d903c01e7425647bdefa82b12d9bad5e3e6865bee0502694b94ca58b666abc0a5c3b00000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f03683af1ba5743bdfc798cf814efeeab2735ec52d95eced528e692b8e34c4e56690141d63b5000000080e5fea12a97b927fc9dc3d2cb0d1ea1cf50aa5a1fdc1f933e8906bb38df3377bd026557fdda1d5d43d79611f784780471f086d58e8126b8c40acb82272a7712e7f247051e3c",
			keys: []string{
				"bitcoin-xkey:010040025a61ff8effffffffbe17a268474a6bb9c61e1d720cf6215e2a88c5406c4aee7b38547f585c9a37d903c01e7425647bdefa82b12d9bad5e3e6865bee0502694b94ca58b666abc0a5c3b70dcf396",
				"bitcoin-xkey:01004000000000000000000001d28a3e53cffa419ec122c968b3259e16b65076495494d97cae10bbfec3c36f03683af1ba5743bdfc798cf814efeeab2735ec52d95eced528e692b8e34c4e5669a2d08c85",
				"bitcoin-xkey:0100400141d63b5000000080e5fea12a97b927fc9dc3d2cb0d1ea1cf50aa5a1fdc1f933e8906bb38df3377bd026557fdda1d5d43d79611f784780471f086d58e8126b8c40acb82272a7712e7f2818b85ab",
			},
		},
	}

	for tindex, tt := range tests {
		t.Run(fmt.Sprintf("%d", tindex), func(t *testing.T) {
			xkeysSep := ExtendedKeys{}
			for _, key := range tt.keys {
				xkey, err := ExtendedKeyFromStr(key)
				if err != nil {
					t.Fatalf("Failed to decode xkey : %s", err)
				}
				xkeysSep = append(xkeysSep, xkey)
			}
			t.Logf("Composite : %s", xkeysSep.String())

			xkeys, err := ExtendedKeysFromStr(tt.key)
			if err != nil {
				t.Fatalf("Failed to decode xkeys : %s", err)
			}

			if !xkeys.Equal(xkeysSep) {
				t.Fatalf("Xkeys not equal")
			}

			b58 := xkeys.String58()
			b58Keys, err := ExtendedKeysFromStr58(b58)
			if err != nil {
				t.Fatalf("Failed to decode base 58 xkey : %s", err)
			}
			t.Logf("Base 58 Keys : %s", b58)
			if !b58Keys.Equal(xkeys) {
				t.Fatalf("Base 58 keys not equal :\ngot :\n%+v\nwant :\n%+v", b58Keys, xkeys)
			}
		})
	}
}

// TestOldExtendedKey tests conversion from old BIP-0032 format.
func TestOldExtendedKey(t *testing.T) {
	tests := []struct {
		key string
	}{
		{key: "xpub661MyMwAqRbcFW31YEwpkMuc5THy2PSt5bDMsktWQcFF8syAmRUapSCGu8ED9W6oDMSgv6Zz8idoc4a6mr8BDzTJY47LJhkJ8UB7WEGuduB"},
		{key: "xpub69H7F5d8KSRgmmdJg2KhpAK8SR3DjMwAdkxj3ZuxV27CprR9LgpeyGmXUbC6wb7ERfvrnKZjXoUmmDznezpbZb7ap6r1D3tgFxHmwMkQTPH"},
		{key: "xpub6ASuArnXKPbfEwhqN6e3mwBcDTgzisQN1wXN9BJcM47sSikHjJf3UFHKkNAWbWMiGj7Wf5uMash7SyYq527Hqck2AxYysAA7xmALppuCkwQ"},
		{key: "xpub6D4BDPcP2GT577Vvch3R8wDkScZWzQzMMUm3PWbmWvVJrZwQY4VUNgqFJPMM3No2dFDFGTsxxpG5uJh7n7epu4trkrX7x7DogT5Uv6fcLW5"},
	}

	for tindex, tt := range tests {
		t.Run(fmt.Sprintf("%d", tindex), func(t *testing.T) {
			bip32Key, err := bip32.B58Deserialize(tt.key)
			if err != nil {
				t.Fatalf("Failed to deserialize old key : %s", err)
			}

			newKey, err := ExtendedKeyFromStr(tt.key)
			if err != nil {
				t.Fatalf("Failed to deserialize new key : %s", err)
			}

			if !bytes.Equal(newKey.KeyValue[:], bip32Key.Key) {
				t.Fatalf("Imported keys not equal :\n  got  : %x\n  want : %x", newKey.KeyValue[:],
					bip32Key.Key)
			}

			child32, err := bip32Key.NewChildKey(1)
			if err != nil {
				t.Fatalf("Failed to generate old child : %s", err)
			}

			child, err := newKey.ChildKey(1)
			if err != nil {
				t.Fatalf("Failed to generate new child : %s", err)
			}

			if !bytes.Equal(child.KeyValue[:], child32.Key) {
				t.Fatalf("Children not equal :\n  got  : %x\n  want : %x", child.KeyValue[:],
					child32.Key)
			}
		})
	}
}

func BenchmarkExtendedKeyPrivate(b *testing.B) {
	key, err := GenerateMasterExtendedKey()
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = key.ChildKey(uint32(i))
		if err != nil {
			b.Fatalf("Failed to generate child key : %s", err)
		}
	}
	b.StopTimer()
}

func BenchmarkExtendedKeyPublic(b *testing.B) {
	key, err := GenerateMasterExtendedKey()
	if err != nil {
		b.Fatalf("Failed to generate key : %s", err)
	}

	key = key.ExtendedPublicKey()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = key.ChildKey(uint32(i))
		if err != nil {
			b.Fatalf("Failed to generate child key : %s", err)
		}
	}
	b.StopTimer()
}
