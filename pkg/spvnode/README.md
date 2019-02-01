# SPV Node

The package provids a lightweight node, that is sufficient for the
smart contract to run on.

While there is a binary, it is only used to prove that the Node runs.

This package is meant to be used as a building block, rather than an full
node.


### Makefile

If you're interested in working with the source code, this is your best option.

```
mkdir -p $GOPATH/src/bitbucket.org/tokenized
cd $GOPATH/src/bitbucket.org/tokenized
git clone git@bitbucket.org:tokenized/spvnode.git
cd spvnode
make
```


## Configuration

Configuration is supplied via environment variables. See the
[example](conf/dev.env.example) file that would export environment variables
on Linux and OS X.

Make a copy the example file, and edit the values to suit.

The file can be placed anywhere you prefer.


## Running

This example shows the config file containing the environment variables
residing at `./tmp/dev.env`

```
source ./tmp/dev.env && go get cmd/spvnode/main.go
```

## License

(c) Tokenized Cash 2018 All rights reserved
