# Smart Contract

An autonomous agent that operates on behalf of the issuer to recognize and enforce some or all the terms and conditions of the specified contract.  At a minimum, the Smart Contract operates within the bounds of the Tokenized Protocol on the Bitcoin SV (BSV) network. The Smart Contract is written in Go and is intended to offer all the basic functionality required to issue, manage and exchange tokens, but it can also be used as the basis for much more complex smart contracts.

### Usage Warning

**This is a Beta release, with more changes likely before the final version. Do not use this in production.**

## Getting Started

### Quick Start

If you're only ever going to run the binary and are not interested in
working with the source code.

    go get github.com/tokenized/smart-contract/cmd/...

### Build from Source

To build the project from source, clone the GitHub repo and in a command line terminal.

    mkdir -p $GOPATH/src/github.com/tokenized
    cd $GOPATH/src/github.com/tokenized
    git clone git@github.com:tokenized/smart-contract
    cd smart-contract

Navigate to the root directory and run:

    make

## Project Components

#### COMMAND BINARIES

- `cmd/smartcontract` - Command line interface
- `cmd/smartcontractd` - Smart Contract node daemon

#### PUBLIC KIT

- `pkg/bitcoin` - Bitcoin utility package
- `pkg/spynode` - Connects to a trusted node as a peer via the public interface.
- `pkg/rpcnode` - Connects to a trusted node via the RPC interface.
- `pkg/inspector` - Looks at transaction objects, converts them to a special transaction type (inspector.Transaction / itx) used throughout the app.
- `pkg/txbuilder` - Generic BCH library for performing bitcoin related tasks, mainly around tx building.
- `pkg/storage` - Storage engine, supporting S3 and local filesystem.
- `pkg/wallet` - Private key manager
- `pkg/wire` - Bitcoin protocol message definitions.

## Configuration

Configuration is supplied via environment variables. See the
[example](conf/dev.env.example) file that would export environment variables
on Linux and macOS.

Make a copy the example file, and edit the values to suit. The file can be placed anywhere if you prefer.

    cp conf/dev.env.example ~/.contract/acme.env

The following configuration values are needed:

##### Contract config

- `OPERATOR_NAME` the name of the operator of the smart contract. Eg: _ACME Corporation_
- `FEE_ADDRESS` public address to earn fees upon every action
- `FEE_RATE` the cost in satoshis to perform an action (<2000 at this stage)
- `DUST_LIMIT` dust limit as determined by the network (default: 546)

##### Node config

- `NODE_ADDRESS` hostname or IP address for a public node
- `NODE_USER_AGENT` the user agent to provide when connecting to the public node
- `RPC_HOST` hostname or IP address for a private node (RPC)
- `RPC_USERNAME` username for RPC authentication
- `RPC_PASSWORD` password for RPC authentication
- `PRIV_KEY` private key (WIF) used by the smart contract
- `BITCOIN_CHAIN` bitcoin network as: mainnet, testnet (default: mainnet)

##### Contract storage

- `CONTRACT_STORAGE_BUCKET` S3 bucket for data storage, use *standalone* for local filesystem
- `CONTRACT_STORAGE_ROOT` root directory for storage

##### Node storage

- `NODE_STORAGE_BUCKET` S3 bucket for data storage, use *standalone* for local filesystem
- `NODE_STORAGE_ROOT` base directory for storage files

##### AWS credentials (optional S3 storage)

- `AWS_REGION` hosted region for data storage
- `AWS_ACCESS_KEY_ID` access key for data storage
- `AWS_SECRET_ACCESS_KEY` secret for data storage

## Running

This example shows the config file containing the environment variables
residing at `./conf/dev.env.example`:

    source ./conf/dev.env.example && make run

### Dependencies

The Smart Contract requires RPC access to a full bitcoin node, such as [Bitcoin SV](https://github.com/bitcoin-sv/bitcoin-sv). Once installed and syncronised with the BSV network, ensure that RPC is enabled by modifying the `bitcoin.conf` file.

    # bitcoin.conf
    server=1
    rpuser=someUserName
    rpcpassword=somePassword
    rpcport=8332

## Running unit tests

To perform unit tests run:

    make test

## Deployment

See the [deploy directory](deploy/) for information on how to deploy the smart contract.

# License

The Tokenized Smart Contract is open-sourced software licensed under the [OPEN BITCOIN SV](LICENSE.md) license.
