#!/bin/bash

echo "starting etcd"

docker run --rm -d \
  --name load-test-etcd \
  --env ALLOW_NONE_AUTHENTICATION=yes \
  --publish 4379:2379 \
  --publish 4380:2380 \
  --publish 4381:2381 \
  --publish 4382:2382 \
  --publish 4383:2383 \
  bitnami/etcd:latest

echo "done starting etcd"
