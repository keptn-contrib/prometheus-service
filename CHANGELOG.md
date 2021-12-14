# Changelog
All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.7.1](https://github.com/keptn-contrib/prometheus-service/compare/0.7.0...0.7.1) (2021-11-04)

This release enhances prometheus-service to use the distributor of Keptn 0.10.0.

### New Features

- Distributor of Keptn 0.10.0 is used
- Updated to keptn/go-utils 0.10.0

### [0.7.0](https://github.com/keptn-contrib/prometheus-service/compare/0.6.2...0.7.0) (2021-10-11)

This release conducts adoptions to support the auto-remediation use case with Keptn.

### New Features

- Allow environment variable for configuration-service #180
- Adaptations to support the auto-remediation use case #177

### [0.6.2](https://github.com/keptn-contrib/prometheus-service/compare/0.6.1...0.6.2) (2021-09-03)

This release enhances prometheus-service to use the `go-utils` package of Keptn 0.9.0.

### New Features

- `go-utils` package of Keptn 0.9.0 is used

### [0.6.1](https://github.com/keptn-contrib/prometheus-service/compare/0.6.0...0.6.1) (2021-06-18)

This release enhances prometheus-service to use the distributor of Keptn 0.8.4.

### New Features

- Distributor of Keptn 0.8.4 is used

### Known Limitations
- A know limitation is that the same keptn context is calculated for different problems [#60](https://github.com/keptn-contrib/prometheus-service/issues/60)


### [0.6.0](https://github.com/keptn-contrib/prometheus-service/compare/0.5.0...0.6.0) (2021-06-07)

This release merges the functionality of the `prometheus-sli-service` into the `prometheus-service`. As a result, it is not required to deploy the `prometheus-sli-service` for fetching SLI metrics.

:warning: The repository https://github.com/keptn-contrib/prometheus-sli-service is changed to read-only and will be archived. Further development of this functionality will happen here: https://github.com/keptn-contrib/prometheus-service

### New Features

- Adding Prometheus SLI functionalities to promtheus-services [#132](https://github.com/keptn-contrib/prometheus-service/pull/132)

### Known Limitations
- A know limitation is that the same keptn context is calculated for different problems [#60](https://github.com/keptn-contrib/prometheus-service/issues/60)


### [0.5.0](https://github.com/keptn-contrib/prometheus-service/compare/0.4.0...0.5.0) (2021-04-14)

### New Features
- Adding support for connecting external prometheus server and alert manager [#117](https://github.com/keptn-contrib/prometheus-service/pull/117)

### Known Limitations
- A know limitation is that the same keptn context is calculated for different problems [#60](https://github.com/keptn-contrib/prometheus-service/issues/60)


### Older Releases

You can find release notes of older releases here:

* [0.4.0](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.4.0)
* [0.3.6](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.6)
* [0.3.5](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.5)
* [0.3.4](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.4)
* [0.3.3](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.3)
* [0.3.2](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.2)
* [0.3.1](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.1)
* [0.3.0](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.3.0)
* [0.2.0](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.2.0)
* [0.1.0](https://github.com/keptn-contrib/prometheus-service/releases/tag/0.1.0)
