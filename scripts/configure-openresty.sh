#!/bin/bash

DIR=$(dirname $0)

sudo cp $DIR/../websrv/openresty/nginx.conf /etc/openresty/nginx.conf
sudo systemctl restart openresty

echo "4096" | sudo tee /sys/fs/cgroup/cpu\,cpuacct/system.slice/openresty.service/cpu.shares
