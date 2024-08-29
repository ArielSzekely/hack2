#!/bin/bash

ROOT_DIR=$(realpath $(dirname $0)/..)
cd $ROOT_DIR

./scripts/make.sh
docker build --platform linux/amd64 --progress=plain -f Dockerfile -t cpuperf .
rm main
