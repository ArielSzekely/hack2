#!/bin/bash

docker run -dit \
  --rm \
  --name consul-bench \
  --network host \
  hashicorp/consul
