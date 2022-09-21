module github.com/tokenized/smart-contract

go 1.12

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.3.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/tokenized/config v0.2.2-0.20220902160347-43a4340c357e
	github.com/tokenized/logger v0.1.1
	github.com/tokenized/pkg v0.4.1-0.20220906215914-999453d439df
	github.com/tokenized/specification v1.1.2-0.20220909144939-3aa7ec366bbb
	github.com/tokenized/spynode v0.2.2-0.20220906224702-89137f93b0e3
	github.com/tokenized/threads v0.1.1-0.20220908162622-5e406dccfad8
	go.opencensus.io v0.22.2
)

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
