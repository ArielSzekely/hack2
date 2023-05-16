#!/bin/bash

echo "Stopping nginx"
sudo systemctl stop nginx

echo "Installing nginx config"
sudo cp ./nginx.conf /etc/nginx/

echo "Starting nginx"
sudo systemctl start nginx

echo "Running server..."
spawn-fcgi -n -p 9000 -- $(pwd)/srv -w 1
