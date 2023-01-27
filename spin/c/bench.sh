#!/bin/bash

# Calculate number of cores for high & low priority processes.
ncores=$(nproc)
lopri_ncores=1
hipri_ncores=$(($ncores-$lopri_ncores))

# Benchmark parameters.
prog=spin
niter=2000000000
nthread=$hipri_ncores

# CPU sets
hipri_cpu_list=$lopri_ncores-$(($ncores - 1))
lowpri_cpu_list=0-$(($lopri_ncores - 1))

echo "===== baseline hipri ====="
time -p ./$prog --nthread $nthread --niter $niter

sleep 2

echo "===== baseline hipri taskset $hipri_cpu_list ====="
time -p taskset --cpu-list $hipri_cpu_list ./$prog --nthread $nthread --niter $niter

sleep 2

echo "===== lowpri few threads, hipri taskset $hipri_cpu_list lowpri taskset $lowpri_cpu_list ====="
time -p taskset --cpu-list $hipri_cpu_list ./$prog --nthread $nthread --niter $niter &
time -p taskset --cpu-list $lowpri_cpu_list ./$prog --nthread 1 --niter $niter &
wait

echo "===== both many threads, hipri taskset $hipri_cpu_list lowpri taskset $lowpri_cpu_list ====="
time -p taskset --cpu-list $hipri_cpu_list ./$prog --nthread $nthread --niter $niter &
time -p taskset --cpu-list $lowpri_cpu_list ./$prog --nthread $nthread --niter $niter &
wait
