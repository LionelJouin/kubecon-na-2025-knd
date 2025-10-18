# Documentation

## Build the Kubernetes Node image for Kind

Build the Kind base image with:
- runc supporting the the `linux.netDevices` field to allow for devices to be moved into container network namespaces seamlessly ([opencontainers/runc#4538](https://github.com/opencontainers/runc/pull/4538)). This is supported from runc v1.4.0-rc.1.
- containerd containing the NRI changes to allow adjusting linux net devices ([containerd/nri#157](https://github.com/containerd/nri/pull/157)). This is not yet available in containerd, so a fork based on containerd v2.1.4 is being used ([LionelJouin/containerd#linux-net-device-support](https://github.com/LionelJouin/containerd/commits/linux-net-device-support)).
```sh
# Clone Kind
git clone git@github.com:kubernetes-sigs/kind.git && cd kind
# Checkout to the latest version
git checkout v0.30.0
# Build the base image
make -C images/base quick EXTRA_BUILD_OPT="--build-arg CONTAINERD_CLONE_URL=https://github.com/LionelJouin/containerd --build-arg CONTAINERD_VERSION=linux-net-device-support --build-arg RUNC_VERSION=v1.4.0-rc.2 --no-cache" TAG=kubecon-na-2025-knd
```

Build Kubernetes node image for Kind with the base image built previously
```sh
# Clone Kubernetes
git clone git@github.com:kubernetes/kubernetes.git && cd kubernetes
# Checkout to the latest version
git checkout v1.34.1
# Build the Kubernetes node image for Kind
kind build node-image . --image kindest/node:kubecon-na-2025-knd --base-image gcr.io/k8s-staging-kind/base:kubecon-na-2025-knd
# The image created is kindest/node:kubecon-na-2025-knd, it must now be set in the /docs/demo/kind.yaml file.
```

## Build and run the Kubecon-NA-2025-KND driver

Build and Push the kubecon-na-2025-knd image
```sh
make push-image REGISTRY=localhost:5000/kubecon-na-2025-knd VERSION=latest
```

Create and configure the Kind Cluster
```sh
# Create a Kind Cluster
kind create cluster --config docs/demo/kind.yaml
# Add Dummy interfaces in the kind-worker
docker exec -it kind-worker ip link add dummy0 type dummy
docker exec -it kind-worker ip link set dummy0 up
```

Deploy the driver
```sh
# Deploy the Kubecon-NA-2025-KND driver
kubectl apply -f deployments/kubecon-na-2025-knd.yaml
# Set the image to previously created one
kubectl set image daemonset/kubecon-na-2025-knd kubecon-na-2025-knd=localhost:5000/kubecon-na-2025-knd/kubecon-na-2025-knd:latest
# Create the device class
kubectl apply -f docs/demo/deviceclass.yaml
```

Deploy the demo
```sh
kubectl apply -f docs/demo/pod.yaml
```
