BUILD_DATE = `date +%FT%T%z`
BUILD_USER = $(USER)@`hostname`
VERSION = `git describe --tags`

# command to build and run on the local OS.
GO_BUILD = go build

# command to compiling the distributable. Specify GOOS and GOARCH for the
# target OS.
GO_DIST = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -a -tags netgo -ldflags "-w -X main.buildVersion=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.buildUser=$(BUILD_USER)"

BINARY=smartcontractd

# tools
BINARY_CONTRACT_CLI=smartcontract
BINARY_SPVNODE=spvnode

all: clean prepare deps test dist

ci: all lint

deps:
	go get -t ./...

dist: dist-smartcontractd dist-tools

dist-smartcontractd:
	$(GO_DIST) -o dist/$(BINARY) cmd/$(BINARY)/smartcontractd.go

dist-tools: dist-cli \
	dist-spvnode

dist-cli:
	$(GO_DIST) -o dist/$(BINARY_CONTRACT_CLI) cmd/$(BINARY_CONTRACT_CLI)/smartcontract.go

dist-spvnode:
	$(GO_DIST) -o dist/$(BINARY_SPVNODE) cmd/$(BINARY_SPVNODE)/spvnode.go

prepare:
	mkdir -p dist tmp

tools:
	go get golang.org/x/tools/cmd/goimports
	go get github.com/golang/lint/golint

run:
	go run cmd/$(BINARY)/smartcontractd.go

run-cli:
	go run cmd/$(BINARY_CONTRACT_CLI)/smartcontract.go

run-spvnode:
	go run cmd/$(BINARY_SPVNODE)/spvnode.go

lint: golint vet goimports

vet:
	go vet

golint:
	ret=0 && test -z "$$(golint ./... | tee /dev/stderr)" || ret=1 ; exit $$ret

goimports:
	ret=0 && test -z "$$(goimports -l ./... | tee /dev/stderr)" || ret=1 ; exit $$ret

# run the tests with coverage
test: prepare
	go test -coverprofile=tmp/coverage.out

# run tests with coverage and open html file in the browser
#
# See https://blog.golang.org/cover for more output options
test-coverage: test
	go tool cover -html=tmp/coverage.out

clean:
	rm -rf dist
