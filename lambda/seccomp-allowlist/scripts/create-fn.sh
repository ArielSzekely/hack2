#!/bin/bash

aws lambda create-function \
  --function-name seccomp-allowlist \
  --package-type Image \
  --code ImageUri=223652007189.dkr.ecr.us-east-1.amazonaws.com/seccomp-allowlist:latest \
  --role arn:aws:iam::223652007189:role/lambda-ex
