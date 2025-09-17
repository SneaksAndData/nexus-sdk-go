![coverage](https://raw.githubusercontent.com/SneaksAndData/nexus-sdk-go/badges/.badges/main/coverage.svg)

# Introduction
Nexus SDK Go is a Golang development kit for Nexus client applications. Base client code is generated using `ogen` from OpenAPI v3 specification of the Nexus API.

SDK is tested against a Nexus stack deployed to a `kind` cluster (WIP).

## Local testing with `kind` and `docker`

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

# update permissions
chmod 777 ./test-resources/kubecfg/controller
chmod 777 ./test-resources/kubecfg/shards/kind-nexus-shard-0.kubeconfig
```

4. Install CRDs into both clusters:
```shell
cd .helm
helm dependency build .
helm --kube-context kind-nexus-controller install nexus-test-stack . --create-namespace --namespace nexus
helm --kube-context kind-nexus-shard-0 install nexus-test-stack . --create-namespace --namespace nexus
```

5. Launch docker-compose stack
```shell
docker compose up --quiet-pull -d
```

**Important** If running Docker Desktop on MacOS, make sure to enable host networking under `Settings` -> `Resources` -> `Enable host networking`

Now, try to access the scheduler API:
```shell
> curl -vvv http://localhost:8080/algorithm/v1/results/tags/aaa
```

and check for a response like below:
```text
* Host localhost:8080 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:8080...
* Connected to localhost (::1) port 8080
> GET /algorithm/v1/results/tags/aaa HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 04 Aug 2025 08:23:44 GMT
< Content-Length: 2
```
