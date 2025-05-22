package controller

import (
	"context"
	"fmt"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RedisPort         = int32(6379)
	RedisSentinelPort = int32(26379)

	clusterInitScript            = "cluster-init.sh"
	clusterInitializedAnnotation = "redis.database.example.com/cluster-initialized"
)

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

func (r *RedisReconciler) createOrUpdate(ctx context.Context, obj client.Object) error {
	// Handle different types of resources
	switch obj := obj.(type) {
	case *appsv1.StatefulSet:
		return r.createOrUpdateStatefulSet(ctx, obj)
	case *appsv1.Deployment:
		return r.createOrUpdateDeployment(ctx, obj)
	case *batchv1.CronJob:
		return r.createOrUpdateCronJob(ctx, obj)
	case *monitoringv1.ServiceMonitor:
		return r.createOrUpdateServiceMonitor(ctx, obj)
	case *corev1.ConfigMap:
		return r.createOrUpdateConfigMap(ctx, obj)
	case *corev1.Service:
		return r.createOrUpdateServices(ctx, obj)
	case *batchv1.Job:
		return r.createOrUpdateJob(ctx, obj)
	default:
		return fmt.Errorf("unsupported create or update resource type: %T", obj)
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

func (r *RedisReconciler) createOrUpdateDeployment(ctx context.Context, deploy *appsv1.Deployment) error {
	existing := &appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      deploy.GetName(),
		Namespace: deploy.GetNamespace(),
	}
	err := r.Get(ctx, key, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new Deployment", "name", deploy.Name)
			return r.Create(ctx, deploy)
		}
		return err
	}

	// Update mutable fields
	existing.Spec.Replicas = deploy.Spec.Replicas
	existing.Spec.Template = deploy.Spec.Template
	existing.Spec.Strategy = deploy.Spec.Strategy

	r.Logger.Info("Updating Deployment", "name", deploy.Name)
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

func (r *RedisReconciler) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing)

	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new ConfigMap", "name", cm.Name)
			return r.Create(ctx, cm)
		}
		return err
	}

	// Update ConfigMap specific fields
	existing.Data = cm.Data
	existing.BinaryData = cm.BinaryData
	existing.Labels = cm.Labels
	existing.Annotations = cm.Annotations

	r.Logger.Info("Updating ConfigMap", "name", cm.Name)
	return r.Update(ctx, existing)
}

func (r *RedisReconciler) createOrUpdateServices(ctx context.Context, svc *corev1.Service) error {
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new Service", "name", svc.Name)
			return r.Create(ctx, svc)
		}
		return err
	}

	// immutable fields
	svc.Spec.ClusterIP = existing.Spec.ClusterIP
	svc.ObjectMeta.ResourceVersion = existing.ObjectMeta.ResourceVersion

	// update service fields
	existing.Spec.Ports = svc.Spec.Ports
	existing.Spec.Selector = svc.Spec.Selector
	existing.Labels = svc.Labels
	existing.Annotations = svc.Annotations

	r.Logger.Info("Updating Service", "name", svc.Name)
	return r.Update(ctx, existing)
}

func (r *RedisReconciler) createOrUpdateJob(ctx context.Context, job *batchv1.Job) error {
	existing := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("Creating new Job", "name", job.Name)
			return r.Create(ctx, job)
		}
		return err
	}

	if existing.Status.Succeeded > 0 {
		r.Logger.Info("Job already completed successfully", "name", job.Name)
		return nil
	}

	if existing.Status.Failed > 0 {
		r.Logger.Info("Deleting failed Job", "name", job.Name)
		if err := r.Delete(ctx, existing); err != nil {
			return fmt.Errorf("failed to delete failed job: %w", err)
		}

		if err := r.waitForDeletion(ctx, existing); err != nil {
			return fmt.Errorf("failed to wait for job deletion: %w", err)
		}

		r.Logger.Info("Creating new Job after failure", "name", job.Name)
		return r.Create(ctx, job)
	}

	r.Logger.Info("Job is still running", "name", job.Name)
	return nil
}

func (r *RedisReconciler) waitForDeletion(ctx context.Context, obj client.Object) error {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	for range 30 {
		err := r.Get(ctx, key, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for resource deletion")
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
	// TODO sentinel mode redis name is redis-master, redis-replica and redis-sentinel
	redisName := redis.Name
	if redis.Spec.Mode == "sentinel" {
		redisName = fmt.Sprintf("%s-master", redis.Name)
	}
	if err := r.Get(ctx, types.NamespacedName{Name: redisName, Namespace: redis.Namespace}, sts); err != nil {
		return ctrl.Result{}, err
	}

	redis.Status.Phase = "Running"
	redis.Status.ReadyReplicas = *sts.Spec.Replicas

	if err := r.Status().Update(ctx, redis); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *RedisReconciler) createServices(ctx context.Context, redis *databasesv1.Redis) error {
	// create headless service
	headlessSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name, // TODO provide custom config
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app": redis.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // Headless Service
			Ports: []corev1.ServicePort{
				{
					Name:     "redis",
					Port:     RedisPort,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "cluster",
					Port:     RedisPort + 10000, // cluster bus port
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": redis.Name,
			},
		},
	}

	// creat client service provide redis service
	clientSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-client", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app": redis.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:     "redis",
					Port:     RedisPort,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": redis.Name,
			},
		},
	}

	if err := r.createOrUpdate(ctx, headlessSvc); err != nil {
		return fmt.Errorf("failed to create/update headless service: %w", err)
	}

	if err := r.createOrUpdate(ctx, clientSvc); err != nil {
		return fmt.Errorf("failed to create/update client service: %w", err)
	}

	return nil
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
