#!/bin/bash

# Start minikube
minikube start --extra-config 'controller-manager.horizontal-pod-autoscaler-sync-period=5s' #--extra-config 'kube-proxy.proxy-mode=ipvs'
# Enable metrics server addon for minikube
minikube addons enable metrics-server
# Increase metrics server's metrics scraping frequency to the maximum (1/10s)
minikube kubectl -- patch deployment -n kube-system metrics-server --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value":"--metric-resolution=10s"}]'
# Create service account for LB proc to access kubernetes API
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default
