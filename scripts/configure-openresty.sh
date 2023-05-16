#!/bin/bash

DIR=$(dirname $0)

sudo cp $DIR/../websrv/openresty/nginx.conf /etc/openresty/nginx.conf
sudo systemctl restart openresty
