#!/bin/bash

DIR=$(dirname $0)
cd $DIR/..
docker build -t arielszekely/rs-hello-world -f Dockerfile .
docker push arielszekely/rs-hello-world
