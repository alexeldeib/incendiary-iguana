module github.com/alexeldeib/incendiary-iguana

go 1.12

require (
	github.com/Azure/azure-sdk-for-go v31.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.5.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/autorest/validation v0.1.0 // indirect
	github.com/apex/log v1.1.1
	github.com/autonomy/conform v0.1.0-alpha.16 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.3.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.2.0
	github.com/sanity-io/litter v1.1.0
	github.com/satori/go.uuid v1.2.0
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.0-beta.4
	sigs.k8s.io/controller-tools v0.2.0-beta.4 // indirect
)

replace sigs.k8s.io/controller-runtime => github.com/alexeldeib/controller-runtime v0.2.0-beta.4-master-opts
