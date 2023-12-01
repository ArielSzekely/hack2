#!/bin/bash

docker tag lambda-start-latency:latest 223652007189.dkr.ecr.us-east-1.amazonaws.com/lambda-start-latency:latest
aws ecr get-login-password | docker login --username AWS --password-stdin 223652007189.dkr.ecr.us-east-1.amazonaws.com
docker push 223652007189.dkr.ecr.us-east-1.amazonaws.com/lambda-start-latency:latest
