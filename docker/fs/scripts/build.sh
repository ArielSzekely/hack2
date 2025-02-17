#!/bin/bash

DOCKER_BUILDKIT=1 docker build --progress=plain -f Dockerfile -t arielszekely/docker-fs-bench . 2>&1
