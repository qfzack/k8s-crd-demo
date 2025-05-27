package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	redisMasterPod     = "redis"
	redisMasterService = "redis"

	redisSentinel     = "%s-sentinel"
	redisSentinelPort = int32(26379)

	sentinelInitScript = "sentinel-init.sh"
)

func (r *RedisReconciler) reconcileSentinel(ctx context.Context, redis *databasesv1.Redis) error {
	envs := []corev1.EnvVar{
		{
			Name:  "REDIS_MASTER_HOST",
			Value: fmt.Sprintf("%s-0.%s", redisMasterPod, redisMasterService),
		},
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	if err := r.createSentinelConfigMap(ctx, redis); err != nil {
		r.Logger.Error(err, "Failed to create or update configmap")
		return err
	}

	// master statefulset configuration
	masterSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
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
					"app":  redisMasterPod,
					"role": "master",
				},
			},
			ServiceName: redisMasterService,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  redisMasterPod,
						"role": "master",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "redis-init",
							Image:   fmt.Sprintf(RedisImage, redis.Spec.Version),
							Command: []string{"sh", "-c"},
							Args:    []string{"chown -R 1001:1001 /data"},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: ptr.To(int64(0)),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      redis.Name,
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    redis.Name,
							Image:   fmt.Sprintf(RedisImage, redis.Spec.Version),
							Command: []string{"redis-server"},
							Args: []string{
								"--dir", "/data",
								"--protected-mode", "no",
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: RedisPort},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   convertResourceList(redis.Spec.Resource.Limits),
								Requests: convertResourceList(redis.Spec.Resource.Requests),
							},
							Env: envs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      redis.Name,
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
		},
	}

	// sentinel deployment configuration
	sentinelDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(redisSentinel, redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "sentinel",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(3)),
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
					InitContainers: []corev1.Container{
						{
							Name:    "redis-sentinel-init",
							Image:   fmt.Sprintf(RedisImage, redis.Spec.Version),
							Command: []string{"sh", "-c"},
							Args: []string{
								`
chown -R 1001:1001 /data
printf "port 26379\n\
sentinel resolve-hostnames yes\n\
sentinel announce-ip $POD_IP\n\
sentinel announce-port 26379\n\
sentinel monitor mymaster $REDIS_MASTER_HOST 6379 2\n\
sentinel down-after-milliseconds mymaster 5000\n\
sentinel failover-timeout mymaster 60000\n\
sentinel parallel-syncs mymaster 1\n" > /data/sentinel.conf
`,
							},
							Env: envs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf(redisSentinel, redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    fmt.Sprintf(redisSentinel, redis.Name),
							Image:   fmt.Sprintf(RedisImage, redis.Spec.Version),
							Command: []string{"redis-server"},
							Args:    []string{"/data/sentinel.conf", "--sentinel"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: redisSentinelPort},
							},
							Env: envs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf(redisSentinel, redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: fmt.Sprintf(redisSentinel, redis.Name),
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// TODO apply storage configuration if specified
	if redis.Spec.Storage.Storage != "" {
		masterSts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
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
	} else {
		masterSts.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: redis.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
	}

	// create cronjob for redis cluster configuring
	if err := r.createSentinelManagerJob(ctx, redis); err != nil {
		r.Logger.Error(err, "Failed to create or update services")
		return err
	}
	// create or update StatefulSets
	if err := r.createOrUpdate(ctx, masterSts); err != nil {
		r.Logger.Error(err, "Failed to create or update statefulset")
		return err
	}
	if err := r.createOrUpdate(ctx, sentinelDeployment); err != nil {
		r.Logger.Error(err, "Failed to create or update deployment")
		return err
	}
	// create services
	if err := r.createSentinelSvcs(ctx, redis); err != nil {
		r.Logger.Error(err, "Failed to create or update services")
		return err
	}

	return nil
}

func (r *RedisReconciler) createSentinelSvcs(ctx context.Context, redis *databasesv1.Redis) error {
	// Master Service
	masterSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redisMasterService,
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "master",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // Headless Service
			Selector: map[string]string{
				"app":  redisMasterPod,
				"role": "master",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "redis-port",
					Port: RedisPort,
				},
			},
		},
	}

	// Sentinel Service
	sentinelSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(redisSentinel, redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "sentinel",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  redis.Name,
				"role": "sentinel",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "redis-sentinel-port",
					Port: redisSentinelPort,
				},
			},
		},
	}

	if err := r.createOrUpdate(ctx, masterSvc); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, sentinelSvc); err != nil {
		return err
	}

	return nil
}

func (r *RedisReconciler) createSentinelConfigMap(ctx context.Context, redis *databasesv1.Redis) error {
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	scriptPath := filepath.Join(baseDir, "..", "resource", sentinelInitScript)
	script, err := os.ReadFile(scriptPath) //nolint:gosec
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-script", redis.Name),
			Namespace: redis.Namespace,
		},
		Data: map[string]string{
			sentinelInitScript: string(script),
		},
	}
	return r.createOrUpdate(ctx, cm)
}

func (r *RedisReconciler) createSentinelManagerJob(ctx context.Context, redis *databasesv1.Redis) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sentinel-manager", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app": redis.Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": redis.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "sentinel-manager",
							Image:   fmt.Sprintf(RedisImage, redis.Spec.Version),
							Command: []string{"/scripts/sentinel-init.sh"},
							Env: []corev1.EnvVar{
								{
									Name:  "REPLICAS",
									Value: strconv.Itoa(int(redis.Spec.Replicas)),
								},
								{
									Name:  "REDIS_PORT",
									Value: strconv.Itoa(int(RedisPort)),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "init-script",
									MountPath: "/scripts",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "init-script",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-init-script", redis.Name),
									},
									DefaultMode: ptr.To(int32(0755)),
								},
							},
						},
					},
				},
			},
		},
	}

	return r.createOrUpdate(ctx, job)
}
