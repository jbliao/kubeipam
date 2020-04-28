package clientset

import (
	"context"
	"fmt"
	"log"

	ipamv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IPPoolClient struct {
	client.Client
	logger *log.Logger
}

// NewForConfig ...
func NewForConfig(c *rest.Config, logger *log.Logger) (*IPPoolClient, error) {
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewForConfig")
	}
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = ipamv1alpha1.AddToScheme(scheme)

	kubeclient, err := client.New(c, client.Options{Scheme: scheme})
	if err != nil {
		logger.Println(err)
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
		c.logger.Println(err)
		return nil, err
	}
	return pool, nil
}
