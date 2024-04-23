/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	apiv1 "github.com/jrp-enf/license-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups=api.license-operator.enfabrica.net,resources=licenses,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("In pod reconcile")
	thisPod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, thisPod); err != nil {
		if apierrors.IsNotFound(err) {
			// We're being deleted.
			l.Info(fmt.Sprintf("Pod being deleted: %s::%s", req.Namespace, req.Name))
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		l.Error(fmt.Errorf("unable to get pod %s::%s", req.Namespace, req.Name), "unable to get pod")
		return ctrl.Result{}, nil
	}
	bIsBeingDeleted := false
	if thisPod.ObjectMeta.DeletionTimestamp != nil {
		bIsBeingDeleted = true
	}
	//	if thisPod.Status.Phase != "Running" {
	l.Info("Pod status", "phase", thisPod.Status.Phase)
	//	}
	for k, v := range thisPod.Annotations {
		if k == "enfabrica.net/license" {
			l.Info("Found pod with licenses: ", "licenses", v)
			for _, lic := range strings.Split(v, ",") {
				usage := 1
				val := strings.Split(lic, "/")
				if len(val) != 1 && len(val) != 2 {
					l.Error(errors.New("unable to parse license for pod"), "parse license", "namespace", req.Namespace, "name", req.Name, "license", lic)
					continue
				}
				if len(val) == 2 {
					u, err := strconv.Atoi(val[1])
					if err != nil {
						l.Error(err, "unable to parse license for pod", "namespace", req.Namespace, "name", req.Name, "license", lic)
						continue
					}
					usage = u
				}
				myLicense := &apiv1.License{}

				if err := r.Get(ctx, client.ObjectKey{Namespace: "default", Name: val[0]}, myLicense); err != nil {
					l.Error(err, "unable to retrieve a license for pod", "namespace", req.Namespace, "name", req.Name, "license", val[0])
					continue
				}
				thisPodStatusEntry := PodStatusEntry(thisPod.Namespace, thisPod.Name, usage)
				bFound := false
				for _, podStatusEntry := range myLicense.Status.UsedBy {
					if thisPodStatusEntry == podStatusEntry {
						bFound = true
					}
				}
				if thisPod.Status.Phase == "Pending" {
					if bIsBeingDeleted {
						var entries []string
						for _, v := range myLicense.Status.Queuing {
							if thisPodStatusEntry != v {
								entries = append(entries, v)
							}
						}
						myLicense.Status.Queuing = entries
					} else {
						if !slices.Contains(myLicense.Status.Queuing, PodStatusEntry(thisPod.Namespace, thisPod.Name, usage)) {
							myLicense.Status.Queuing = append(myLicense.Status.Queuing, PodStatusEntry(thisPod.Namespace, thisPod.Name, usage))
						}
					}
					r.Status().Update(ctx, myLicense)
				} else if !bFound || bIsBeingDeleted {
					l.Info("Found a pod not in right status", "namespace", req.Namespace, "name", req.Name)
					if bIsBeingDeleted {
						usage = 0 - usage
						myLicense.Status.UsedBy = append(myLicense.Status.UsedBy, PodStatusEntry(thisPod.Namespace, thisPod.Name, usage))
						usage = 0 // leaving usage negative here ends up with restoring too many licenses for some reason
						var entries []string
						for _, v := range myLicense.Status.Queuing {
							if thisPodStatusEntry != v {
								entries = append(entries, v)
							}
						}
						myLicense.Status.Queuing = entries
					} else {
						myLicense.Status.UsedBy = append(myLicense.Status.UsedBy, thisPodStatusEntry)
					}
					myLicense.Status.Available = myLicense.Status.Available - usage
					myLicense.Status.RealAvailable = myLicense.Status.RealAvailable - usage
					r.Status().Update(ctx, myLicense)
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
