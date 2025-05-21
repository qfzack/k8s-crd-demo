package controller

import (
	"context"
	"fmt"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	RedisConfKey    = "redis.conf"
	SentinelConfKey = "sentinel.conf"
)

func (r *RedisReconciler) reconcileSentinel(ctx context.Context, redis *databasesv1.Redis) error {
	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// master statefulset configuration
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
			Replicas: pointer.Int32(1),
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
							Name:    redis.Spec.Name,
							Image:   redis.Spec.Image,
							Command: []string{"redis-server", "/etc/redis/redis.conf"},
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
									Name:      "master-config",
									MountPath: "/etc/redis",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "master-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-master-config", redis.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// replica statefulset configuration
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
							Name:    redis.Name,
							Image:   redis.Spec.Image,
							Command: []string{"redis-server", "/etc/redis/redis.conf"},
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
									Name:      "replica-config",
									MountPath: "/etc/redis",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "replica-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-replica-config", redis.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	//  sentinel statefulset configuration
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
							Name:    "sentinel",
							Image:   redis.Spec.Image,
							Command: []string{"redis-server", "/etc/redis/sentinel.conf"},
							Args:    []string{"--sentinel"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: RedisSentinelPort},
							},
							Env: envs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "sentinel-config",
									MountPath: "/etc/redis",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "sentinel-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-sentinel-config", redis.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// TODO apply storage configuration if specified

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

func (r *RedisReconciler) createConfigMaps(ctx context.Context, redis *databasesv1.Redis) error {
	masterConfig := map[string]string{
		RedisConfKey: `
port 6379
dir /data
appendonly yes
protected-mode no
`}
	replicaConfig := map[string]string{
		RedisConfKey: `
port 6379
dir /data
appendonly yes
protected-mode no
replicaof ${REDIS_MASTER_HOST} 6379
`}
	sentinelConfig := map[string]string{
		SentinelConfKey: `
port 26379
sentinel announce-ip ${POD_IP}
sentinel announce-port 26379
sentinel monitor mymaster ${REDIS_MASTER_HOST} 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 60000
sentinel parallel-syncs mymaster 1
`}

	masterCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-config", redis.Name),
			Namespace: redis.Namespace,
		},
		Data: masterConfig,
	}

	replicaCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-replica-config", redis.Name),
			Namespace: redis.Namespace,
		},
		Data: replicaConfig,
	}

	sentinelCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sentinel-config", redis.Name),
			Namespace: redis.Namespace,
		},
		Data: sentinelConfig,
	}

	// 创建或更新 ConfigMap
	if err := r.createOrUpdate(ctx, masterCm); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, replicaCm); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, sentinelCm); err != nil {
		return err
	}

	return nil
}

// TODO
func (r *RedisReconciler) createServices(ctx context.Context, redis *databasesv1.Redis) error {
	// Master Service
	masterSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "master",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  redis.Name,
				"role": "master",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "redis",
					Port: RedisPort,
				},
			},
		},
	}

	// Sentinel Service
	sentinelSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sentinel", redis.Name),
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
					Name: "sentinel",
					Port: RedisSentinelPort,
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
