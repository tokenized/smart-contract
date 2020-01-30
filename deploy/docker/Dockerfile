# Build Container
FROM golang:1.12-alpine3.10 AS build-env
RUN apk add --no-cache --upgrade git make ca-certificates
WORKDIR /go/src/github.com/tokenized/smart-contract
COPY . .
ENV GO111MODULE on
RUN make clean prepare dist-smartcontractd

# Final Container
FROM alpine:3.10
LABEL maintainer="Tokenized"
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env /go/src/github.com/tokenized/smart-contract/dist/smartcontractd /usr/local/bin/smartcontractd
ENTRYPOINT ["/usr/local/bin/smartcontractd"]
