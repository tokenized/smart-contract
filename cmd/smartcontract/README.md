# Smart Contract Command Line Interface

This package is just for testing and convenience. It should not be part of any production system.

## Key Generation

This package can be used to generate and derive keys to be used in other systems.

### Generate individual key

The below command will generate an individual key and print the WIF (Wallet Import Format), hex compressed public key, and base58 bitcoin address associated with it.

	smartcontract gen

### Generate extended key

The below command will generate an extended key and print the private and public representations.

	smartcontract gen --x

#### Derive Children

The below command will derive children for an extended key. 

	smartcontract derive <xkey> <path>

`<xkey>` is the parent extended key and must start with "bitcoin-xkey:". It can be private or public and if it is private the WIF will be output.

`<path>` is valid with or without the leading "m/". If the extended key is private it can include "hardened" indexes "m/0'/1".
