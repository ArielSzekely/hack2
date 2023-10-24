#!python3

import os
import sys
import ray
import time
import numpy as np

context = ray.init()
print(context.dashboard_url)

@ray.remote
def sleep(x):
    lat = time.time() - x
    return lat

def run_sleeper():
  ref = sleep.remote(time.time())
  return ray.get(ref)

N = 3

start = time.time()
start_latencies = [ run_sleeper() for i in range(N) ]
for l in start_latencies:
  print("Latency: {:.3f}ms".format(l * 1000.0))

start_latencies_ms = 1000.0 * np.array(start_latencies)

print("({} pid = {}) E2e latency (all tasks): {:.3f}ms\nTask start latency:\n  Mean: {:.3f}ms\n  Std: {:.3f}ms\n  Median: {:.3f}ms\n  Max: {:.3f}ms".format(
  sys.argv[0],
  os.getpid(),
  (time.time() - start) * 1000.0,
  np.mean(start_latencies_ms),
  np.std(start_latencies_ms),
  np.median(start_latencies_ms),
  max(start_latencies_ms),
  ))

time.sleep(30)
