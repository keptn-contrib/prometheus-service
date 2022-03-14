package prometheus

import (
	"fmt"
	"github.com/alecthomas/units"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"
	"net/url"
)

// UntypedElement identifies arbitrary data
type UntypedElement map[string]interface{}

type GlobalConfig struct {
	ScrapeInterval     model.Duration    `mapstructure:"scrape_interval,omitempty" yaml:"scrape_interval,omitempty"`
	ScrapeTimeout      model.Duration    `mapstructure:"scrape_timeout,omitempty" yaml:"scrape_timeout,omitempty"`
	EvaluationInterval model.Duration    `mapstructure:"evaluation_interval,omitempty" yaml:"evaluation_interval,omitempty"`
	QueryLogFile       string            `mapstructure:"query_log_file,omitempty" yaml:"query_log_file,omitempty"`
	ExternalLabels     map[string]string `mapstructure:"external_labels,omitempty" yaml:"external_labels,omitempty"`
}

func (c GlobalConfig) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating global_config string: %s>", err)
	}
	return string(b)
}

// Config is the root configure structure of Prometheus
type Config struct {
	GlobalConfig   GlobalConfig    `mapstructure:"global" yaml:"global"`
	AlertingConfig UntypedElement  `mapstructure:"alerting,omitempty" yaml:"alerting,omitempty"`
	RuleFiles      []string        `mapstructure:"rule_files,omitempty" yaml:"rule_files,omitempty"`
	ScrapeConfigs  []*ScrapeConfig `mapstructure:"scrape_configs,omitempty" yaml:"scrape_configs,omitempty"`
	StorageConfig  UntypedElement  `mapstructure:"storage,omitempty" yaml:"storage,omitempty"`
	TracingConfig  UntypedElement  `mapstructure:"tracing,omitempty" yaml:"tracing,omitempty"`

	RemoteWriteConfigs []*UntypedElement `mapstructure:"remote_write,omitempty" yaml:"remote_write,omitempty"`
	RemoteReadConfigs  []*UntypedElement `mapstructure:"remote_read,omitempty" yaml:"remote_read,omitempty"`
}

func (c Config) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating static_config string: %s>", err)
	}
	return string(b)
}

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	JobName               string           `mapstructure:"job_name" yaml:"job_name"`
	HonorLabels           bool             `mapstructure:"honor_labels,omitempty" yaml:"honor_labels,omitempty"`
	HonorTimestamps       bool             `mapstructure:"honor_timestamps" yaml:"honor_timestamps,omitempty"`
	Params                url.Values       `mapstructure:"params,omitempty" yaml:"params,omitempty"`
	ScrapeInterval        model.Duration   `mapstructure:"scrape_interval,omitempty" yaml:"scrape_interval,omitempty"`
	ScrapeTimeout         model.Duration   `mapstructure:"scrape_timeout,omitempty" yaml:"scrape_timeout,omitempty"`
	MetricsPath           string           `mapstructure:"metrics_path,omitempty" yaml:"metrics_path,omitempty"`
	Scheme                string           `mapstructure:"scheme,omitempty" yaml:"scheme,omitempty"`
	BodySizeLimit         units.Base2Bytes `mapstructure:"body_size_limit,omitempty" yaml:"body_size_limit,omitempty"`
	SampleLimit           uint             `mapstructure:"sample_limit,omitempty" yaml:"sample_limit,omitempty"`
	TargetLimit           uint             `mapstructure:"target_limit,omitempty" yaml:"target_limit,omitempty"`
	LabelLimit            uint             `mapstructure:"label_limit,omitempty" yaml:"label_limit,omitempty"`
	LabelNameLengthLimit  uint             `mapstructure:"label_name_length_limit,omitempty" yaml:"label_name_length_limit,omitempty"`
	LabelValueLengthLimit uint             `mapstructure:"label_value_length_limit,omitempty" yaml:"label_value_length_limit,omitempty"`

	StaticConfigs        []StaticConfigLike `mapstructure:"static_configs,omitempty" yaml:"static_configs,omitempty"`
	RelabelConfigs       []*UntypedElement  `mapstructure:"relabel_configs,omitempty" yaml:"relabel_configs,omitempty"`
	MetricRelabelConfigs []*UntypedElement  `mapstructure:"metric_relabel_configs,omitempty" yaml:"metric_relabel_configs,omitempty"`

	RemainingElements UntypedElement `mapstructure:",remain" yaml:",inline"`
}

func (c ScrapeConfig) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating scrape_config string: %s>", err)
	}
	return string(b)
}

type Configs []StaticConfigLike

// StaticConfigLike represents a static_config element or any other _config element in the Prometheus configuration
type StaticConfigLike struct {
	Targets []string       `mapstructure:"targets,omitempty" yaml:"targets,omitempty"`
	Labels  model.LabelSet `mapstructure:"labels,omitempty" yaml:"labels,omitempty"`
	Source  string         `mapstructure:"source,omitempty" yaml:"source,omitempty"`

	OtherElements UntypedElement `mapstructure:",remain" yaml:",inline"`
}

func (c StaticConfigLike) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating static_config string: %s>", err)
	}
	return string(b)
}
