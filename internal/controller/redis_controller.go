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
	"time"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
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
	// 	if err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// }
	// return ctrl.Result{}, nil
}

func (r *RedisReconciler) reconcileStandalone(ctx context.Context, redis *databasesv1.Redis) error {
	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": redis.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": redis.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  redis.Spec.Name,
							Image: redis.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: redis.Spec.Port,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   convertResourceList(redis.Spec.Resource.Limits),
								Requests: convertResourceList(redis.Spec.Resource.Requests),
							},
							Env: envs,
						},
					},
				},
			},
		},
	}

	// Apply storage configuration if specified
	if redis.Spec.Storage.Storage != "" {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: redis.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.PersistentVolumeAccessMode(redis.Spec.Storage.AccessMode),
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(redis.Spec.Storage.Storage),
						},
					},
					StorageClassName: &redis.Spec.Storage.StorageClassName,
				},
			},
		}
	}

	if err := r.createOrUpdate(ctx, sts); err != nil {
		r.Logger.Error(err, "Failed to create or update StatefulSet")
		return err
	}
	return nil
}

func (r *RedisReconciler) reconcileSentinel(ctx context.Context, redis *databasesv1.Redis) error {
	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	masterSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "master",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  redis.Name,
					"role": "master",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  redis.Name,
						"role": "master",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  redis.Spec.Name,
							Image: redis.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: redis.Spec.Port},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   convertResourceList(redis.Spec.Resource.Limits),
								Requests: convertResourceList(redis.Spec.Resource.Requests),
							},
							Env: envs,
						},
					},
				},
			},
		},
	}

	replicaSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-replica", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "replica",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  redis.Name,
					"role": "replica",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  redis.Name,
						"role": "replica",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  redis.Name,
							Image: redis.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: redis.Spec.Port},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   convertResourceList(redis.Spec.Resource.Limits),
								Requests: convertResourceList(redis.Spec.Resource.Requests),
							},
							Env: []corev1.EnvVar{
								{
									Name: "REDIS_MASTER_HOST",
									// TODO optimize the value string
									Value: fmt.Sprintf("%s-master-0.%s", redis.Name, redis.Namespace),
								},
							},
						},
					},
				},
			},
		},
	}

	sentinelSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sentinel", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "sentinel",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: pointer.Int32(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  redis.Name,
					"role": "sentinel",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  redis.Name,
						"role": "sentinel",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sentinel",
							Image: redis.Spec.Image,
							Args:  []string{"--sentinel"},
							// TODO remove hardcode
							Ports: []corev1.ContainerPort{
								{ContainerPort: 26379},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "REDIS_MASTER_HOST",
									Value: fmt.Sprintf("%s-master-0.%s", redis.Name, redis.Namespace),
								},
							},
						},
					},
				},
			},
		},
	}

	// apply storage configuration if specified
	if redis.Spec.Storage.Storage != "" {
		storageSpec := corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(redis.Spec.Storage.AccessMode),
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(redis.Spec.Storage.Storage),
				},
			},
			StorageClassName: &redis.Spec.Storage.StorageClassName,
		}

		masterSts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{Name: redis.Spec.Name}, Spec: storageSpec},
		}
		replicaSts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{Name: redis.Spec.Name}, Spec: storageSpec},
		}
	}

	// create or update StatefulSets
	if err := r.createOrUpdate(ctx, masterSts); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, replicaSts); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, sentinelSts); err != nil {
		return err
	}

	return nil
}

func (r *RedisReconciler) reconcileCluster(ctx context.Context, redis *databasesv1.Redis) error {
	const clusterReplicas = 1
	masterCount := redis.Spec.Replicas / 2
	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app": redis.Name,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": redis.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": redis.Name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "cluster-init",
							Image: redis.Spec.Image,
							Command: []string{
								"bash",
								"-c",
								fmt.Sprintf(`
                                    if [ ${HOSTNAME##*-} -eq 0 ]; then
                                        redis-cli --cluster create \
                                            $(for i in $(seq 0 %d); do echo "${HOSTNAME%%-*}-$i.${HOSTNAME%%-*}:%d"; done) \
                                            --cluster-replicas %d
                                    fi
                                `, masterCount-1, redis.Spec.Port, clusterReplicas),
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  redis.Spec.Name,
							Image: redis.Spec.Image,
							Args:  []string{"--cluster-enabled", "yes"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: redis.Spec.Port},
								{ContainerPort: redis.Spec.Port + 10000}, // cluster bus port
							},
							Resources: corev1.ResourceRequirements{
								Limits:   convertResourceList(redis.Spec.Resource.Limits),
								Requests: convertResourceList(redis.Spec.Resource.Requests),
							},
							Env: envs,
						},
					},
				},
			},
		},
	}

	// apply storage configuration if specified
	if redis.Spec.Storage.Storage != "" {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: redis.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.PersistentVolumeAccessMode(redis.Spec.Storage.AccessMode),
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(redis.Spec.Storage.Storage),
						},
					},
					StorageClassName: &redis.Spec.Storage.StorageClassName,
				},
			},
		}
	}

	// Create or update StatefulSet
	if err := r.createOrUpdate(ctx, sts); err != nil {
		return err
	}

	return nil
}

func (r *RedisReconciler) reconcileCommonConfig(ctx context.Context, redis *databasesv1.Redis) error {
	if redis.Spec.Security.EnableTLS {
		// TODO: Implement TLS configuration
		// Create secrets for TLS certificates
		// Mount certificates to pods
	}

	// Handle monitoring configuration
	if redis.Spec.Monitor.Enabled {
		// Create ServiceMonitor for Prometheus
		serviceMonitor := &monitoringv1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      redis.Name,
				Namespace: redis.Namespace,
			},
			Spec: monitoringv1.ServiceMonitorSpec{
				Endpoints: []monitoringv1.Endpoint{
					{
						Port:     "metrics",
						Path:     "/metrics",
						Interval: "30s",
					},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": redis.Name,
					},
				},
			},
		}
		if err := r.createOrUpdate(ctx, serviceMonitor); err != nil {
			return fmt.Errorf("failed to configure monitoring: %w", err)
		}
	}

	// Handle backup configuration
	if redis.Spec.Backup.Enabled {
		// Create cronjob for backup
		backupJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-backup", redis.Name),
				Namespace: redis.Namespace,
			},
			Spec: batchv1.CronJobSpec{
				Schedule: redis.Spec.Backup.Schedule,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  fmt.Sprintf("%s-backup", redis.Spec.Name),
										Image: redis.Spec.Image,
										Command: []string{
											"redis-cli",
											"SAVE",
										},
									},
								},
								RestartPolicy: corev1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		}
		if err := r.createOrUpdate(ctx, backupJob); err != nil {
			return fmt.Errorf("failed to configure backup cronjob: %w", err)
		}
	}

	return nil
}

func (r *RedisReconciler) updateStatus(ctx context.Context, redis *databasesv1.Redis) (ctrl.Result, error) {
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, sts); err != nil {
		return ctrl.Result{}, err
	}

	redis.Status.Phase = "Running"
	redis.Status.ReadyReplicas = *sts.Spec.Replicas

	if err := r.Status().Update(ctx, redis); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *RedisReconciler) updateStatusWithError(ctx context.Context, redis *databasesv1.Redis, err error) (ctrl.Result, error) {
	redis.Status.Phase = "Failed"
	redis.Status.Conditions = append(redis.Status.Conditions, metav1.Condition{
		Type:               "Error",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "ReconcilizationError",
		Message:            err.Error(),
	})

	if updateErr := r.Status().Update(ctx, redis); updateErr != nil {
		return ctrl.Result{}, fmt.Errorf("original error: %v, status update error: %v", err, updateErr)
	}
	return ctrl.Result{}, nil
}

func convertResourceList(resources databasesv1.ResourceList) corev1.ResourceList {
	if resources.CPU == "" && resources.Memory == "" {
		return nil
	}

	result := corev1.ResourceList{}

	if resources.CPU != "" {
		result[corev1.ResourceCPU] = resource.MustParse(resources.CPU)
	}

	if resources.Memory != "" {
		result[corev1.ResourceMemory] = resource.MustParse(resources.Memory)
	}

	return result
}

func (r *RedisReconciler) createOrUpdate(ctx context.Context, obj client.Object) error {
	// Handle different types of resources
	switch obj := obj.(type) {
	case *appsv1.StatefulSet:
		return r.createOrUpdateStatefulSet(ctx, obj)
	case *batchv1.CronJob:
		return r.createOrUpdateCronJob(ctx, obj)
	case *monitoringv1.ServiceMonitor:
		return r.createOrUpdateServiceMonitor(ctx, obj)
	default:
		return fmt.Errorf("unsupported resource type: %T", obj)
	}
}

func (r *RedisReconciler) createOrUpdateStatefulSet(ctx context.Context, sts *appsv1.StatefulSet) error {
	existing := &appsv1.StatefulSet{}
	key := types.NamespacedName{
		Name:      sts.GetName(),
		Namespace: sts.GetNamespace(),
	}
	err := r.Get(ctx, key, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new StatefulSet", "name", sts.Name)
			return r.Create(ctx, sts)
		}
		return err
	}

	// TODO update only mutable fields
	existing.Spec.Replicas = sts.Spec.Replicas
	existing.Spec.Template = sts.Spec.Template
	existing.Spec.UpdateStrategy = sts.Spec.UpdateStrategy
	existing.Spec.MinReadySeconds = sts.Spec.MinReadySeconds

	r.Logger.Info("Updating StatefulSet", "name", sts.Name)
	return r.Update(ctx, existing)
}

func (r *RedisReconciler) createOrUpdateCronJob(ctx context.Context, cronJob *batchv1.CronJob) error {
	existing := &batchv1.CronJob{}
	key := types.NamespacedName{
		Name:      cronJob.GetName(),
		Namespace: cronJob.GetNamespace(),
	}
	err := r.Get(ctx, key, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new CronJob", "name", cronJob.Name)
			return r.Create(ctx, cronJob)
		}
		return err
	}

	// Update CronJob fields
	existing.Spec.Schedule = cronJob.Spec.Schedule
	existing.Spec.JobTemplate = cronJob.Spec.JobTemplate
	existing.Spec.Suspend = cronJob.Spec.Suspend

	r.Logger.Info("Updating CronJob", "name", cronJob.Name)
	return r.Update(ctx, existing)
}

func (r *RedisReconciler) createOrUpdateServiceMonitor(ctx context.Context, sm *monitoringv1.ServiceMonitor) error {
	existing := &monitoringv1.ServiceMonitor{}
	err := r.Get(ctx, types.NamespacedName{Name: sm.Name, Namespace: sm.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new ServiceMonitor", "name", sm.Name)
			return r.Create(ctx, sm)
		}
		return err
	}

	// Update ServiceMonitor specific fields
	existing.Spec.Endpoints = sm.Spec.Endpoints
	existing.Spec.Selector = sm.Spec.Selector
	existing.Spec.NamespaceSelector = sm.Spec.NamespaceSelector

	r.Logger.Info("Updating ServiceMonitor", "name", sm.Name)
	return r.Update(ctx, existing)
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
