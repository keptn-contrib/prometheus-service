package prometheus

import (
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

const sampleConfigurationYAML = `global:
  scrape_interval: 1m
  scrape_timeout: 10s
  evaluation_interval: 1m
alerting:
  alertmanagers:
  - kubernetes_sd_configs:
    - role: pod
    bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
    tls_config:
      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      insecure_skip_verify: false
    scheme: http
    timeout: 10s
    api_version: v1
    relabel_configs:
    - source_labels: [__meta_kubernetes_namespace]
      separator: ;
      regex: monitoring
      replacement: $1
      action: keep
    - source_labels: [__meta_kubernetes_pod_label_app]
      separator: ;
      regex: prometheus
      replacement: $1
      action: keep
rule_files:
- /etc/config/recording_rules.yml
- /etc/config/alerting_rules.yml
- /etc/config/rules
- /etc/config/alerts
scrape_configs:
- job_name: prometheus
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - localhost:9090
- job_name: kubernetes-apiservers
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  kubernetes_sd_configs:
  - role: endpoints
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
    separator: ;
    regex: default;kubernetes;https
    replacement: $1
    action: keep
- job_name: kubernetes-nodes
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  kubernetes_sd_configs:
  - role: node
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  relabel_configs:
  - separator: ;
    regex: __meta_kubernetes_node_label_(.+)
    replacement: $1
    action: labelmap
  - separator: ;
    regex: (.*)
    target_label: __address__
    replacement: kubernetes.default.svc:443
    action: replace
  - source_labels: [__meta_kubernetes_node_name]
    separator: ;
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/$1/proxy/metrics
    action: replace
- job_name: kubernetes-nodes-cadvisor
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  kubernetes_sd_configs:
  - role: node
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  relabel_configs:
  - separator: ;
    regex: __meta_kubernetes_node_label_(.+)
    replacement: $1
    action: labelmap
  - separator: ;
    regex: (.*)
    target_label: __address__
    replacement: kubernetes.default.svc:443
    action: replace
  - source_labels: [__meta_kubernetes_node_name]
    separator: ;
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
    action: replace
- job_name: kubernetes-service-endpoints
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
    separator: ;
    regex: "true"
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape_slow]
    separator: ;
    regex: "true"
    replacement: $1
    action: drop
- job_name: kubernetes-service-endpoints-slow
  honor_timestamps: true
  scrape_interval: 5m
  scrape_timeout: 30s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape_slow]
    separator: ;
    regex: "true"
    replacement: $1
    action: keep
- job_name: prometheus-pushgateway
  honor_labels: true
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: service
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
    separator: ;
    regex: pushgateway
    replacement: $1
    action: keep
- job_name: kubernetes-services
  honor_timestamps: true
  params:
    module:
    - http_2xx
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /probe
  scheme: http
  kubernetes_sd_configs:
  - role: service
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
    separator: ;
    regex: "true"
    replacement: $1
    action: keep
- job_name: kubernetes-pods
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_phase]
    separator: ;
    regex: Pending|Succeeded|Failed|Completed
    replacement: $1
    action: drop
- job_name: kubernetes-pods-slow
  honor_timestamps: true
  scrape_interval: 5m
  scrape_timeout: 30s
  metrics_path: /metrics
  scheme: http
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_slow]
    separator: ;
    regex: "true"
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scheme]
    separator: ;
    regex: (https?)
    target_label: __scheme__
    replacement: $1
    action: replace
- job_name: carts-sockshop-production
  honor_timestamps: false
  scrape_interval: 5s
  scrape_timeout: 3s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - carts.sockshop-production:80
`

func TestLoadComplexConfig(t *testing.T) {
	config, err := LoadYamlConfiguration(sampleConfigurationYAML)
	require.NoError(t, err)

	require.NotNil(t, config.ScrapeConfigs)
	require.Len(t, config.ScrapeConfigs, 11)

	duration5s, _ := model.ParseDuration("5s")
	duration3s, _ := model.ParseDuration("3s")
	expectedScrapeConfig := ScrapeConfig{
		JobName:         "carts-sockshop-production",
		HonorTimestamps: false,
		ScrapeInterval:  duration5s,
		ScrapeTimeout:   duration3s,
		MetricsPath:     "/metrics",
		Scheme:          "http",
		StaticConfigs: []StaticConfigLike{
			{
				Targets: []string{"carts.sockshop-production:80"},
			},
		},
	}

	assert.Equal(t, config.ScrapeConfigs[10], &expectedScrapeConfig)
	assert.Contains(t, config.ScrapeConfigs, &expectedScrapeConfig)
}

func TestToStringMinimalConfiguration(t *testing.T) {
	minConfig := Config{
		GlobalConfig: GlobalConfig{},
		ScrapeConfigs: []*ScrapeConfig{
			generateScrapeConfig("carts-sockshop-production", "carts.sockshop-production:80"),
		},
	}

	minConfigString, err := yaml.Marshal(minConfig)
	require.NoError(t, err)

	minConfigResult := `global: {}
scrape_configs:
- job_name: carts-sockshop-production
  honor_timestamps: false
  scrape_interval: 5s
  scrape_timeout: 3s
  metrics_path: /metrics
  static_configs:
  - targets:
    - carts.sockshop-production:80
`

	assert.Equal(t, string(minConfigString), minConfigResult)
}

func TestLoadAndMarshal(t *testing.T) {
	config, err := LoadYamlConfiguration(sampleConfigurationYAML)
	require.NoError(t, err)
	require.NotNil(t, config)

	configYamlString, err := yaml.Marshal(config)
	require.NoError(t, err)

	err = compareYamlEq(t, string(configYamlString), sampleConfigurationYAML)
	require.NoError(t, err)
}

func TestModificationWithoutChangingExistingContent(t *testing.T) {
	yamlConfig := `global: {}
scrape_configs:
    - job_name: kubernetes-pods-slow
      honor_timestamps: true
      scrape_interval: 5m
      scrape_timeout: 30s
      metrics_path: /metrics
      scheme: http
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_slow]
        separator: ;
        regex: "true"
        replacement: $1
        action: keep
    - job_name: carts-sockshop-production
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 3s
      metrics_path: /metrics
      static_configs:
        - targets:
            - carts.sockshop-production:80
`

	config, err := LoadYamlConfiguration(yamlConfig)
	require.NoError(t, err)

	config.ScrapeConfigs = append(config.ScrapeConfigs, generateScrapeConfig("carts-sockshop-dev", "carts.sockshop-dev:80"))
	config.ScrapeConfigs = append(config.ScrapeConfigs, generateScrapeConfig("carts-sockshop-staging", "carts.sockshop-staging:80"))
	config.ScrapeConfigs = append(config.ScrapeConfigs, generateScrapeConfig("carts-sockshop-production-canary", "carts.sockshop-production-canary:80"))

	modifiedConfigYaml, err := yaml.Marshal(config)
	require.NoError(t, err)

	resultingConfigYaml := `global: {}
scrape_configs:
    - job_name: kubernetes-pods-slow
      honor_timestamps: true
      scrape_interval: 5m
      scrape_timeout: 30s
      metrics_path: /metrics
      scheme: http
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_slow]
        separator: ;
        regex: "true"
        replacement: $1
        action: keep
    - job_name: carts-sockshop-production
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 3s
      metrics_path: /metrics
      static_configs:
        - targets:
            - carts.sockshop-production:80
    - job_name: carts-sockshop-dev
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 3s
      metrics_path: /metrics
      static_configs:
        - targets:
            - carts.sockshop-dev:80
    - job_name: carts-sockshop-staging
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 3s
      metrics_path: /metrics
      static_configs:
        - targets:
            - carts.sockshop-staging:80
    - job_name: carts-sockshop-production-canary
      honor_timestamps: false
      scrape_interval: 5s
      scrape_timeout: 3s
      metrics_path: /metrics
      static_configs:
        - targets:
            - carts.sockshop-production-canary:80
`

	err = compareYamlEq(t, string(modifiedConfigYaml), resultingConfigYaml)
	assert.NoError(t, err)
}

func compareYamlEq(t *testing.T, a string, b string) error {
	var untypedYamlA map[string]interface{}
	if err := yaml.Unmarshal([]byte(a), &untypedYamlA); err != nil {
		return err
	}

	var untypedYamlB map[string]interface{}
	if err := yaml.Unmarshal([]byte(b), &untypedYamlB); err != nil {
		return err
	}

	assert.Equal(t, untypedYamlA, untypedYamlB)
	return nil
}

func generateScrapeConfig(jobName string, target string) *ScrapeConfig {
	duration5s, _ := model.ParseDuration("5s")
	duration3s, _ := model.ParseDuration("3s")

	return &ScrapeConfig{
		JobName:         jobName,
		HonorTimestamps: false,
		ScrapeInterval:  duration5s,
		ScrapeTimeout:   duration3s,
		MetricsPath:     "/metrics",
		StaticConfigs: []StaticConfigLike{
			{
				Targets: []string{target},
			},
		},
	}
}
