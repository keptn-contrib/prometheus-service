module github.com/keptn-contrib/prometheus-service

go 1.13

require (
	cloud.google.com/go v0.44.1 // indirect
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/armon/go-metrics v0.0.0-20190430140413-ec5e00d3c878 // indirect
	github.com/cloudevents/sdk-go/v2 v2.3.1
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/keptn/go-utils v0.6.3-0.20201217115623-17e6dc4eb089
	github.com/keptn/kubernetes-utils v0.1.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/prometheus v0.0.0-20200326161412-ae041f97cfc6
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
