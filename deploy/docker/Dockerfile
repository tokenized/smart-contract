# Build Container
FROM golang:1.11-alpine3.8 AS build-env
RUN apk add --no-cache --upgrade git make ca-certificates
WORKDIR /go/src/github.com/tokenized/smart-contract
COPY . .
RUN go get -v -t ./...
RUN make clean prepare dist-smartcontractd

# Final Container
FROM alpine:3.8
LABEL maintainer="Tokenized"
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env /go/src/github.com/tokenized/smart-contract/dist/smartcontractd /usr/local/bin/smartcontractd
ENTRYPOINT ["/usr/local/bin/smartcontractd"]
