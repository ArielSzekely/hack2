#!/bin/bash

b64=$(echo '{"id": "abc"}' | base64)
aws lambda invoke --function-name seccomp-allowlist response.json --payload $b64
cat response.json
rm -f response.json
