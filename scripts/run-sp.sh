#!/bin/bash

DIR=$(dirname $0)

N_ITER=40000000000
CGROUP=/sys/fs/cgroup/cpu,cpuacct/system.slice/spin

if ! [ -d $CGROUP ]; then
  echo "Make cgroup"
  sudo mkdir $CGROUP
fi

echo "Killing old spinners"
pkill spin
echo "Starting spinner"
$DIR/../spin/c/spin -t $(nproc) -i $N_ITER &
sleep 2s
echo "Setting cgroup shares"
echo 100 | sudo tee $CGROUP/cpu.shares
SPIN_PID=$(pgrep spin)
echo "Adding $SPIN_PID to cgroup"
echo $SPIN_PID | sudo tee $CGROUP/cgroup.procs
echo "Added to cgroup"
sudo chrt -a -i -p $SPIN_PID
