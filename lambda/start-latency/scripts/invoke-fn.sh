#!/bin/bash

current_time_micro=(($(date +%s%N)/1000))
b64=$(echo "{\"cur_time_micro\": \"$current_time_micro\"}" | base64)
aws lambda invoke --function-name lambda-start-latency response.json --payload $b64
cat response.json
rm -f response.json
