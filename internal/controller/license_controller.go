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
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1 "github.com/jrp-enf/license-operator/api/v1"
)

// LicenseReconciler reconciles a License object
type LicenseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.license-operator.enfabrica.net,resources=licenses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.license-operator.enfabrica.net,resources=licenses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.license-operator.enfabrica.net,resources=licenses/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the License object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *LicenseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("In reconcile")
	thisLicense := &apiv1.License{}
	err := r.Get(ctx, req.NamespacedName, thisLicense)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// We're being deleted.
			l.Info(fmt.Sprintf("License being deleted: %s::%s", req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
		l.Error(fmt.Errorf("unable to get license %s::%s", req.Namespace, req.Name), "unable to get license")
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}
	podList := &corev1.PodList{}
	if err = r.List(ctx, podList); err != nil {
		l.Error(err, "failed to list pods")
	}
	var podListStr []string
	queued := thisLicense.Status.Queuing
	used := 0
	for _, pod := range podList.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		if pod.Status.Phase != "Running" {
			continue
		}
		for k, a := range pod.Annotations {
			if k == "enfabrica.net/license" {
				for _, r := range strings.Split(a, ",") {
					count := 1
					val := strings.Split(r, "/")
					if val[0] != thisLicense.Name {
						continue
					}
					if len(val) != 1 && len(val) != 2 {
						l.Error(fmt.Errorf("failed to get license on pod %s::%s, %s", pod.Namespace, pod.Name, val), "license should be / separated from count")
					}
					if len(val) == 2 {
						count, err = strconv.Atoi(val[1])
						if err != nil {
							l.Error(err, fmt.Sprintf("Cannot convert 2nd argument on pod to int %s::%s, %s", pod.Namespace, pod.Name, val))
							count = 1
						}
					}
					used += count
					podListStr = append(podListStr, PodStatusEntry(pod.Namespace, pod.Name, count))

					var entries []string
					for _, v := range queued {
						if PodStatusEntry(pod.Namespace, pod.Name, count) != v {
							entries = append(entries, v)
						}
					}
					queued = entries
				}
			}
		}
	}
	thisLicense.Status.UsedBy = podListStr
	thisLicense.Status.Available = thisLicense.Spec.LicenseCount - used
	thisLicense.Status.RealAvailable = thisLicense.Spec.LicenseCount - used
	if thisLicense.Spec.InUseLicenseCount > used {
		thisLicense.Status.RealAvailable = thisLicense.Spec.LicenseCount - thisLicense.Spec.InUseLicenseCount
	}
	queued = deleteMissing(queued, podList.Items)
	thisLicense.Status.Queuing = queued
	err = r.Status().Update(ctx, thisLicense)
	if err != nil {
		l.Error(err, "failed to update license status")
	}
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func deleteMissing(queued []string, pods []corev1.Pod) []string {
	var tmpQueued []string
	for _, q := range queued {
		v1 := strings.Split(q, "/")
		v := strings.Split(v1[0], "::")
		namespace := v[0]
		podname := v[1]
		bFound := false
		for _, p := range pods {
			if p.Namespace == namespace && p.Name == podname && p.Status.Phase != "Running" {
				bFound = true
			}
		}
		if bFound {
			tmpQueued = append(tmpQueued, q)
		}
	}
	return tmpQueued
}

func PodStatusEntry(namespace, name string, count int) string {
	return namespace + "::" + name + "/" + strconv.Itoa(count)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LicenseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.License{}).
		Complete(r)
}
