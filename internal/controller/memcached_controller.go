/*
Copyright 2026.

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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/Umesh-Mallipudi/memcached-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	// corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Memcached object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("Reconciling Memcached", "name", memcached.Name)
	depName := memcached.Name + "-memcached-deploy"
	existingDeployment := appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: memcached.Namespace,
		Name:      depName,
	}, &existingDeployment)

	labels := map[string]string{
		"app":      "memcached",
		"instance": memcached.Name,
	}
	image := memcached.Spec.Image
	if image == "" {
		image = "memcached:1.6.39"
	}
	if apierrors.IsNotFound(err) {

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      depName,
				Namespace: memcached.Namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &memcached.Spec.Replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "memcached",
								Image: image,
							},
						},
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(memcached, deployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else if *existingDeployment.Spec.Replicas != memcached.Spec.Replicas || existingDeployment.Spec.Template.Spec.Containers[0].Image != image {
		existingDeployment.Spec.Replicas = &memcached.Spec.Replicas
		existingDeployment.Spec.Template.Spec.Containers[0].Image = image
		err = r.Update(ctx, &existingDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	existingService := &v1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: memcached.Namespace,
		Name:      memcached.Name + "-memcached-service",
	}, existingService)

	if apierrors.IsNotFound(err) {

		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      memcached.Name + "-memcached-service",
				Namespace: memcached.Namespace,
			},
			Spec: v1.ServiceSpec{
				Selector: labels,
				Ports: []v1.ServicePort{
					{
						Port:       11211,
						TargetPort: intstr.FromInt(11211),
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(memcached, service, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		r.Create(ctx, service)

	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Named("memcached").
		Complete(r)
}
