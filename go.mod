module github.com/jbliao/kubeipam

go 1.13

require (
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.5
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/runtime v0.19.15
	github.com/netbox-community/go-netbox v0.0.0-20200507032154-fbb6900a912a
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.10.0
	gopkg.in/intel/multus-cni.v3 v3.4.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)

replace github.com/netbox-community/go-netbox v0.0.0-20200507032154-fbb6900a912a => github.com/kobayashi/go-netbox v0.0.0-20200507051839-fff5f7ecc242
