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
	r.initLogger()
	redisConfig := &databasesv1.Redis{}

	// Try to get existed Redis CRD instance from k8s cluster
	if err := r.Get(ctx, req.NamespacedName, redisConfig); err != nil {
		r.Logger.Errorf("Fail to get redis instance with keywords %s:  %v", req.NamespacedName, err)
		return ctrl.Result{}, err
	}

	// Clear all Redis pods if CRD instance was deleted
	if !redisConfig.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.clearRedisPod(ctx, redisConfig)
	}

	// Generate pod name by `name` and `replicas` in CRD instance configuration
	r.Logger.Info("Spec configuration of redis object: ", redisConfig.Spec)
	podNames := helper.GetRedisPodNames(redisConfig)

	isEdit := false
	for _, podName := range podNames {
		name, err := helper.CreateRedisPod(r.Client, redisConfig, podName, r.Scheme)
		if err != nil {
			r.Logger.Errorf("Fail to create pod %s: %v", podName, err)
			return ctrl.Result{}, err
		}
		// Pod exist or fail to create pod
		if name == "" || controllerutil.ContainsFinalizer(redisConfig, name) {
			r.Logger.Info("Redis pod existed: ", podName)
			continue
		}
		r.Logger.Info("Created redis pod: ", podName)
		redisConfig.Finalizers = append(redisConfig.Finalizers, podName)
		isEdit = true
	}

	// Delete redis pod when reduce the num of replicas
	if len(redisConfig.Finalizers) > len(podNames) {
		isEdit = true
		r.EventRecord.Event(redisConfig, corev1.EventTypeNormal, "Scaled", "Reduce redis pod")
		r.Logger.Infof("Reduce redis pod num from %d to %d", len(redisConfig.Finalizers), len(podNames))
		err := r.deleteRedisPod(ctx, podNames, redisConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status configutation when CRD instance adjustment completed
	if isEdit {
		r.EventRecord.Event(redisConfig, corev1.EventTypeNormal, "Updated", "Update redis pod")
		r.Logger.Info("Update CRD instance status configuration")
		err := r.Client.Update(ctx, redisConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Status().Update(ctx, redisConfig)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RedisReconciler) initLogger() {
	r.Logger = logrus.New()
	r.Logger.SetLevel(logrus.DebugLevel)
	r.Logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
}

func (r *RedisReconciler) deleteRedisPod(ctx context.Context, podNames []string, redisConfig *databasesv1.Redis) error {
	finalizers := redisConfig.Finalizers
	for _, finalizer := range finalizers {
		isDelete := true
		for _, pod := range podNames {
			if finalizer == pod {
				isDelete = false
				break
			}
		}
		if isDelete {
			err := r.Client.Delete(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      finalizer,
					Namespace: redisConfig.Namespace,
				},
			})
			if err != nil {
				return err
			}
		}
	}

	redisConfig.Finalizers = podNames
	return nil
}

func (r *RedisReconciler) clearRedisPod(ctx context.Context, redisConfig *databasesv1.Redis) error {
	podList := redisConfig.Finalizers
	for _, podName := range podList {
		r.Logger.Info("Delete redis pod: ", podName)
		err := r.Client.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: redisConfig.Namespace,
			},
		})

		if err != nil {
			r.Logger.Error("Fail to delete redis pod: ", podName)
		}
	}

	redisConfig.Finalizers = []string{}
	return r.Client.Update(ctx, redisConfig)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecord = mgr.GetEventRecorderFor("RedisController")

	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1.Redis{}).
		Watches(&corev1.Pod{}, handler.Funcs{DeleteFunc: r.podDeleteHandler}).
		Complete(r)
}

func (r *RedisReconciler) podDeleteHandler(ctx context.Context, event event.TypedDeleteEvent[client.Object], limitInterface workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	r.Logger.Info("Deleted redis pod: ", event.Object.GetName())

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
