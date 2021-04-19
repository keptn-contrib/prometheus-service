# Prometheus Service
![GitHub release (latest by date)](https://img.shields.io/github/v/release/keptn-contrib/prometheus-service)
[![Build Status](https://travis-ci.org/keptn-contrib/prometheus-service.svg?branch=master)](https://travis-ci.org/keptn-contrib/prometheus-service)
[![Go Report Card](https://goreportcard.com/badge/github.com/keptn-contrib/prometheus-service)](https://goreportcard.com/report/github.com/keptn-contrib/prometheus-service)

The *prometheus-service* is a [Keptn](https://keptn.sh) service that is responsible for

1. configuring Prometheus for monitoring services managed by Keptn, and
1. receiving alerts from Prometheus Alertmanager and translating the alert payload to a cloud event that is sent to the Keptn API.


## Compatibility Matrix

Please always double-check the version of Keptn you are using compared to the version of this service, and follow the compatibility matrix below.


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
|   0.8.1, 0.8.2    | keptncontrib/prometheus-service:0.5.0  |


## Setup Prometheus Monitoring

Keptn doesn't install or manage Prometheus and its components. Users need to install Prometheus and Prometheus Alert manager as a prerequisite.

Some environment variables have to set up in the prometheus-service deployment
```yaml
    # Prometheus installed namespace
    - name: PROMETHEUS_NS
      value: 'default'
    # Prometheus server configmap name
    - name: PROMETHEUS_CM
      value: 'prometheus-server'
    # Prometheus server app labels
    - name: PROMETHEUS_LABELS
      value: 'component=server'
    # Prometheus configmap data's config filename
    - name: PROMETHEUS_CONFIG_FILENAME
      value: 'prometheus.yml'
    # AlertManager configmap data's config filename
    - name: ALERT_MANAGER_CONFIG_FILENAME
      value: 'alertmanager.yml'
    # Alert Manager config map name
    - name: ALERT_MANAGER_CM
      value: 'prometheus-alertmanager'
    # Alert Manager app labels
    - name: ALERT_MANAGER_LABELS
      value: 'component=alertmanager'
    # Alert Manager installed namespace
    - name: ALERT_MANAGER_NS
      value: 'default'
    # Alert Manager template configmap name
    - name: ALERT_MANAGER_TEMPLATE_CM
      value: 'alertmanager-templates'
```

### Execute the following steps to install prometheus-service

* Download the Keptn's Prometheus service manifest
```bash
wget https://raw.githubusercontent.com/keptn-contrib/prometheus-service/release-0.4.0/deploy/service.yaml
```

* Replace the environment variable value according to the use case and apply the manifest
```bash
kubectl apply -f service.yaml
```

* Install Role and Rolebinding to permit Keptn's prometheus-service for performing operations in the Prometheus installed namespace.
```bash
kubectl apply -f https://raw.githubusercontent.com/keptn-contrib/prometheus-service/release-0.4.0/deploy/role.yaml -n <PROMETHEUS_NS>
```

* Execute the following command to install Prometheus and set up the rules for the *Prometheus Alerting Manager*:
```bash
keptn configure monitoring prometheus --project=sockshop --service=carts
```

### Optional: Verify Prometheus setup in your cluster

* To verify that the Prometheus scrape jobs are correctly set up, you can access Prometheus by enabling port-forwarding for the prometheus-server:
```bash
kubectl port-forward svc/prometheus-server 8080 -n <PROMETHEUS_NS>
```

Prometheus is then available on [localhost:8080/targets](http://localhost:8080/targets) where you can see the targets for the service.
# Contributions

You are welcome to contribute using Pull Requests against the **master** branch. Before contributing, please read our [Contributing Guidelines](CONTRIBUTING.md).

# Travis-CI setup

Travis is configured with CI to automatically build docker images for pull requests and commits. The pipeline can be viewed at https://travis-ci.org/keptn-contrib/prometheus-service.
The Travis pipeline needs to be configured with the `REGISTRY_USER` and `REGISTRY_PASSWORD` variables. 
