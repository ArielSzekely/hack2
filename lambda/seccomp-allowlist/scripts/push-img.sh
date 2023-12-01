#!/bin/bash

docker tag seccomp-allowlist:latest 223652007189.dkr.ecr.us-east-1.amazonaws.com/seccomp-allowlist:latest
aws ecr get-login-password | docker login --username AWS --password-stdin 223652007189.dkr.ecr.us-east-1.amazonaws.com
docker push 223652007189.dkr.ecr.us-east-1.amazonaws.com/seccomp-allowlist:latest
