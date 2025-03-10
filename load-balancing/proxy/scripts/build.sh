#!/bin/bash

DIR=$(dirname $0)
cd $DIR/..

mkdir -p bin

go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
go build -o bin/netproxy cmd/netproxy/main.go
go build -o bin/pipeproxy cmd/pipeproxy/main.go

#docker build -t arielszekely/lb-proxy-perf -f Dockerfile .
#docker push arielszekely/lb-proxy-perf
