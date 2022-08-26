module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/config v0.2.1
	github.com/tokenized/pkg v0.4.1-0.20220809141349-ce8dde7d3da0
	github.com/tokenized/specification v1.1.2-0.20220826015300-8a77ac76380b
	github.com/tokenized/spynode v0.2.2-0.20220809142636-f87c0f951f02
	go.opencensus.io v0.22.2
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
