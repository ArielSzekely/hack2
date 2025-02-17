#!/bin/bash

echo "starting ctr"

docker run -it \
  --rm \
  --name docker-fs-bench \
  --network host \
  --mount type=bind,src=$(pwd),dst=/app \
  arielszekely/docker-fs-bench


echo "done starting ctr"
