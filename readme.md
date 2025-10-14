# Kubecon NA 2025 - Kubernetes Network Driver (KND)

containerd
git checkout v2.1.4

kubernetes
v1.34.1

```sh
git checkout v0.30.0
make -C images/base quick EXTRA_BUILD_OPT="--build-arg CONTAINERD_CLONE_URL=https://github.com/LionelJouin/containerd --build-arg CONTAINERD_VERSION=kubecon-na-2025-knd --build-arg RUNC_VERSION=v1.4.0-rc.2 --no-cache" TAG=kubecon-na-2025-knd
```

```sh
kind build node-image . --image kindest/node:kubecon-na-2025-knd --base-image gcr.io/k8s-staging-kind/base:kubecon-na-2025-knd
```

Build and Push
```sh
make push-image REGISTRY=localhost:5000/kubecon-na-2025-knd VERSION=latest
```

Run
```sh
kubectl apply -f deployments/kubecon-na-2025-knd.yaml
kubectl set image daemonset/kubecon-na-2025-knd kubecon-na-2025-knd=localhost:5000/kubecon-na-2025-knd/kubecon-na-2025-knd:latest
kubectl apply -f docs/demo/deviceclass.yaml
```

```sh
docker exec -it kind-worker ip link add dummy0 type dummy
docker exec -it kind-worker ip link set dummy0 up
```

```sh
kubectl apply -f docs/demo/deployment.yaml
```


https://github.com/containerd/nri/issues/180


docker exec -it kind-worker ip link add dummy0 type dummy