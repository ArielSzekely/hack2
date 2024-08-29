#!/bin/bash

usage() {
  echo "Usage: $0 --obj-path OBJ_PATH" 1>&2
}

OBJ_PATH=""
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  --obj-path)
    shift
    OBJ_PATH=$1
    shift
    ;;
  -help)
    usage
    exit 0
    ;;
  *)
    echo "Error: unexpected argument '$1'"
    usage
    exit 1
    ;;
  esac
done

if [ -z "$OBJ_PATH" ] || [ $# -gt 0 ]; then
    usage
    exit 1
fi

b64=$(echo "{\"obj_path\": \"$OBJ_PATH\"}" | base64)
aws lambda invoke --function-name s3perf response.json --payload $b64
cat response.json
rm -f response.json
