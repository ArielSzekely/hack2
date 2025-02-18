#!/bin/bash

echo "shutting down consul"

if docker ps -a | grep -qE 'consul-bench'; then
  for container in $(docker ps -a | grep -E 'consul-bench' | cut -d ' ' -f1) ; do
    docker stop $container
    while docker ps -a | grep -qE "$container"; do
      printf "."
      sleep 1
    done
  done
fi

echo "done shutting down consul"
