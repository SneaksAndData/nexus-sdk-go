# Introduction
Nexus SDK Go is a Golang development kit for Nexus client applications. Base client code is generated using `ogen` from OpenAPI v3 specification of the Nexus API.

SDK is tested against a Nexus stack deployed to a `kind` cluster (WIP).

## Local testing with `kind`

1. Create a controller cluster:
```shell
kind create cluster --name nexus-controller
```

2. Create a shard cluster:
```shell
kind create cluster --name nexus-shard-0
```

3. Export kubeconfigs:
```shell
kind export kubeconfig --name nexus-controller --kubeconfig ./test-resources/kubecfg/controller
kind export kubeconfig --name nexus-shard-0 --kubeconfig ./test-resources/kubecfg/shards/kind-nexus-shard-0.kubeconfig
```

4. Install CRDs into both clusters:
```shell
cd .helm
helm dependency build .
helm --kube-context kind-nexus-controller install nexus-test-stack . --create-namespace --namespace nexus

sleep 5
helm --kube-context kind-nexus-shard-0 install nexus-test-stack . --create-namespace --namespace nexus
```