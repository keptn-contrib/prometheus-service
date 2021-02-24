# Prometheus Service
![GitHub release (latest by date)](https://img.shields.io/github/v/release/keptn-contrib/prometheus-service)
[![Build Status](https://travis-ci.org/keptn-contrib/prometheus-service.svg?branch=master)](https://travis-ci.org/keptn-contrib/prometheus-service)
[![Go Report Card](https://goreportcard.com/badge/github.com/keptn-contrib/prometheus-service)](https://goreportcard.com/report/github.com/keptn-contrib/prometheus-service)

The *prometheus-service* is a [Keptn](https://keptn.sh) service that is responsible for

1. configuring Prometheus for monitoring services managed by Keptn, and
1. receiving alerts from Prometheus Alertmanager and translating the alert payload to a cloud event that is sent to the Keptn eventbroker.


## Compatibility Matrix

Please always double check the version of Keptn you are using compared to the version of this service, and follow the compatibility matrix below.


| Keptn Version    | [Prometheus Service Image](https://hub.docker.com/r/keptncontrib/prometheus-service/tags) |
|:----------------:|:----------------------------------------:|
|       0.5.x      | keptncontrib/prometheus-service:0.2.0  |
|       0.6.x      | keptncontrib/prometheus-service:0.3.0  |
|       0.6.1      | keptncontrib/prometheus-service:0.3.2  |
|       0.6.2      | keptncontrib/prometheus-service:0.3.4  |
|   0.7.0, 0.7.1   | keptncontrib/prometheus-service:0.3.5  |
|       0.7.2      | keptncontrib/prometheus-service:0.3.6  |
|   0.8.0-alpha    | keptncontrib/prometheus-service:0.4.0-alpha  |
|   0.8.0    | keptncontrib/prometheus-service:0.4.0  |

# Installation

The *prometheus-service* is installed as a part of [Keptn's uniform](https://keptn.sh). Please follow the instructions 
 below if you want to manually install it.
 
## Verify whether prometheus-service is already installed

```console
kubectl get deployments prometheus-service -n keptn
kubectl get pods -n keptn -l run=prometheus-service
```

## Deploy in your Kubernetes cluster

To deploy the current version of the *prometheus-service* in your Keptn Kubernetes cluster, use the `deploy/*.yaml` files from this repository and apply it:

```console
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/distributor.yaml
```

## Delete in your Kubernetes cluster

To delete a deployed *prometheus-service*, use the `deploy/*.yaml` files from this repository and delete the Kubernetes resources:

```console
kubectl delete -f deploy/distributor.yaml
kubectl delete -f deploy/service.yaml
```

# Contributions

You are welcome to contribute using Pull Requests against the **master** branch. Before contributing, please read our [Contributing Guidelines](CONTRIBUTING.md).

# Travis-CI setup

Travis is configured with CI to automatically build docker images for pull requests and commits. The  pipeline can be viewed at https://travis-ci.org/keptn-contrib/prometheus-service.
The travis pipeline needs to be configured with the `REGISTRY_USER` and `REGISTRY_PASSWORD` variables. 
