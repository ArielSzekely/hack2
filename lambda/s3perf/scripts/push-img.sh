#!/bin/bash

docker tag s3perf:latest 223652007189.dkr.ecr.us-east-1.amazonaws.com/s3perf:latest
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 223652007189.dkr.ecr.us-east-1.amazonaws.com
docker push 223652007189.dkr.ecr.us-east-1.amazonaws.com/s3perf:latest
aws lambda update-function-code \
  --function-name s3perf \
  --image-uri 223652007189.dkr.ecr.us-east-1.amazonaws.com/s3perf:latest \
  --publish
