#!/bin/bash

DIR=$(dirname $0)
cd $DIR/..

mkdir -p bin

go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go

docker build -t arielszekely/netperf -f Dockerfile .
docker push arielszekely/netperf
