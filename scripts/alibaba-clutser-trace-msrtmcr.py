#!python3

import os
from tqdm import tqdm
import argparse
import pandas as pd

def process_csv(input_dir, fn, nrows):
  df = pd.read_csv(os.path.join(input_dir, fn), nrows=nrows)
  res = df.pivot_table(
    index="timestamp",
    columns="msname",
    values="consumerrpc_mcr",
    aggfunc="sum",
    fill_value=0
  ).reset_index()
  return res

def aggregate_mcr(input_dir, nfiles, nrows):
  fnames = sorted(os.listdir(input_dir), key=lambda x: int(x[x.index("_")+1:x.index(".tar.gz")]))
  if nfiles is not None:
    fnames = fnames[:nfiles]
  print("reading input files: {}".format(fnames))
  dfs = [process_csv(input_dir, fn, nrows) for fn in tqdm(fnames)]
  df = pd.concat(dfs, axis=0)
  df.to_csv("~/alibaba-cluster-trace-msrtmcr.csv")

def main():
  parser = argparse.ArgumentParser()

  parser.add_argument("--nrows", type=int, default=None)
  parser.add_argument("--nfiles", type=int, default=None)
  parser.add_argument("--input_dir", type=str, required=True)
  args = parser.parse_args()

  aggregate_mcr(args.input_dir, args.nfiles, args.nrows)

if __name__ == "__main__":
  main()
