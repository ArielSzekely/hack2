#!/bin/bash

minikube start --extra-config 'controller-manager.horizontal-pod-autoscaler-sync-period=5s'
minikube addons enable metrics-server:w
minikube kubectl -- patch deployment -n kube-system metrics-server --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value":"--metric-resolution=10s"}]'
