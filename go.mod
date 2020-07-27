module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/aws/aws-sdk-go v1.31.6
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/cmars/basen v0.0.0-20150613233007-fe3947df716e // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.4.0
	github.com/google/uuid v1.1.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/pkg v0.0.0-20200724032845-ccddf7770c5a
	github.com/tokenized/specification v0.2.3-0.20200727013108-2201c9e27a22
	github.com/tyler-smith/go-bip32 v0.0.0-20170922074101-2c9cfd177564
	go.opencensus.io v0.22.2
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	launchpad.net/gocheck v0.0.0-20140225173054-000000000087 // indirect
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
