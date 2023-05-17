#!/bin/python3 

import argparse

def cgroup_path(cgroup):
  return "/sys/fs/cgroup/cpu,cpuacct/system.slice/{}/cpuacct.usage".format(cgroup)

def get_ns(p):
 with open(p, "r") as f: 
   ns = int(str(f.read()).strip())
 return ns

def main():
  parser = argparse.ArgumentParser()
  parser.add_argument("--cgroup", type=str, required=True)
  parser.add_argument("--prev", type=str, default=None)

  args = parser.parse_args()

  cnt = get_ns(cgroup_path(args.cgroup))
  if args.prev is not None:
    prev = get_ns(args.prev)
    print("Nanoseconds difference: {}".format(cnt - prev))
  else:
    print("{}".format(cnt))

if __name__ == "__main__":
  main()
