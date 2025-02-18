#!/bin/bash

echo "shutting down etcd"

if docker ps -a | grep -qE 'load-test-etcd'; then
  for container in $(docker ps -a | grep -E 'load-test-etcd' | cut -d ' ' -f1) ; do
    docker stop $container
    while docker ps -a | grep -qE "$container"; do
      printf "."
      sleep 1
    done
  done
fi

echo "done shutting down etcd"
