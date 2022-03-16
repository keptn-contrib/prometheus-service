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

func TestLoad(t *testing.T) {
	config, err := LoadYamlConfiguration(sampleConfigurationYAML)
	require.NoError(t, err)

	require.NotNil(t, config.ScrapeConfigs)
	require.Len(t, config.ScrapeConfigs, 11)

	// Check if the "carts-sockshop-production" is correctly parsed
	scrapeConfig := config.ScrapeConfigs[10]
	assert.Equal(t, scrapeConfig.JobName, "carts-sockshop-production")
	assert.Equal(t, scrapeConfig.HonorTimestamps, false)

	duration5s, _ := model.ParseDuration("5s")
	duration3s, _ := model.ParseDuration("3s")
	assert.Equal(t, scrapeConfig.ScrapeInterval, duration5s)
	assert.Equal(t, scrapeConfig.ScrapeTimeout, duration3s)
	assert.Equal(t, scrapeConfig.MetricsPath, "/metrics")

	require.NotNil(t, scrapeConfig.StaticConfigs)
	require.Len(t, scrapeConfig.StaticConfigs, 1)
	require.NotNil(t, scrapeConfig.StaticConfigs[0].Targets)
	assert.Equal(t, scrapeConfig.StaticConfigs[0].Targets[0], "carts.sockshop-production:80")
}

func TestMinimalConfiguration(t *testing.T) {
	duration5s, _ := model.ParseDuration("5s")
	duration3s, _ := model.ParseDuration("3s")
	minConfig := Config{
		GlobalConfig: GlobalConfig{},
		ScrapeConfigs: []*ScrapeConfig{
			{
				JobName:         "carts-sockshop-production",
				HonorTimestamps: false,
				ScrapeInterval:  duration5s,
				ScrapeTimeout:   duration3s,
				MetricsPath:     "/metrics",
				StaticConfigs: []StaticConfigLike{
					{
						Targets: []string{"carts.sockshop-production:80"},
					},
				},
			},
		},
	}

	minConfigString := minConfig.String()
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

	assert.Equal(t, minConfigString, minConfigResult)
}

func TestSimpleLoadAndToString(t *testing.T) {
	config, err := LoadYamlConfiguration(sampleConfigurationYAML)

	require.NoError(t, err)
	require.NotNil(t, config)

	str := config.String()

	require.NotContainsf(t, str, "<error creating", "yaml marshaling should not contain error string")

	// Check if yaml configurations are equivalent
	var untypedYamlConfig map[string]interface{}
	err = yaml.Unmarshal([]byte(str), &untypedYamlConfig)
	require.NoError(t, err)

	var sampleUntypedYaml map[string]interface{}
	err = yaml.Unmarshal([]byte(sampleConfigurationYAML), &sampleUntypedYaml)
	require.NoError(t, err)

	assert.Equal(t, untypedYamlConfig, sampleUntypedYaml)
}
