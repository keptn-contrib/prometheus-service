# Prometheus Service
![GitHub release (latest by date)](https://img.shields.io/github/v/release/keptn-contrib/prometheus-service)
[![Go Report Card](https://goreportcard.com/badge/github.com/keptn-contrib/prometheus-service)](https://goreportcard.com/report/github.com/keptn-contrib/prometheus-service)

The *prometheus-service* is a [Keptn](https://keptn.sh) integration responsible for:

1. configuring Prometheus for monitoring services managed by Keptn, 
2. receiving alerts (on port 8080) from Prometheus Alertmanager and translating the alert payload to a cloud event (remediation.triggered) that is sent to the Keptn API,
3. retrieving Service Level Indicators (SLIs) from a Prometheus API endpoint. 

## Compatibility Matrix

Please always double-check the version of Keptn you are using compared to the version of this service, and follow the compatibility matrix below.


| Keptn Version    | [Prometheus Service Image](https://hub.docker.com/r/keptncontrib/prometheus-service/tags) |
|:----------------:|:-----------------------------------------------------------------------------------------:|
|       0.5.x      |                           keptncontrib/prometheus-service:0.2.0                           |
|       0.6.x      |                           keptncontrib/prometheus-service:0.3.0                           |
|       0.6.1      |                           keptncontrib/prometheus-service:0.3.2                           |
|       0.6.2      |                           keptncontrib/prometheus-service:0.3.4                           |
|   0.7.0, 0.7.1   |                           keptncontrib/prometheus-service:0.3.5                           |
|       0.7.2      |                           keptncontrib/prometheus-service:0.3.6                           |
|   0.8.0-alpha    |                        keptncontrib/prometheus-service:0.4.0-alpha                        |
|   0.8.0          |                           keptncontrib/prometheus-service:0.4.0                           |
|   0.8.1, 0.8.2   |                           keptncontrib/prometheus-service:0.5.0                           |
|   0.8.1 - 0.8.3  |                           keptncontrib/prometheus-service:0.6.0                           |
|   0.8.4 - 0.8.7  |                           keptncontrib/prometheus-service:0.6.1                           |
|       0.9.0      |                           keptncontrib/prometheus-service:0.6.2                           |
|   0.9.0 - 0.9.2  |                           keptncontrib/prometheus-service:0.7.0                           |
|   0.10.0         |                           keptncontrib/prometheus-service:0.7.1                           |
|   0.10.0         |                           keptncontrib/prometheus-service:0.7.2                           |


## Installation instructions

### Setup Prometheus Monitoring

Keptn does not install or manage Prometheus and its components. Users need to install Prometheus and Prometheus Alert manager as a prerequisite.

The easiest way would be to setup Prometheus using helm, e.g.:
```console
kubectl create namespace monitoring
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/prometheus --namespace monitoring
```

**Note**: After setting up prometheus, make sure to apply [deploy/role.yaml](deploy/role.yaml) such that prometheus-service can access the `monitoring` namespace (see instructions below).


### Install prometheus-service

Please replace the placeholders in the commands below. Examples are provided.

* `<VERSION>`: prometheus-service version, e.g., `0.7.1`
* `<PROMETHEUS_NS>`: If prometheus is installed in the same Kubernetes cluster, the namespace needs to be provided, e.g., `monitoring`
* `<PROMETHEUS_ENDPOINT>`: Endpoint for prometheus (primarily used for fetching metrics), e.g., `http://prometheus-server.monitoring.svc.cluster.local:80`
* `<ALERT_MANAGER_NS>`: if prometheus alert manager is installed in the same Kubernetes cluster, the namespace needs to be provided, e.g., `monitoring`


Once this is done, you can go ahead and install prometheus-service:


* Install Keptn prometheus-service in Kubernetes using

```bash
helm install -n keptn prometheus-service https://github.com/keptn-contrib/prometheus-service/releases/download/<VERSION>/prometheus-service-<VERSION>.tgz
# or
helm upgrade --install -n keptn prometheus-service https://github.com/keptn-contrib/prometheus-service/releases/download/<VERSION>/prometheus-service-<VERSION>.tgz
```

Prior to version 0.7.2 installation should be done via `kubectl`:
```bash
kubectl apply -f https://raw.githubusercontent.com/keptn-contrib/prometheus-service/release-<VERSION>/deploy/service.yaml
```

* Install Role and RoleBinding to permit prometheus-service for performing operations in the Prometheus installed namespace:

```bash
kubectl -n <PROMETHEUS_NS> apply -f https://raw.githubusercontent.com/keptn-contrib/prometheus-service/<VERSION>/deploy/role.yaml
```


* (Optional) Replace the environment variable value according to the use case and apply the manifest:

```bash
helm upgrade -n keptn prometheus-service https://github.com/keptn-contrib/prometheus-service/releases/download/<VERSION>/prometheus-service-<VERSION>.tgz --reuse-values --set=prometheus.namespace="<PROMETHEUS_NS>",prometheus.endpoint="<PROMETHEUS_ENDPOINT>",prometheus.namespace_am="<ALERT_MANAGER_NS>"
```


Prior to version 0.7.2 setting variables should be done via `kubectl`:
```
# Prometheus installed namespace
kubectl set env deployment/prometheus-service -n keptn --containers="prometheus-service" PROMETHEUS_NS="<PROMETHEUS_NS>"

# Setup Prometheus Endpoint
kubectl set env deployment/prometheus-service -n keptn --containers="prometheus-service" PROMETHEUS_ENDPOINT="<PROMETHEUS_ENDPOINT>"

# Alert Manager installed namespace
kubectl set env deployment/prometheus-service -n keptn --containers="prometheus-service" ALERT_MANAGER_NS="<ALERT_MANAGER_NS>"
```


* Execute the following command to configure Prometheus and set up the rules for the *Prometheus Alerting Manager*:

```bash
keptn configure monitoring prometheus --project=sockshop --service=carts
```

### Optional: Verify Prometheus setup in your cluster

* To verify that the Prometheus scrape jobs are correctly set up, you can access Prometheus by enabling port-forwarding for the prometheus-server:

```bash
kubectl port-forward svc/prometheus-server 8080:80 -n <PROMETHEUS_NS>
```

Prometheus is then available on [localhost:8080/targets](http://localhost:8080/targets) where you can see the targets for the service.


### Advanced Options

You can customize prometheus-service with the following environment variables:

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

## Prometheus SLI provider

Per default, the service works with the following assumptions regarding the setup of the Prometheus instance:

- Each **service** within a **stage** of a **project** has a Prometheus scrape job definition with the name: `<service>-<project>-<stage>`

  For example, if `project=sockshop`, `stage=production` and `service=carts`, the scrape job name would have to be `carts-sockshop-production`.

- Every service provides the following metrics for its corresponding scrape job:
    - http_response_time_milliseconds (Histogram)
    - http_requests_total (Counter)

      This metric has to contain the `status` label, indicating the HTTP response code of the requests handled by the service.
      It is highly recommended that this metric also provides a label to query metric values for specific endpoints, e.g. `handler`.

      An example of an entry would look like this: `http_requests_total{method="GET",handler="VersionController.getInformation",status="200",} 4.0`

- Based on those metrics, the queries for the SLIs are built as follows:

    - **throughput**: `sum(rate(http_requests_total{job="<service>-<project>-<stage>-canary"}[<test_duration_in_seconds>s]))`
    - **error_rate**: `sum(rate(http_requests_total{job="<service>-<project>-<stage>-canary",status!~'2..'}[<test_duration_in_seconds>s]))/sum(rate(http_requests_total{job="<service>-<project>-<stage>-canary"}[<test_duration_in_seconds>s]))`
    - **response_time_p50**: `histogram_quantile(0.50, sum(rate(http_response_time_milliseconds_bucket{job='<service>-<project>-<stage>-canary'}[<test_duration_in_seconds>s])) by (le))`
    - **response_time_p90**: `histogram_quantile(0.90, sum(rate(http_response_time_milliseconds_bucket{job='<service>-<project>-<stage>-canary'}[<test_duration_in_seconds>s])) by (le))`
    - **response_time_p95**: `histogram_quantile(0.95, sum(rate(http_response_time_milliseconds_bucket{job='<service>-<project>-<stage>-canary'}[<test_duration_in_seconds>s])) by (le))`

## Advanced Usage

### Using an external Prometheus instance

To use a Prometheus instance other than the one that is being managed by Keptn for a certain project, a secret containing the URL and the access credentials has to be deployed into the `keptn` namespace. The secret must have the following format:

```yaml
user: test
password: test
url: http://prometheus-service.monitoring.svc.cluster.local:8080
```

If this information is stored in a file, e.g. `prometheus-creds.yaml`, it can be stored with the following command (don't forget to replace the `<project>` placeholder with the name of your project:

```console
kubectl create secret -n keptn generic prometheus-credentials-<project> --from-file=prometheus-credentials=./mock_secret.yaml
```

Please note that there is a naming convention for the secret, because this can be configured per **project**. Therefore, the secret has to have the name `prometheus-credentials-<project>`

### Custom SLI queries

Users can override the predefined queries, as well as add custom queries by creating a SLI configuration.

* A SLI configuration is a yaml file as shown below:

    ```yaml
    ---
    spec_version: '1.0'
    indicators:
      cpu_usage: avg(rate(container_cpu_usage_seconds_total{namespace="$PROJECT-$STAGE",pod_name=~"$SERVICE-primary-.*"}[5m]))
      response_time_p95: histogram_quantile(0.95, sum by(le) (rate(http_response_time_milliseconds_bucket{handler="ItemsController.addToCart",job="$SERVICE-$PROJECT-$STAGE-canary"}[$DURATION_SECONDS])))
    ```

* To store this configuration, you need to add this file to a Keptn's configuration store. This is done by using the Keptn CLI with the [keptn add-resource](https://keptn.sh/docs/0.8.x/reference/cli/commands/keptn_add-resource/) command (see [SLI Provider](https://keptn.sh/docs/0.8.x/quality_gates/sli-provider/) for more information).

---

Within the user-defined queries, the following variables can be used to dynamically build the query, depending on the project/stage/service, and the time frame:

- $PROJECT: will be replaced with the name of the project
- $STAGE: will be replaced with the name of the stage
- $SERVICE: will be replaced with the name of the service
- $DURATION_SECONDS: will be replaced with the test run duration, e.g. 30s

For example, if an evaluation for the service **carts**  in the stage **production** of the project **sockshop** is triggered, and the tests ran for 30s these will be the resulting queries:

```
rate(my_custom_metric{job='$SERVICE-$PROJECT-$STAGE',handler=~'$handler'}[$DURATION_SECONDS]) => rate(my_custom_metric{job='carts-sockshop-production',handler=~'$handler'}[30s])
```

# Contributions

You are welcome to contribute using Pull Requests against the **master** branch. Before contributing, please read our [Contributing Guidelines](CONTRIBUTING.md).
