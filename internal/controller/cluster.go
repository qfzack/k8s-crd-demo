package controller

import (
	"context"
	"fmt"

	databasesv1 "github.com/qfzack/redis-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
                                `, masterCount-1, RedisPort, clusterReplicas),
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  redis.Spec.Name,
							Image: redis.Spec.Image,
							Args:  []string{"--cluster-enabled", "yes"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: RedisPort},
								{ContainerPort: RedisPort + 10000}, // cluster bus port
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
