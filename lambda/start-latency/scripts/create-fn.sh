#!/bin/bash

aws lambda create-function \
  --function-name lambda-start-latency \
  --package-type Image \
  --code ImageUri=223652007189.dkr.ecr.us-east-1.amazonaws.com/lambda-start-latency:latest \
  --role arn:aws:iam::223652007189:role/lambda-ex
