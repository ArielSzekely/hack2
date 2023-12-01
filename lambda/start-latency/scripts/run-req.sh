#!/bin/bash

current_time_micro=$(($(date +%s%N)/1000))
data=$(echo "{\"cur_time_micro\": $current_time_micro}")
curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" -d "$data"
