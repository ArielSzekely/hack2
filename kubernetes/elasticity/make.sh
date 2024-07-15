#!/bin/bash

RACE=""

usage() {
    echo "Usage: $0 [-race]" 1>&2
}

while [[ "$#" -gt 0 ]]; do
    case "$1" in
    -race)
        shift
        RACE="-race"
	;;
    *)
	error "unexpected argument $1"
	usage
	exit 1
	;;
    esac
done

mkdir -p bin

for f in `ls cmd`
do
  echo "go build $RACE -o bin/$f cmd/$f/main.go"
  go build $RACE -o bin/$f cmd/$f/main.go
done
