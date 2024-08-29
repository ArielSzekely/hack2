#!/bin/bash

docker tag cpuperf:latest 223652007189.dkr.ecr.us-east-1.amazonaws.com/cpuperf:latest
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 223652007189.dkr.ecr.us-east-1.amazonaws.com
docker push 223652007189.dkr.ecr.us-east-1.amazonaws.com/cpuperf:latest
aws lambda update-function-code \
  --function-name cpuperf \
  --image-uri 223652007189.dkr.ecr.us-east-1.amazonaws.com/cpuperf:latest \
  --publish
