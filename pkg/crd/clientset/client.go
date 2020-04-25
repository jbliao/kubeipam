package clientset

import (
	"context"

	ipamv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IPPoolClient struct {
	client.Client
}

// NewForConfig ...
func NewForConfig(c *rest.Config) (*IPPoolClient, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = ipamv1alpha1.AddToScheme(scheme)

	kubeclient, err := client.New(c, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &IPPoolClient{Client: kubeclient}, nil
}

func (c *IPPoolClient) GetIPPool(namespace, name string) (*ipamv1alpha1.IPPool, error) {
	pool := &ipamv1alpha1.IPPool{}
	if err := c.Get(
		context.Background(),
		types.NamespacedName{Name: name, Namespace: namespace},
		pool,
	); err != nil {
		return nil, err
	}
	return pool, nil
}
