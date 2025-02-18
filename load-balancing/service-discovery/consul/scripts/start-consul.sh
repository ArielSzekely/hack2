#!/bin/bash

echo "starting consul"

docker run -dit \
  --rm \
  --name consul-bench \
  --network host \
  hashicorp/consul

echo "done starting consul"
