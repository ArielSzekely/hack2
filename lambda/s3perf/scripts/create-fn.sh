#!/bin/bash

aws lambda create-function \
  --function-name s3perf \
  --memory-size 1769 \
  --timeout 90 \
  --package-type Image \
  --code ImageUri=223652007189.dkr.ecr.us-east-1.amazonaws.com/s3perf:latest \
  --role arn:aws:iam::223652007189:role/lambda-ex
