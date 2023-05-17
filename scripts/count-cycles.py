#!/bin/python3 

import argparse

def count_cgroup_cycles(cgroup):
 with open("/sys/fs/cgroup/cpu,cpuacct/system.slice/{}/cpuacct.usage".format(cgroup), "r") as f: 
   ns = int(str(f.read()).strip())
 print(ns)

def main():
  parser = argparse.ArgumentParser()
  parser.add_argument("--cgroup", type=str, required=True)

  args = parser.parse_args()

  count_cgroup_cycles(args.cgroup)

if __name__ == "__main__":
  main()
