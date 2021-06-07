# Release Notes

This release merges the functionality of the `prometheus-sli-service` into the `prometheus-service`. As a result, it is not required to deploy the `prometheus-sli-service` for fetching SLI metrics. 

:warning: The repository https://github.com/keptn-contrib/prometheus-sli-service is changed to read-only and will be archived. Further development of this functionality will happen here: https://github.com/keptn-contrib/prometheus-service

## New Features

- Adding Prometheus SLI functionalities to promtheus-services [#132](https://github.com/keptn-contrib/prometheus-service/pull/132)

## Known Limitations
- A know limitation is that the same keptn context is calculated for different problems [#60](https://github.com/keptn-contrib/prometheus-service/issues/60)
