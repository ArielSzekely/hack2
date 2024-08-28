# Experiments with network performance isolation using cgroups

## Kubernetes

### Build

Build the client and server binaries:

```
$ ./make.sh
```

### Start K8s

If running on Minikube, start with:
```
$ ./start-minikube.sh
```

### Run the App

Stop previous instances of the app with:
```
$ kubectl delete -Rf kubernetes
```

Start the app with:
```
$ kubectl apply -Rf kubernetes/app
```

Start the load balancer of your choice with:
```
$ kubectl apply -f kubernetes/lb/<DESIRED_LOAD_BALANCER>
```

Start the autoscaler with:
```
$ kubectl apply -f kubernetes/autoscale/<DESIRED_AUTOSCALER>
```

Wait for a bit for the app pods to start (10s is usually more than enough), and
the load balancer pods to find them.

If using minikube, get the IP address for the load-balancer (frontend) service
using:
```
$ minikube service spinhttp-lb --url 
```

### Generate Load

Using the `<IP_ADDR>` returned above, run the load-generator experiment with:
```
$ go clean -testcache
$ go test -timeout 0 -v elasticity/loadgen --dur 1s --exp_dur 180s --rps 23 --addr <IP_ADDR>
```
