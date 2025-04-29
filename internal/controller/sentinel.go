package controller

import (
	"context"
	"fmt"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

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
								{ContainerPort: RedisPort},
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
								{ContainerPort: RedisPort},
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
