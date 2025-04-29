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

	databasesv1 "github.com/qfzack/redis-operator/api/v1"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	EventRecord record.EventRecorder
	Logger      logr.Logger
}

// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/finalizers,verbs=update
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = log.FromContext(ctx)

	// 1.Try to get existed Redis CRD instance from k8s cluster
	redisConfig := &databasesv1.Redis{}
	r.Logger.Info("Fetch CRD instance", "Name", req.NamespacedName.Name)
	if err := r.Get(ctx, req.NamespacedName, redisConfig); err != nil {
		r.Logger.Error(err, "Failed to fetch CRD instance")
		return ctrl.Result{}, err
	}

	// Initialize the status for a new instance
	if redisConfig.Status.Phase == "" {
		redisConfig.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, redisConfig); err != nil {
			return ctrl.Result{}, nil
		}
	}

	// Handle different Redis modes
	switch redisConfig.Spec.Mode {
	case "standalone":
		if err := r.reconcileStandalone(ctx, redisConfig); err != nil {
			return r.updateStatusWithError(ctx, redisConfig, err)
		}
	case "sentinel":
		if err := r.reconcileSentinel(ctx, redisConfig); err != nil {
			return r.updateStatusWithError(ctx, redisConfig, err)
		}
	case "cluster":
		if err := r.reconcileCluster(ctx, redisConfig); err != nil {
			return r.updateStatusWithError(ctx, redisConfig, err)
		}
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Spec.Mode)
	}

	// Handle common configurations
	if err := r.reconcileCommonConfig(ctx, redisConfig); err != nil {
		return r.updateStatusWithError(ctx, redisConfig, err)
	}

	return r.updateStatus(ctx, redisConfig)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecord = mgr.GetEventRecorderFor("RedisController")

	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1.Redis{}).
		// Watch StatefulSet
		Owns(&appsv1.StatefulSet{}).
		// Watch Services
		Owns(&corev1.Service{}).
		// Watch ConfigMaps
		Owns(&corev1.ConfigMap{}).
		// Watch Secrets
		Owns(&corev1.Secret{}).
		// Watch PVCs
		Owns(&corev1.PersistentVolumeClaim{}).
		// Watch CronJobs
		Owns(&batchv1.CronJob{}).
		// Watch ServiceMonitors for monitoring
		Owns(&monitoringv1.ServiceMonitor{}).
		// Add indexes for faster lookups
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
