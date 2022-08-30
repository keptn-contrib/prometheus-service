# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.9.1](https://github.com/keptn-contrib/prometheus-service/compare/0.9.0...0.9.1) (2022-08-30)


### Features

* Reimplement service with go-sdk ([#358](https://github.com/keptn-contrib/prometheus-service/issues/358)) ([dbb3576](https://github.com/keptn-contrib/prometheus-service/commit/dbb357614041e16999fd0026a56b3dd7c5bbf8b0))


### Bug Fixes

* Deployment Type in remediation.triggered incompatible with Keptn cloud events ([#366](https://github.com/keptn-contrib/prometheus-service/issues/366)) ([af1db26](https://github.com/keptn-contrib/prometheus-service/commit/af1db26db108b3233ff4212b787f49045c65c108))

## [0.9.0](https://github.com/keptn-contrib/prometheus-service/compare/0.8.5...0.9.0) (2022-08-29)


### ⚠ BREAKING CHANGES

* Keptn 0.18 compatibility (#363)

### Features

* Keptn 0.18 compatibility ([#363](https://github.com/keptn-contrib/prometheus-service/issues/363)) ([1288de5](https://github.com/keptn-contrib/prometheus-service/commit/1288de5580c086d135d968d4d10794638e9becd0))
* Switch to resource-service in order to guarantee compat. with Keptn 0.18 and newer ([#361](https://github.com/keptn-contrib/prometheus-service/issues/361)) ([d33a1a5](https://github.com/keptn-contrib/prometheus-service/commit/d33a1a5e08672fde1cdd1f6ccd60881990d8e84c))

### [0.8.5](https://github.com/keptn-contrib/prometheus-service/compare/0.8.4...0.8.5) (2022-08-17)


### Features

* Autodetect prometheus and alertmanager namespaces ([#349](https://github.com/keptn-contrib/prometheus-service/issues/349)) ([e607d4b](https://github.com/keptn-contrib/prometheus-service/commit/e607d4b62c93dc82f6430c1adc82a07028405c9e))


### Bug Fixes

* Alertmanager cloudevent payload ([#353](https://github.com/keptn-contrib/prometheus-service/issues/353)) ([55caac4](https://github.com/keptn-contrib/prometheus-service/commit/55caac456f98314d1af1bda93ad552daee8419c5))
* Auto-detection feature ([#350](https://github.com/keptn-contrib/prometheus-service/issues/350)) ([7748961](https://github.com/keptn-contrib/prometheus-service/commit/774896105abba2c77db933a68bfd69eba12e90c7))
* Default deployment in alerts/remediations to "primary" if not specified ([#355](https://github.com/keptn-contrib/prometheus-service/issues/355)) ([6dffc14](https://github.com/keptn-contrib/prometheus-service/commit/6dffc14b31e07baf3a7f5542cf4ece23f6a6ae55))


### Other

* Update Readme.md for release of 0.8.5 ([#356](https://github.com/keptn-contrib/prometheus-service/issues/356)) ([cf7e32d](https://github.com/keptn-contrib/prometheus-service/commit/cf7e32d19a657570dbb0d5a7b8e40e8ab9b90e08))

### [0.8.4](https://github.com/keptn-contrib/prometheus-service/compare/0.8.3...0.8.4) (2022-07-18)


### Features

* Allow customization of the installation namespace ([#328](https://github.com/keptn-contrib/prometheus-service/issues/328)) ([6ca8cf8](https://github.com/keptn-contrib/prometheus-service/commit/6ca8cf8969fec57e578f4f78d2c978f7f7852db7))
* Disable automatic creation of Prometheus alerts and targets ([#327](https://github.com/keptn-contrib/prometheus-service/issues/327)) ([9ed3eb2](https://github.com/keptn-contrib/prometheus-service/commit/9ed3eb2f9bd11d2944cae1bf5a028eda9182cb1c))
* Integration tests ([#333](https://github.com/keptn-contrib/prometheus-service/issues/333)) ([44e9691](https://github.com/keptn-contrib/prometheus-service/commit/44e9691ced4c53cda54a3fce2b3fb9c51249e02d))
* Post integration test summary to GH workflow ([#340](https://github.com/keptn-contrib/prometheus-service/issues/340)) ([0d4da6d](https://github.com/keptn-contrib/prometheus-service/commit/0d4da6def060d4d4f4911643fb15470c1c957d85))
* Upgrade to Keptn 0.17 ([#345](https://github.com/keptn-contrib/prometheus-service/issues/345)) ([d83a956](https://github.com/keptn-contrib/prometheus-service/commit/d83a956617121269ed945d79fdc78c9410971893))
* Use Helm build action ([#334](https://github.com/keptn-contrib/prometheus-service/issues/334)) ([1ab1bca](https://github.com/keptn-contrib/prometheus-service/commit/1ab1bcad9b6d97a061119016156911211679244b))


### Bug Fixes

* Integration tests ([#339](https://github.com/keptn-contrib/prometheus-service/issues/339)) ([a5112a6](https://github.com/keptn-contrib/prometheus-service/commit/a5112a65d373f80ea6b8c4a627b4d777b328b03e))

### [0.8.3](https://github.com/keptn-contrib/prometheus-service/compare/0.8.2...0.8.3) (2022-06-30)


### Bug Fixes

* Compatibility issues with alertmanager configuration ([#330](https://github.com/keptn-contrib/prometheus-service/issues/330)) ([278617e](https://github.com/keptn-contrib/prometheus-service/commit/278617ec31c0e5085d4481bdeb5df0a39bf91fa1))


### Docs

* Update Compatibility matrix in README.md ([#331](https://github.com/keptn-contrib/prometheus-service/issues/331)) ([13ec763](https://github.com/keptn-contrib/prometheus-service/commit/13ec763114dfb3837c64d0e7185be52fb7520aae))

### [0.8.2](https://github.com/keptn-contrib/prometheus-service/compare/0.8.1...0.8.2) (2022-06-29)


### Features

* Upgrade to Keptn 0.16 ([#323](https://github.com/keptn-contrib/prometheus-service/issues/323)) ([1045f30](https://github.com/keptn-contrib/prometheus-service/commit/1045f302dc4bbc5959c79947959653bbec607f39))

### [0.8.1](https://github.com/keptn-contrib/prometheus-service/compare/0.8.0...0.8.1) (2022-06-21)


### Features

* Add role binding error handling ([#307](https://github.com/keptn-contrib/prometheus-service/issues/307)) ([9e3b607](https://github.com/keptn-contrib/prometheus-service/commit/9e3b607ad6199d9166846665ad44077bc6fafd53))
* Upgrade to Keptn 0.15.1 ([#319](https://github.com/keptn-contrib/prometheus-service/issues/319)) ([ec85241](https://github.com/keptn-contrib/prometheus-service/commit/ec85241e93fa5067754d876a3818038d52434d42))
* Use Keptn secrets for storing external prometheus-server URL ([#308](https://github.com/keptn-contrib/prometheus-service/issues/308)) ([5e35109](https://github.com/keptn-contrib/prometheus-service/commit/5e35109c30efaa8bb10d9401c485f240bc54986c))


### Bug Fixes

* **docs:** Release version in compatibility matrix ([#321](https://github.com/keptn-contrib/prometheus-service/issues/321)) ([d1a0c30](https://github.com/keptn-contrib/prometheus-service/commit/d1a0c305badc16acb660d4fc9cbf92a7d0c4159f))
* sanitize criteria string for alerts by removing white spaces ([#233](https://github.com/keptn-contrib/prometheus-service/issues/233)) ([f7bcb51](https://github.com/keptn-contrib/prometheus-service/commit/f7bcb5161cbb4750ee9a568e3012dd33560306b3))


### Other

* Remove ingress, cleanup helm chart ([#309](https://github.com/keptn-contrib/prometheus-service/issues/309)) ([aac5014](https://github.com/keptn-contrib/prometheus-service/commit/aac50149464da8508caea5768207ad6e8be43ebf))

## [0.8.0](https://github.com/keptn-contrib/prometheus-service/compare/0.7.4...0.8.0) (2022-05-11)


### ⚠ BREAKING CHANGES

* This release requires Keptn 0.14.2 or newer to be installed.

Signed-off-by: Christian Kreuzberger <christian.kreuzberger@dynatrace.com>

### Features

* Upgrade to Keptn 0.13 ([#296](https://github.com/keptn-contrib/prometheus-service/issues/296)) ([7b04146](https://github.com/keptn-contrib/prometheus-service/commit/7b041462c5d51729adfe8efee9b503729a606be0))
* Upgrade to Keptn 0.14 ([5d75b1f](https://github.com/keptn-contrib/prometheus-service/commit/5d75b1fea57a8a3e123242032d3dcb495a179d3e))


### Bug Fixes

* Queries with no data points ([#297](https://github.com/keptn-contrib/prometheus-service/issues/297)) ([1dffad4](https://github.com/keptn-contrib/prometheus-service/commit/1dffad48281c25d0ec4a621535d7bc271831fa3b))


### Docs

* compatibility with Keptn 0.14.2 ([#305](https://github.com/keptn-contrib/prometheus-service/issues/305)) ([e98a002](https://github.com/keptn-contrib/prometheus-service/commit/e98a0021d5a95de87f2073a4e0d6536252e650b3))

### [0.7.4](https://github.com/keptn-contrib/prometheus-service/compare/0.7.3...0.7.4) (2022-04-26)


### Features

* Allow scrape_interval configuration ([#286](https://github.com/keptn-contrib/prometheus-service/issues/286)) ([32ca698](https://github.com/keptn-contrib/prometheus-service/commit/32ca69859c8b353ffe88b6731441fa4d361c33ea))
* Reimplement prometheus configuration parsing ([#282](https://github.com/keptn-contrib/prometheus-service/issues/282)) ([bb693ec](https://github.com/keptn-contrib/prometheus-service/commit/bb693ecab8877f51def821c5a132cf10400bf5f7))
* Remove unneeded Kubernetes role bindings ([#281](https://github.com/keptn-contrib/prometheus-service/issues/281)) ([8c10544](https://github.com/keptn-contrib/prometheus-service/commit/8c105440a3fc281c13b0040a27ef30cca345648c))
* Use cloudevent for alerting endpoint ([#270](https://github.com/keptn-contrib/prometheus-service/issues/270)) ([4d93045](https://github.com/keptn-contrib/prometheus-service/commit/4d9304589babc8bd29e3e13c5efef3a8a0a3f37d))
* Utilize official prometheus api for querying SLI values ([#288](https://github.com/keptn-contrib/prometheus-service/issues/288)) ([dfc6c24](https://github.com/keptn-contrib/prometheus-service/commit/dfc6c24cedc4fa3c7b74d4003a80fa0ebb5183b2))


### Docs

* Added note about incompatibility with Keptn 0.14.x ([#293](https://github.com/keptn-contrib/prometheus-service/issues/293)) ([7020465](https://github.com/keptn-contrib/prometheus-service/commit/7020465d89c60d6c370b9da93f287fcf6488df28))

### [0.7.3](https://github.com/keptn-contrib/prometheus-service/compare/0.7.2...0.7.3) (2022-02-21)


### Features

* Allow labels and deployment types as placeholders for SLIs ([#265](https://github.com/keptn-contrib/prometheus-service/issues/265)) ([ac86eed](https://github.com/keptn-contrib/prometheus-service/commit/ac86eed9e6be7423afb23b34b3401826bfc8acc3))
* Keptn 0.12 compatibility ([#269](https://github.com/keptn-contrib/prometheus-service/issues/269)) ([8bc2180](https://github.com/keptn-contrib/prometheus-service/commit/8bc2180b328351eafd40e4e599c15359febf6db3))
* only add prometheus alerts if remediation.yaml is defined for the stage ([#253](https://github.com/keptn-contrib/prometheus-service/issues/253)) ([#255](https://github.com/keptn-contrib/prometheus-service/issues/255)) ([05dae73](https://github.com/keptn-contrib/prometheus-service/commit/05dae73f24a5a19b36605303267049316e68e3cb))


### Bug Fixes

* installing role and rolebinding from correct tag ([#257](https://github.com/keptn-contrib/prometheus-service/issues/257)) ([2f41b77](https://github.com/keptn-contrib/prometheus-service/commit/2f41b77ba7b9db103e5f1be3050b431bef30508d))


### Refactoring

* Do not delete prometheus pods after configuring ConfigMaps ([#252](https://github.com/keptn-contrib/prometheus-service/issues/252)) ([1202556](https://github.com/keptn-contrib/prometheus-service/commit/12025565524992998454eae6ea3631eb955023af))
* move SLI retrieval into eventhandling, re-use prometheus metric fetching ([#264](https://github.com/keptn-contrib/prometheus-service/issues/264)) ([97adeec](https://github.com/keptn-contrib/prometheus-service/commit/97adeec26cd390e83bf053bf93bc5ceb9280c03c))
* use built-in send event functionality of keptn/go-utils ([#267](https://github.com/keptn-contrib/prometheus-service/issues/267)) ([908b4d5](https://github.com/keptn-contrib/prometheus-service/commit/908b4d5234438a5618cfae4bfe6a4a137a0089de))


### Other

* added a way to test prometheus alerts ([#271](https://github.com/keptn-contrib/prometheus-service/issues/271)) ([ac6e9c4](https://github.com/keptn-contrib/prometheus-service/commit/ac6e9c479a644e8bdc8fad14ba19b7a909484829))

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
