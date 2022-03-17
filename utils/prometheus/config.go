// Package prometheus contains copied code parts of the prometheus project (https://github.com/prometheus/prometheus)
// See the description of the different structures to get more information on where they have been extracted
// These definitions should match the structs in the supported prometheus version
package prometheus

import (
	"github.com/alecthomas/units"
	"github.com/prometheus/common/model"
	"net/url"
)

// UntypedElement identifies arbitrary data
type UntypedElement map[string]interface{}

// GlobalConfig describes the contents of the prometheus.yml file
// Origin: https://github.com/prometheus/prometheus/blob/fb2da1f26aec023b1e3c864222aaccbe01969f11/config/config.go#L335-L348
type GlobalConfig struct {
	ScrapeInterval     model.Duration    `mapstructure:"scrape_interval,omitempty" yaml:"scrape_interval,omitempty"`
	ScrapeTimeout      model.Duration    `mapstructure:"scrape_timeout,omitempty" yaml:"scrape_timeout,omitempty"`
	EvaluationInterval model.Duration    `mapstructure:"evaluation_interval,omitempty" yaml:"evaluation_interval,omitempty"`
	QueryLogFile       string            `mapstructure:"query_log_file,omitempty" yaml:"query_log_file,omitempty"`
	ExternalLabels     map[string]string `mapstructure:"external_labels,omitempty" yaml:"external_labels,omitempty"`
}

// Config is the root configure structure of Prometheus
// Origin: https://github.com/prometheus/prometheus/blob/fb2da1f26aec023b1e3c864222aaccbe01969f11/config/config.go#L221-L232
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

// ScrapeConfig configures a scraping unit for Prometheus.
// Origin: https://github.com/prometheus/prometheus/blob/fb2da1f26aec023b1e3c864222aaccbe01969f11/config/config.go#L405-L452
type ScrapeConfig struct {
	JobName               string           `mapstructure:"job_name" yaml:"job_name"`
	HonorLabels           bool             `mapstructure:"honor_labels,omitempty" yaml:"honor_labels,omitempty"`
	HonorTimestamps       bool             `mapstructure:"honor_timestamps" yaml:"honor_timestamps"`
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

// Configs is a type alias for an array of StaticConfigLike structs
type Configs []StaticConfigLike

// StaticConfigLike represents a static_config element in the Prometheus configuration
// This structure has been adapted from https://github.com/prometheus/prometheus/blob/53ac9d6d666acaebbdb8a004b51958f3147b1bd5/discovery/targetgroup/targetgroup.go#L23-L33
type StaticConfigLike struct {
	Targets []string       `mapstructure:"targets,omitempty" yaml:"targets,omitempty"`
	Labels  model.LabelSet `mapstructure:"labels,omitempty" yaml:"labels,omitempty"`
	Source  string         `mapstructure:"source,omitempty" yaml:"source,omitempty"`

	OtherElements UntypedElement `mapstructure:",remain" yaml:",inline"`
}
