#!python3

import os
from tqdm import tqdm
import argparse
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def read_mcr(input):
  df = pd.read_csv(input)
  return df

def graph_mcr(df, output_dir):
  x = df["timestamp"] / 60000.0
  ms_names = [ms for ms in df.columns if "MS_" in ms ]
  for ms in ms_names:
    if np.count_nonzero(df[ms]) == 0:
      print("Skip graphing MCR for ms {}".format(ms))
      continue
    print("Graph MCR for ms {}".format(ms))
    plt.plot(x, df[ms])
    plt.savefig("{}.pdf".format(os.path.join(output_dir, ms)))
    plt.xlabel("Time (mins)")
    plt.ylabel("Normalized Microservice Call Rate")
    plt.clf()

def get_max_mcr_delta(mcrs, window_sizes):
  maxes = [0 for w in window_sizes]
  avgs = [0 for w in window_sizes]
  for i in range(len(mcrs)):
    if mcrs[i] > 0:
      for w_idx in range(len(window_sizes)):
        cur_max = 0
        all_gt_0 = True
        for j in range(window_sizes[w_idx] + 1):
          if mcrs[j] > 0:
            ratio = mcrs[j] / mcrs[i]
            if ratio > cur_max:
              cur_max = ratio
          else:
            all_gt_0 = False
        if cur_max > maxes[w_idx]:
          maxes[w_idx] = cur_max
        # Calc avg over window if all values in window were set
        if all_gt_0:
          avg_ratio = (sum(mcrs[i:window_sizes[w_idx]]) / float(window_sizes[w_idx])) / mcrs[i]
          if avg_ratio > avgs[w_idx]:
            avgs[w_idx] = avg_ratio
  return maxes + avgs

def max_mcr_delta(df):
  ms_names = [ms for ms in df.columns if "MS_" in ms ]
  window_sizes = [1, 5, 10]
  max_mcr_deltas = {}
  for ms in tqdm(ms_names):
    if np.count_nonzero(df[ms]) == 0:
      print("Skip calculating max MCR for ms {}".format(ms))
      continue
    print("Calculating max MCR for ms {}".format(ms))
    mcrs = df[ms]
    # 1 min, 5 min, 10 min
    deltas = get_max_mcr_delta(mcrs, window_sizes)
    if np.count_nonzero(deltas) == 0:
      print("Zero deltas for ms {}".format(ms))
      continue
    max_mcr_deltas[ms] = deltas
    print("Max MCR for ms {}: {}".format(ms, deltas))
  labels1 = ["max-{}mins".format(w) for w in window_sizes] 
  labels2 = ["max-avg-{}mins".format(w) for w in window_sizes] 
  return pd.DataFrame(max_mcr_deltas, index=(labels1 + labels2))

def main():
  parser = argparse.ArgumentParser()

  parser.add_argument("--input", type=str, required=True)
  parser.add_argument("--output_dir", type=str, required=True)
  args = parser.parse_args()

  df = read_mcr(args.input)
  max_mcr_deltas = max_mcr_delta(df)
  max_mcr_deltas.to_csv("~/alibaba-cluster-trace-max-mcr-delta.csv")
  print(max_mcr_deltas)

#  graph_mcr(df, args.output_dir)

if __name__ == "__main__":
  main()
