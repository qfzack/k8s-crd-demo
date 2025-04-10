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
	"slices"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	"github.com/qfzack/redis-operator/internal/helper"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	EventRecord record.EventRecorder
	Logger      *logrus.Logger
}

// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databases.qfzack.com,resources=redis/finalizers,verbs=update

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
	_ = log.FromContext(ctx)
	redisConfig := &databasesv1.Redis{}

	// 1.Try to get existed Redis CRD instance from k8s cluster
	r.Logger.Infof("Try to fetch CRD instance with NamespacedName %+v", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, redisConfig); err != nil {
		r.Logger.Errorf("Failed to fetch CRD instance:  %v", err)
		return ctrl.Result{}, err
	}

	// 2.Clear all resources if CRD instance was deleted
	if !redisConfig.DeletionTimestamp.IsZero() {
		r.Logger.Info("CRD instance have been deleted, clean up all resources")
		err := r.cleanUpResources(ctx, redisConfig)
		return ctrl.Result{}, err
	}

	// 3.Replica num control
	// Get pod name list with format {Spec.Name}-{serial_number} in Spec configuration
	r.Logger.Infof("The CRD instance spec configuration: %+v", redisConfig.Spec)
	podNames := helper.GetRedisPodNames(redisConfig)

	// Scale up
	// Create new pods as Spec configuration
	updated := false
	for _, podName := range podNames {
		name, err := helper.CreateRedisPod(r.Client, r.Scheme, redisConfig, podName)
		if err != nil {
			r.Logger.Errorf("Fail to create pod %s in namespace %s: %v", podName, redisConfig.Namespace, err)
			return ctrl.Result{}, err
		}
		// Pod exist or fail to create pod
		if name == "" || controllerutil.ContainsFinalizer(redisConfig, name) {
			r.Logger.Infof("Redis pod %s existed in namespace %s", podName, redisConfig.Namespace)
			continue
		}
		r.Logger.Infof("Created redis pod %s in namespace %s", podName, redisConfig.Namespace)
		redisConfig.Finalizers = append(redisConfig.Finalizers, helper.FinalizerName(redisConfig, podName))
		updated = true
	}

	// Scale down
	// Delete pods out of Spec configuration
	if len(redisConfig.Finalizers) > len(podNames) {
		updated = true
		r.EventRecord.Event(redisConfig, corev1.EventTypeNormal, "Scaled", "Reduce redis pod")
		r.Logger.Infof("Reduce redis pod num from %d to %d in namespace %s", len(redisConfig.Finalizers), len(podNames), redisConfig.Namespace)
		err := r.deleteRedisPod(ctx, podNames, redisConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status configutation when CRD instance adjustment completed
	if updated {
		r.EventRecord.Event(redisConfig, corev1.EventTypeNormal, "Updated", "Update redis pod")
		r.Logger.Info("Update CRD instance configuration")

		err := r.Client.Update(ctx, redisConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.initLogger()
	r.EventRecord = mgr.GetEventRecorderFor("RedisController")

	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1.Redis{}).
		Watches(&corev1.Pod{}, handler.Funcs{DeleteFunc: r.podDeleteHandler}).
		Complete(r)
}

func (r *RedisReconciler) initLogger() {
	r.Logger = logrus.New()
	r.Logger.SetLevel(logrus.DebugLevel)
	r.Logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
}

func (r *RedisReconciler) podDeleteHandler(ctx context.Context, event event.TypedDeleteEvent[client.Object], limitInterface workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	r.Logger.Infof("Deleted redis pod %s", event.Object.GetName())

	for _, ref := range event.Object.GetOwnerReferences() {
		if ref.Kind == "Redis" && ref.APIVersion == "databases.qfzack.com/v1" {
			limitInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ref.Name,
					Namespace: event.Object.GetNamespace(),
				},
			})
		}
	}
}

func (r *RedisReconciler) deleteRedisPod(ctx context.Context, podNames []string, redisConfig *databasesv1.Redis) error {
	newFinalizers := []string{}
	finalizers := redisConfig.Finalizers
	for _, finalizer := range finalizers {
		finalizerPod := helper.ParseFinalizer(finalizer)
		if slices.Contains(podNames, finalizerPod) {
			newFinalizers = append(newFinalizers, finalizer)
			continue
		}

		// Delete pods outside the Spec scope
		err := r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      finalizerPod,
				Namespace: redisConfig.Namespace,
			},
		})
		if err != nil {
			return err
		}
	}

	redisConfig.Finalizers = newFinalizers
	return nil
}

func (r *RedisReconciler) cleanUpResources(ctx context.Context, redisConfig *databasesv1.Redis) error {
	finalizers := redisConfig.Finalizers
	for _, finalizer := range finalizers {
		finalizerPod := helper.ParseFinalizer(finalizer)

		r.Logger.Infof("Delete redis pod %s in namespace %s", finalizerPod, redisConfig.Namespace)
		err := r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      finalizerPod,
				Namespace: redisConfig.Namespace,
			},
		})
		if err != nil {
			r.Logger.Error("Fail to delete redis pod: ", finalizerPod)
		}
	}

	redisConfig.Finalizers = []string{}
	return r.Client.Update(ctx, redisConfig)
}
