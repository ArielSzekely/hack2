#!/bin/bash

usage() {
  echo "Usage: $0" 1>&2
}

while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
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

if [ $# -gt 0 ]; then
    usage
    exit 1
fi

b64=$(echo "{}" | base64)
aws lambda invoke --function-name cpuperf response.json --payload $b64
cat response.json
rm -f response.json
