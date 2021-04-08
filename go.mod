module github.com/keptn-contrib/prometheus-service

go 1.13

require (
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/armon/go-metrics v0.0.0-20190430140413-ec5e00d3c878 // indirect
	github.com/cloudevents/sdk-go/v2 v2.3.1
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/keptn/go-utils v0.8.0
	github.com/mitchellh/mapstructure v1.2.2 // indirect
	github.com/prometheus/alertmanager v0.21.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/prometheus v0.0.0-20200326161412-ae041f97cfc6
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
