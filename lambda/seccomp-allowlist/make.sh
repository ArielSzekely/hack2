#!/bin/bash

ROOT_DIR=$(realpath $(dirname $0)/..)
cd $ROOT_DIR

mkdir -p bin
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/app cmd/main.go
