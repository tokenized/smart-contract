# TxBuilder

A package for building Bitcoin transactions (Tx) in Go.

## Usage

Building a transaction is fairly simple. See the [tests](txbuilder/txbuilder_test.go) for complete usage.

_Error handling omitted here for brevity._

```
// Load your private key
wif := "cQDgbH4C7HP3LSJevMSb1dPMCviCPoLwJ28mxnDRJueMSCa72xjm"
key, net, err := bitcoin.DecodeKeyString(wif)

// Decode an address to use for "change".
// Middle return parameter is the network detected. This should be checked to ensure the address
//   was encoded for the currently specified network.
changeAddress, net, err := bitcoin.DecodeAddressString("mq4htwkZSAG9isuVbEvcLaAiNL59p26W64")

// Create an instance of the TxBuilder using 512 as the dust limit and 1.1 sat/byte fee rate.
builder := txbuilder.NewTxBuilder(changeAddress, 512, 1.1)

// Add an input
// To spend an input you need the txid, output index, and the locking script and value from that output.
hash, err := bitcoin.NewHash32FromStr("c762a29a4beb4821ad843590c3f11ffaed38b7eadc74557bdf36da3539921531")
index := uint32(0)
value := uint64(2000)
spendAddress, net, err := bitcoin.DecodeAddressString("mupiWN44gq3NZmvZuMMyx8KbRwism69Gbw")
err = builder.AddInput(*wire.NewOutPoint(hash, index), spendAddress.LockingScript(), value)

// Add an output to the recipient
paymentAddress, net, err := bitcoin.DecodeAddressString("n1kBjpqmH82jgiRnEHLmFMNv77kvugBomm")
err = builder.AddPaymentOutput(paymentAddress, 1000, false) // false because it isn't change

// Sign the first and only input
err = builder.Sign([]bitcoin.Key{key})

// Get the raw transaction bytes
data, err = builder.Serialize()
```

## Build

Install deps, run tests, and build and install the utility binaries.

    make


## Contributing

1. Fork the repository.
1. Create a new branch to work on. Branch from develop if it exists, else
   from master.
1. Implement/fix your feature, comment your code.
1. Follow the existing code style of the project, including indentation.
1. PR's must have tests.


## References

- [Stress Test Network](https://bitcoinscaling.io/)
- [Gigabyte Testnet Faucet](https://faucet.bitcoinscaling.io/)


## License

Copyright 2019 Tokenized Group Pty Ltd.

This source code is released under the terms of the Open BSV license. See
[LICENSE](LICENSE) for more information.
