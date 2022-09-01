# Performance Tests

You can execute test on GKE or Kind cluster. 

## GKE

1. Create GKE cluster with `Enable Kubernetes Network Policy`
2. Deploy [kuma](https://kuma.io/docs/1.8.x/networking/cni/#installation) 
   1. with CNI and iptables
 ```bash
  --set "cni.enabled=true" 
  --set "cni.chained=true" \
  --set "cni.netDir=/etc/cni/net.d" \
  --set "cni.binDir=/home/kubernetes/bin" \
  --set "cni.confName=10-calico.conflist"
  ```
  2. with eBPF

```bash
  --set "cni.enabled=true" 
  --set "cni.chained=true" \
  --set "cni.netDir=/etc/cni/net.d" \
  --set "cni.binDir=/home/kubernetes/bin" \
  --set "cni.confName=10-calico.conflist"
  --set "experimental.ebpf.enabled=true" 
  --set "cni.experimental.imageEbpf.registry=docker.io/merbridge" # bug in helm 1.8 fixed in 1.8+
```

3. Deploy sample services to k8s cluster

```bash
kubectl apply -f single-node/http-echo.yaml &&
kubectl apply -f single-node/wrk.yaml
```

4. Run performance test

```bash
kubectl exec -it deployment/wrk -n kuma-perf -c wrk -- wrk -c100 -t12 -d60s --latency http://$(kubectl get services/http-echo -n kuma-demo -o go-template='{{(.spec.clusterIP)}}'):5000/
```

## VM with Kind

1. Start Linux VM with atleast 4CPU and 40GB of Disk space
2. Install [`build-essential`](https://www.how2shout.com/linux/install-build-essential-tools-on-ubuntu-22-04-or-20-04-lts-linux/) and [`docker`](https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository)
3. Run `make install/all` to install all required dependencies
4. Start kind cluster `make kind/start`. By default, kind cluster starts only with one node. You can change it by setting `USE_MULTI_NODE` environment variable. 
5. Deploy kuma and test applications `make kind/deploy/ebpf` or `make kind/deploy/iptables`, depends which configuration you want to test. Default version of kuma is `1.8` but you can change it with environment `KUMA_VERSION`.
6. Run test `make run/perf`. You can tune configuration of `wrk` with environments variables 
```bash
WRK_CONN # number of connections
WRK_THREAD # number of thread >= number of connections
WRK_DURATION # duration of the test
``` 