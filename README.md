# Prometheus Service

The *prometheus-service* is a Keptn service with two responsibilites: 
1. configures Prometheus for monitoring services managed by keptn,
1. receives alerts from Prometheus Alertmanager and translates the alert payload to a cloud event that is sent to the Keptn eventbroker.

## Installation

The *prometheus-service* is installed as a part of [Keptn](https://keptn.sh).

## Deploy in your Kubernetes cluster

To deploy the current version of the *prometheus-service* in your Keptn Kubernetes cluster, use the file `deploy/service.yaml` from this repository and apply it:

```console
kubectl apply -f deploy/service.yaml
```

## Delete in your Kubernetes cluster

To delete a deployed *prometheus-service*, use the file `deploy/service.yaml` from this repository and delete the Kubernetes resources:

```console
kubectl delete -f deploy/service.yaml
```