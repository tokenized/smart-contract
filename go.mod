module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/config v0.0.3
	github.com/tokenized/pkg v0.2.3-0.20210318205230-1b68053795c1
	github.com/tokenized/specification v0.3.2-0.20210325193340-3bd88eee903d
	github.com/tokenized/spynode v0.0.0-20210314214409-1fa9f64838d7
	go.opencensus.io v0.22.2
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
