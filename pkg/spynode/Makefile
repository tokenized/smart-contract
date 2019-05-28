BUILD_DATE = `date +%FT%T%z`
BUILD_USER = $(USER)@`hostname`
VERSION = `git describe --tags`

# command to build and run on the local OS.
GO_BUILD = go build

# command to compiling the distributable. Specify GOOS and GOARCH for the
# target OS.
GO_DIST = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -a -tags netgo -ldflags "-w -X main.buildVersion=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.buildUser=$(BUILD_USER)"

BINARY=spynoded

all: clean prepare deps test dist

ci: all lint

deps:
	go get -t ./...

prepare:
	mkdir -p dist tmp

dist: dist-spynode

# build a version suitable for distribtion on Linux, or Docker.
dist-spynode:
	$(GO_DIST) -o dist/$(BINARY) cmd/$(BINARY)/main.go

build: build-spynode

# build a version suitable for running locally
build-spynode:
	go build -o dist/$(BINARY) cmd/$(BINARY)/main.go

prepare-win:
	mkdir dist | echo dist exists
	mkdir tmp | echo tmp exists
	set GOOS=windows
	set GOARCH=amd64

build-win: prepare-win build-spynode-win

# build a version suitable for running locally
build-spynode-win:
	go build -o dist\$(BINARY).exe cmd\$(BINARY)\main.go

tools:
	go get golang.org/x/tools/cmd/goimports
	go get github.com/golang/lint/golint

run:
	go run cmd/$(BINARY)/main.go

run-win:
	go run cmd\$(BINARY)\main.go

lint: golint vet goimports

vet:
	go vet

golint:
	ret=0 && test -z "$$(golint . | tee /dev/stderr)" || ret=1 ; exit $$ret

goimports:
	ret=0 && test -z "$$(goimports -l . | tee /dev/stderr)" || ret=1 ; exit $$ret

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
