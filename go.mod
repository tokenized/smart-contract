module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/pkg v0.2.3-0.20210120232054-033e3826e206
	github.com/tokenized/specification v0.3.1
	github.com/tokenized/spynode v0.0.0-20210121022332-cbb285fa7768
	go.opencensus.io v0.22.2
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/tools v0.0.0-20200107184032-11e9d9cc0042 // indirect
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
