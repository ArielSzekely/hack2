#!/bin/bash

DIR=$(dirname $0)

N_ITER=1000000
CGROUP=/sys/fs/cgroup/cpu,cpuacct/system.slice/spin

if ! [ -d $CGROUP ]; then
  echo "Make cgroup"
  mkdir $CGROUP
fi

echo "Starting spinner"
$DIR/../spin/c/spin -t $(nproc) 1000000 &
SPIN_PID=$(pgrep spin)
echo "Adding to cgroup"
echo $SPIN_PID | sudo tee $CGROUP/cgroup.procs
echo "Added to cgroup"
