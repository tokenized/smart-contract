module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/aws/aws-sdk-go v1.27.0
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v0.0.0-20191219182022-e17c9730c422
	github.com/cmars/basen v0.0.0-20150613233007-fe3947df716e // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.0
	github.com/google/uuid v1.1.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/specification v0.2.3-0.20200331020322-b65a25099cf8
	github.com/tyler-smith/go-bip32 v0.0.0-20170922074101-2c9cfd177564
	go.opencensus.io v0.22.2
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	launchpad.net/gocheck v0.0.0-20140225173054-000000000087 // indirect
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
