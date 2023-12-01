#!/bin/bash

ROOT_DIR=$(realpath $(dirname $0)/..)
cd $ROOT_DIR

./scripts/make.sh
docker build --progress=plain -f Dockerfile -t seccomp-allowlist .
