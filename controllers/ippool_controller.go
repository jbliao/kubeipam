/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"log"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ipamv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/crd/driver"
)

// IPPoolReconciler reconciles a IPPool object
type IPPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ipam.k8s.cc.cs.nctu.edu.tw,resources=ippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.k8s.cc.cs.nctu.edu.tw,resources=ippools/status,verbs=get;update;patch

// Reconcile ...
func (r *IPPoolReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("ippool", req.NamespacedName)
	gologger := log.New(log.Writer(), "", log.Flags()|log.Lshortfile)
	// your logic here

	var driverObj driver.Driver
	pool := &ipamv1alpha1.IPPool{}
	err := r.Get(ctx, req.NamespacedName, pool)
	if err != nil {
		goto REQUEUE_N_ERROR
	}

	switch tp := pool.Spec.Type; tp {
	case "netbox":
		driverObj, err = driver.NewNetboxDriver(pool.Spec.RawConfig, gologger)
	default:
		driverObj, err = nil, fmt.Errorf("Type %s not implemented", tp)
	}
	if err != nil {
		goto REQUEUE_N_ERROR
	}

	if err := driver.Sync(driverObj, &pool.Spec, gologger); err != nil {
		logger.Error(err, "")
		goto REQUEUE_N_ERROR
	}

	return ctrl.Result{}, nil

REQUEUE_N_ERROR:
	return ctrl.Result{Requeue: true}, err
}

// SetupWithManager ...
func (r *IPPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipamv1alpha1.IPPool{}).
		Complete(r)
}
