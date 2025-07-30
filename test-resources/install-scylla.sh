#!/usr/bin/env bash

set -e

helm repo add jetstack https://charts.jetstack.io --force-update
helm repo add scylla https://scylla-operator-charts.storage.googleapis.com/stable --force-update

helm --kube-context kind-nexus-controller install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.18.2 --set crds.enabled=true
sleep 15
helm --kube-context kind-nexus-controller install scylla-operator scylla/scylla-operator --namespace scylla-operator --create-namespace --version 1.17.1
sleep 30
helm --kube-context kind-nexus-controller install scylla scylla/scylla --namespace scylla --create-namespace --version 1.17.1 --set developerMode=true --set datacenter=us-east-1 \
--set racks[0].name=us-east-1b --set members=1 --set storage.capacity=5Gi --set storage.storageClassName=standard \
--set racks[0].resources.limits.memory=1Gi --set racks[0].resources.limits.cpu=1000m --set racks[0].resources.requests.memory=1Gi --set racks[0].resources.requests.cpu=1000m

sleep 60

kubectl --context kind-nexus-controller -n scylla get pods

kubectl --context kind-nexus-controller -n scylla apply -f ./test-resources/scylla-lb-service.yaml

sleep 15
