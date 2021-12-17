# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.7.2](https://github.com/keptn-contrib/prometheus-service/compare/0.7.1...0.7.2) (2021-12-17)


### Features

* **install:** use helm charts for installing ([#231](https://github.com/keptn-contrib/prometheus-service/issues/231)) ([01e7679](https://github.com/keptn-contrib/prometheus-service/commit/01e76791a8f8f8419c37054565ba8c54219c5e6f))


### Bug Fixes

* **core:** alert manager template does not need to be created, it usually already exists ([#232](https://github.com/keptn-contrib/prometheus-service/issues/232)) ([fc104df](https://github.com/keptn-contrib/prometheus-service/commit/fc104df047f4c49b152ae02d017915792f5aaaba))
* **core:** Fix event forwarding to localhost ([#195](https://github.com/keptn-contrib/prometheus-service/issues/195)) ([#203](https://github.com/keptn-contrib/prometheus-service/issues/203)) ([c24c27e](https://github.com/keptn-contrib/prometheus-service/commit/c24c27efabd08eb61a3ef09544ade9f2e2973f25))
* **core:** fix event sending and added keptncontext ([#60](https://github.com/keptn-contrib/prometheus-service/issues/60)) ([117a503](https://github.com/keptn-contrib/prometheus-service/commit/117a50312f1df4c15a9d3aa0cbc542f126dd4c83))


### Other

* Add pre-release and release workflows, restructure CI test/build steps ([#222](https://github.com/keptn-contrib/prometheus-service/issues/222)) ([34823fa](https://github.com/keptn-contrib/prometheus-service/commit/34823fafd4e816c61419fac91269a95b8d1b3649))
* remove eventbroker from code ([#229](https://github.com/keptn-contrib/prometheus-service/issues/229)) ([b86311d](https://github.com/keptn-contrib/prometheus-service/commit/b86311d423235ae67cf5ab205ece99bc7f228c31))
* renovate should use semantic commits ([#238](https://github.com/keptn-contrib/prometheus-service/issues/238)) ([44488ab](https://github.com/keptn-contrib/prometheus-service/commit/44488abcea2cd4ef4323e288dfc6fe3833099f93))
* update codeowners and readme ([#221](https://github.com/keptn-contrib/prometheus-service/issues/221)) ([de93089](https://github.com/keptn-contrib/prometheus-service/commit/de93089ee589a0836b1434bd8775eb852b2890a4))
* Use validate-semantic-pr workflow from keptn/gh-automation repo ([#220](https://github.com/keptn-contrib/prometheus-service/issues/220)) ([4a02e0b](https://github.com/keptn-contrib/prometheus-service/commit/4a02e0bb9f10b8f89151d6f1ec5bea927bf44af3))

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
