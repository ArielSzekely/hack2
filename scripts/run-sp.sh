#!/bin/bash

DIR=$(dirname $0)

N_ITER=20000000000
CGROUP=/sys/fs/cgroup/cpu,cpuacct/system.slice/spin

if ! [ -d $CGROUP ]; then
  echo "Make cgroup"
  sudo mkdir $CGROUP
  echo 100 | sudo tee $CGROUP/cpu.shares
fi

echo "Killing old spinners"
pkill spin
echo "Starting spinner"
$DIR/../spin/c/spin -t $(nproc) -i $N_ITER &
sleep 2s
SPIN_PID=$(pgrep spin)
echo "Adding $SPIN_PID to cgroup"
echo $SPIN_PID | sudo tee $CGROUP/cgroup.procs
echo "Added to cgroup"
