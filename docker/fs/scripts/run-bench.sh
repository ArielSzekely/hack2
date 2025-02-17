#!/bin/bash

#!/bin/bash

echo "starting ctr"

docker run -it \
  --rm \
  --name docker-fs-bench \
  --network host \
  --mount type=bind,src=$(pwd),dst=/app \
  arielszekely/docker-fs-bench \
  go test -v fs_test.go --n_dummy_files 1000

echo "done running bench"
