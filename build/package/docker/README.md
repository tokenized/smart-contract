# Tokenized - Docker

## Usage

Note: Ensure your configuration is correctly setup to use S3 for storage or you could lose data:

```
docker pull tokenized/smartcontractd
docker run --env-file ./smartcontractd.conf tokenized/smartcontractd
```

## Building

Build:

```
docker build -t tokenized/smartcontractd -f ./Dockerfile ../../../
```

Run as a local test:

```
docker run --rm -it --env-file ./smartcontractd.conf tokenized/smartcontractd
```

Push to dockerhub:

```
docker login

docker push tokenized/smartcontractd
```
