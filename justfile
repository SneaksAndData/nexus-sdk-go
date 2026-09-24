set shell := ["bash", "-c"]

# images
SCYLLA_IMAGE := "scylladb/scylla"
MINIO_IMAGE  := "quay.io/minio/minio"

# configurations
MANIFESTS := invocation_directory() / "test-resources/manifests"
DBSCHEMA := invocation_directory() / "test-resources/e2e"

# Nexus charts and images
SCHEDULER_IMAGE_REPO := "ghcr.io/sneaksanddata/nexus"
RECEIVER_IMAGE_REPO := "ghcr.io/sneaksanddata/nexus-receiver"

NEXUS_CHART_NAME := "oci://ghcr.io/sneaksanddata/helm/nexus"
NEXUS_RECEIVER_CHART_NAME := "oci://ghcr.io/sneaksanddata/helm/nexus-receiver"
NEXUS_VERSION := "1.2.3-2-g7cf0ab7"
NEXUS_RECEIVER_VERSION := "1.2.0"
NEXUS_CRD_VERSION := "1.1.0"

# cluster
NEXUS_CLUSTER_NAME := "nexus-sdk-go-tests"

# Default recipe
fresh: stop up

# Start CI environment
up: start-kind-cluster install-ingress-controller create-namespace create-ingress scylla-kind s2 crd apply-manifests dbschema scheduler receiver

start-kind-cluster:
    kind create cluster --config=test-resources/kind.yaml --name {{NEXUS_CLUSTER_NAME}}

# Run all tests with coverage
test-all:
    APPLICATION_ENVIRONMENT=units NEXUS_TEST_SCHEDULER_URL=http://localhost:5555/scheduler NEXUS_TEST_RECEIVER_URL=http://localhost:5555/receiver go test -v ./... -coverprofile=cover.out -covermode=atomic -coverpkg=./...

# Cleanup CI environment
stop:
    @echo "🧹 Cleaning up..."
    kind delete cluster --name {{NEXUS_CLUSTER_NAME}}
    rm -f cover-indexed.out cover-bare.out cover.out

# View logs
logs name="":
    docker logs -f {{if name == "" { "scylla" } else { name }}}

create-namespace:
    kubectl create namespace nexus --dry-run=client -o yaml | kubectl apply -f -

# install chart
scheduler:
    kubectl create secret generic cassandra-credentials \
        --namespace nexus \
        --from-literal=NEXUS__SCYLLA_CQL_STORE__HOSTS="scylla.nexus.svc.cluster.local" \
        --from-literal=NEXUS__SCYLLA_CQL_STORE__INDEXES_SUPPORTED="true" \
        --from-literal=NEXUS__SCYLLA_CQL_STORE__USER="cassandra" \
        --from-literal=NEXUS__SCYLLA_CQL_STORE__PASSWORD="cassandra" \
        --from-literal=NEXUS__SCYLLA_CQL_STORE__KEYSPACE="nexus" --dry-run=client -o yaml | kubectl apply -f -

    kubectl create secret generic nexus-s3 \
        --namespace nexus \
        --from-literal=NEXUS__S3_BUFFER__REGION="us-east-1" \
        --from-literal=NEXUS__S3_BUFFER__ACCESS_KEY_ID="minioadmin" \
        --from-literal=NEXUS__S3_BUFFER__SECRET_ACCESS_KEY="minioadmin" \
        --from-literal=NEXUS__S3_BUFFER__ENDPOINT="http://minio.nexus.svc.cluster.local:9000" --dry-run=client -o yaml | kubectl apply -f -

    kubectl create secret generic nexus-sign-key \
        --namespace nexus \
        --from-literal=NEXUS__S3_BUFFER__REQUEST_PAYLOAD_PROXY_CONFIGURATION__SIGN_SECRET="test" --dry-run=client -o yaml | kubectl apply -f -

    helm upgrade nexus {{NEXUS_CHART_NAME}} --install --create-namespace --namespace nexus --version v{{NEXUS_VERSION}} \
        --set image.repository={{SCHEDULER_IMAGE_REPO}} \
        --set image.tag={{NEXUS_VERSION}} \
        --set scheduler.config.checkpointStore.type=cassandra-scylla \
        --set scheduler.config.checkpointStore.secretName="cassandra-credentials" \
        --set scheduler.config.s3Buffer.s3Credentials.secretName="nexus-s3" \
        --set scheduler.config.s3Buffer.processing.payloadProxy.externalName="nexus.nexus.svc.cluster.local:8080" \
        --set scheduler.config.s3Buffer.processing.payloadProxy.insecure="true" \
        --set scheduler.config.logLevel="DEBUG"
    kubectl -n nexus rollout status deployment/nexus --timeout=180s

receiver:
    helm upgrade nexus-receiver {{NEXUS_RECEIVER_CHART_NAME}} --install --create-namespace --namespace nexus --version v{{NEXUS_RECEIVER_VERSION}} \
        --set image.repository={{RECEIVER_IMAGE_REPO}} \
        --set image.tag={{NEXUS_RECEIVER_VERSION}} \
        --set receiver.config.checkpointStore.type=cassandra-scylla \
        --set receiver.config.checkpointStore.secretName="cassandra-credentials" \
        --set receiver.config.logLevel="DEBUG"
    kubectl -n nexus rollout status deployment/nexus-receiver --timeout=180s

# cleanup
remove-chart:
    helm uninstall -n nexus nexus

install-ingress-controller:
    kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
    kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=180s

create-ingress:
    # Create ingress rules for services
    for i in $(seq 1 30); do \
      kubectl apply -f {{MANIFESTS}}/ingress.yaml && break || \
      (echo "Retry $i/30: failed to apply ingress, retrying in 1s..." && sleep 1); \
    done; \
    if [ $i -eq 30 ]; then \
      echo "Failed to apply ingress after 30 attempts."; \
      exit 1; \
    fi

scylla-kind:
    kubectl apply -f {{MANIFESTS}}/scylladb.yaml
    kubectl -n nexus rollout status deployment/scylla --timeout=180s

s2:
    kubectl apply -f {{MANIFESTS}}/s2.yaml
    kubectl -n nexus rollout status deployment/minio --timeout=180s

crd:
    helm upgrade --install --namespace nexus nexus-crd  oci://ghcr.io/sneaksanddata/helm/nexus-crd --version v{{NEXUS_CRD_VERSION}}

apply-manifests:
    kubectl apply -n nexus -f {{MANIFESTS}}/nexus-algorithm-sa.yaml
    kubectl apply -n nexus -f {{MANIFESTS}}/hello-world-workgroup.yaml
    kubectl apply -n nexus -f {{MANIFESTS}}/hello-world-algorithm.yaml

dbschema:
  docker run --rm -v {{DBSCHEMA}}:/opt/storage --network=host --entrypoint /opt/storage/prepare-db.sh {{SCYLLA_IMAGE}}