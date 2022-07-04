package utils

// ServiceName holds the name of this service
const ServiceName = "prometheus-service"

// SliResourceURI holds the name of the SLI file that this service uses
const SliResourceURI = "prometheus/sli.yaml"

// EnvConfig holds the configuration of environment variables that this service uses
type EnvConfig struct {
	// Port on which to listen for cloudevents
	Port                          int    `envconfig:"RCV_PORT" default:"8080"`
	Path                          string `envconfig:"RCV_PATH" default:"/events"`
	ConfigurationServiceURL       string `envconfig:"CONFIGURATION_SERVICE" default:""`
	PrometheusNamespace           string `envconfig:"PROMETHEUS_NS" default:""`
	PrometheusConfigMap           string `envconfig:"PROMETHEUS_CM" default:""`
	PrometheusLabels              string `envconfig:"PROMETHEUS_LABELS" default:""`
	AlertManagerLabels            string `envconfig:"ALERT_MANAGER_LABELS" default:""`
	AlertManagerNamespace         string `envconfig:"ALERT_MANAGER_NS" default:""`
	AlertManagerConfigMap         string `envconfig:"ALERT_MANAGER_CM" default:""`
	AlertManagerTemplateConfigMap string `envconfig:"ALERT_MANAGER_TEMPLATE_CM" default:"alertmanager-templates"`
	PrometheusConfigFileName      string `envconfig:"PROMETHEUS_CONFIG_FILENAME" default:"prometheus.yml"`
	AlertManagerConfigFileName    string `envconfig:"ALERT_MANAGER_CONFIG_FILENAME" default:"alertmanager.yml"`
	PodNamespace                  string `envconfig:"POD_NAMESPACE" default:""`
	PrometheusEndpoint            string `envconfig:"PROMETHEUS_ENDPOINT" default:""`
	K8sNamespace                  string `envconfig:"K8S_NAMESPACE" required:"true"`
}
