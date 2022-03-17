package prometheus

import (
	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
	"reflect"
)

// LoadYamlConfiguration parses the given yaml configuration of the prometheus.yaml file
func LoadYamlConfiguration(yamlContent string) (*Config, error) {
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &result); err != nil {
		return nil, err
	}

	var config Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			toDurationDecoderHook(),
		),
		Result: &config,
	})

	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(result); err != nil {
		return nil, err
	}

	return &config, nil
}

func toDurationDecoderHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if t != reflect.TypeOf((*model.Duration)(nil)).Elem() {
			return data, nil
		}

		return model.ParseDuration(data.(string))
	}
}
