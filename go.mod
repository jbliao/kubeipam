module github.com/jbliao/kubeipam

go 1.13

require (
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.5
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/runtime v0.19.15
	github.com/go-openapi/strfmt v0.19.5
	github.com/netbox-community/go-netbox v0.0.0-20200325111416-50e92f3e2076
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.10.0
	gonum.org/v1/netlib v0.0.0-20190331212654-76723241ea4e // indirect
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06 // indirect
)

replace github.com/netbox-community/go-netbox v0.0.0-20200325111416-50e92f3e2076 => github.com/kobayashi/go-netbox v0.0.0-20200507032154-fbb6900a912a
