#!/bin/bash

DIR=$(dirname $0)

cd $DIR/..
docker build -t arielszekely/spinhttpsrv -f docker/Dockerfile . && \
  docker push arielszekely/spinhttpsrv
