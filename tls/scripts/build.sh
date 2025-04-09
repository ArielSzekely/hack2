#!/bin/bash

DIR=$(dirname $0)
cd $DIR/..

# Clear out keys directory
rm -rf keys

mkdir -p bin
mkdir -p keys

# Generate a server key
openssl ecparam -genkey -name secp384r1 -out keys/server.key

# Generate a self-signed x509 pubkey
openssl req -new -x509 -sha256 -key keys/server.key -out keys/server.crt -days 3650 -config cert-config.txt -addext "subjectAltName = DNS:foo.example.org"

# Build client/server bins
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
