When you're writing a Kubernetes operator in Go (typically using [Kubebuilder](https://book.kubebuilder.io/?utm_source=chatgpt.com) or [Operator SDK](https://sdk.operatorframework.io/?utm_source=chatgpt.com)), you generally don't write raw YAML for Deployments or Services. Instead, you create Kubernetes API objects using the Go types from the official Kubernetes API packages.

The main schemas (Go structs) you'll commonly use are:

| Kubernetes Resource   | Go Package                 | Go Type                        |
| --------------------- | -------------------------- | ------------------------------ |
| Deployment            | `k8s.io/api/apps/v1`       | `appsv1.Deployment`            |
| StatefulSet           | `k8s.io/api/apps/v1`       | `appsv1.StatefulSet`           |
| DaemonSet             | `k8s.io/api/apps/v1`       | `appsv1.DaemonSet`             |
| Service               | `k8s.io/api/core/v1`       | `corev1.Service`               |
| ConfigMap             | `k8s.io/api/core/v1`       | `corev1.ConfigMap`             |
| Secret                | `k8s.io/api/core/v1`       | `corev1.Secret`                |
| PersistentVolumeClaim | `k8s.io/api/core/v1`       | `corev1.PersistentVolumeClaim` |
| Pod                   | `k8s.io/api/core/v1`       | `corev1.Pod`                   |
| ServiceAccount        | `k8s.io/api/core/v1`       | `corev1.ServiceAccount`        |
| Job                   | `k8s.io/api/batch/v1`      | `batchv1.Job`                  |
| CronJob               | `k8s.io/api/batch/v1`      | `batchv1.CronJob`              |
| Ingress               | `k8s.io/api/networking/v1` | `networkingv1.Ingress`         |
| NetworkPolicy         | `k8s.io/api/networking/v1` | `networkingv1.NetworkPolicy`   |

## Common imports

```go
import (
    appsv1 "k8s.io/api/apps/v1"
    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    networkingv1 "k8s.io/api/networking/v1"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    "k8s.io/apimachinery/pkg/api/resource"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)
```

## Deployment schema

The top-level object is:

```go
appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{},
    Spec: appsv1.DeploymentSpec{},
}
```

Inside `DeploymentSpec` you'll use:

```go
appsv1.DeploymentSpec{
    Replicas: pointer.Int32(2),
    Selector: &metav1.LabelSelector{},
    Template: corev1.PodTemplateSpec{},
}
```

The pod template contains:

```go
corev1.PodTemplateSpec{
    ObjectMeta: metav1.ObjectMeta{},
    Spec: corev1.PodSpec{},
}
```

The pod spec contains:

```go
corev1.PodSpec{
    Containers: []corev1.Container{},
    Volumes: []corev1.Volume{},
    ServiceAccountName: "...",
}
```

Each container uses:

```go
corev1.Container{
    Name: "...",
    Image: "...",
    Ports: []corev1.ContainerPort{},
    Env: []corev1.EnvVar{},
    Resources: corev1.ResourceRequirements{},
    VolumeMounts: []corev1.VolumeMount{},
    LivenessProbe: &corev1.Probe{},
    ReadinessProbe: &corev1.Probe{},
}
```

## Metadata schema

Almost every Kubernetes resource has:

```go
metav1.ObjectMeta{
    Name: "...",
    Namespace: "...",
    Labels: map[string]string{},
    Annotations: map[string]string{},
}
```

## Owner Reference

Operators typically set the custom resource as the owner:

```go
controllerutil.SetControllerReference(
    myCR,
    deployment,
    r.Scheme,
)
```

This ensures Kubernetes garbage-collects the Deployment when the custom resource is deleted.

## Controller-runtime APIs

In the reconciler you'll commonly use:

```go
r.Get(ctx, key, obj)
r.Create(ctx, obj)
r.Update(ctx, obj)
r.Delete(ctx, obj)
r.Status().Update(ctx, obj)
```

## Your Custom Resource schema

You'll also define your own CRD:

```go
type MyAppSpec struct {
    Replicas *int32 `json:"replicas,omitempty"`
    Image    string `json:"image,omitempty"`
}

type MyAppStatus struct {
    ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}
```

The reconciler reads this spec and creates Kubernetes resources accordingly.

## Typical object hierarchy

```
Custom Resource
      │
      ▼
Deployment
    │
    ├── ObjectMeta
    └── DeploymentSpec
          │
          ├── Replicas
          ├── Selector
          └── PodTemplateSpec
                  │
                  ├── ObjectMeta
                  └── PodSpec
                        │
                        ├── Containers
                        ├── Volumes
                        ├── Affinity
                        ├── Tolerations
                        └── ServiceAccount
```

For most operators, these are the core packages you'll use repeatedly:

* `metav1` — metadata (`ObjectMeta`, `TypeMeta`, `LabelSelector`)
* `appsv1` — `Deployment`, `StatefulSet`, `DaemonSet`
* `corev1` — Pods, Containers, Services, Volumes, ConfigMaps, Secrets, PVCs
* `batchv1` — Jobs and CronJobs
* `networkingv1` — Ingress and NetworkPolicy
* `controller-runtime` — Kubernetes client, reconciliation logic, and controller utilities
* `controllerutil` — owner references and helper functions

These cover the majority of Kubernetes resources an operator typically manages.
