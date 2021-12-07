# Prometheus Service
![GitHub release (latest by date)](https://img.shields.io/github/v/release/keptn-contrib/prometheus-service)
[![Build Status](https://travis-ci.org/keptn-contrib/prometheus-service.svg?branch=master)](https://travis-ci.org/keptn-contrib/prometheus-service)
[![Go Report Card](https://goreportcard.com/badge/github.com/keptn-contrib/prometheus-service)](https://goreportcard.com/report/github.com/keptn-contrib/prometheus-service)

The *prometheus-service* is a [Keptn](https://keptn.sh) service that is responsible for:

1. configuring Prometheus for monitoring services managed by Keptn, and
2. receiving alerts from Prometheus Alertmanager and translating the alert payload to a cloud event that is sent to the Keptn API.
3. It's used for retrieving Service Level Indicators (SLIs) from a Prometheus API endpoint. Per default, it fetches metrics from the prometheus instance set up by Keptn
   (`prometheus-service.monitoring.svc.cluster.local:8080`), but it can also be configured to use any reachable Prometheus endpoint using basic authentication by providing the credentials
   via a secret in the `keptn` namespace of the cluster.

    The supported default SLIs are:

    - throughput
    - error_rate
    - response_time_p50
    - response_time_p90
    - response_time_p95

The provided SLIs are based on the [RED metrics](https://grafana.com/files/grafanacon_eu_2018/Tom_Wilkie_GrafanaCon_EU_2018.pdf)

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


## Setup Prometheus Monitoring

Keptn does not install or manage Prometheus and its components. Users need to install Prometheus and Prometheus Alert manager as a prerequisite.

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

* Download the Keptn Prometheus service manifest:

```bash
wget https://raw.githubusercontent.com/keptn-contrib/prometheus-service/release-0.7.0/deploy/service.yaml
```

* Replace the environment variable value according to the use case and apply the manifest:

```bash
kubectl apply -f service.yaml
```

* Install Role and RoleBinding to permit prometheus-service for performing operations in the Prometheus installed namespace:

```bash
kubectl apply -f https://raw.githubusercontent.com/keptn-contrib/prometheus-service/release-0.7.0/deploy/role.yaml -n <PROMETHEUS_NS>
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
