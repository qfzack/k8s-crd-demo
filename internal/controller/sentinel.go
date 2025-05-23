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

const (
	RedisConfKey    = "redis.conf"
	SentinelConfKey = "sentinel.conf"
)

func (r *RedisReconciler) reconcileSentinel(ctx context.Context, redis *databasesv1.Redis) error {
	envs := []corev1.EnvVar{
		{
			Name:  "REDIS_MASTER_HOST",
			Value: "redis-master-0.redis-master",
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
			Replicas: &redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  redis.Name,
					"role": "master",
				},
			},
			ServiceName: fmt.Sprintf("%s-master", redis.Name),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  redis.Name,
						"role": "master",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "redis-init",
							Image:   redis.Spec.Image,
							Command: []string{"sh", "-c"},
							Args:    []string{"chown -R 1001:1001 /data"},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: pointer.Int64(0),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf("%s-master", redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    redis.Spec.Name,
							Image:   redis.Spec.Image,
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
									Name:      fmt.Sprintf("%s-master", redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
		},
	}

	// // replica statefulset configuration
	// replicaSts := &appsv1.StatefulSet{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      fmt.Sprintf("%s-replica", redis.Name),
	// 		Namespace: redis.Namespace,
	// 		Labels: map[string]string{
	// 			"app":  redis.Name,
	// 			"role": "replica",
	// 		},
	// 	},
	// 	Spec: appsv1.StatefulSetSpec{
	// 		Replicas: &redis.Spec.Replicas,
	// 		Selector: &metav1.LabelSelector{
	// 			MatchLabels: map[string]string{
	// 				"app":  redis.Name,
	// 				"role": "replica",
	// 			},
	// 		},
	// 		ServiceName: fmt.Sprintf("%s-replica", redis.Name),
	// 		Template: corev1.PodTemplateSpec{
	// 			ObjectMeta: metav1.ObjectMeta{
	// 				Labels: map[string]string{
	// 					"app":  redis.Name,
	// 					"role": "replica",
	// 				},
	// 			},
	// 			Spec: corev1.PodSpec{
	// 				InitContainers: []corev1.Container{
	// 					{
	// 						Name:    "redis-init",
	// 						Image:   redis.Spec.Image,
	// 						Command: []string{"sh", "-c"},
	// 						Args:    []string{"chown -R 1001:1001 /data"},
	// 						SecurityContext: &corev1.SecurityContext{
	// 							RunAsUser: pointer.Int64(0),
	// 						},
	// 						VolumeMounts: []corev1.VolumeMount{
	// 							{
	// 								Name:      fmt.Sprintf("%s-replica", redis.Name),
	// 								MountPath: "/data",
	// 							},
	// 						},
	// 					},
	// 				},
	// 				Containers: []corev1.Container{
	// 					{
	// 						Name:    redis.Name,
	// 						Image:   redis.Spec.Image,
	// 						Command: []string{"sh", "-c"},
	// 						Args: []string{
	// 							`
	// 							redis-server --dir /data --protected-mode no &
	// 							echo "Checking local redis-server:"
	// 							until redis-cli ping; do
	// 								echo "Waiting for local redis-server to start..."
	// 								sleep 1
	// 							done
	// 							echo "Checking master $REDIS_MASTER_HOST:"
	// 							until redis-cli -h "$REDIS_MASTER_HOST" ping; do
	// 								echo "Waiting for master $REDIS_MASTER_HOST..."
	// 								sleep 1
	// 							done
	// 							redis-cli replicaof "$REDIS_MASTER_HOST" 6379
	// 							wait
	// 							`,
	// 						},
	// 						Ports: []corev1.ContainerPort{
	// 							{ContainerPort: RedisPort},
	// 						},
	// 						Resources: corev1.ResourceRequirements{
	// 							Limits:   convertResourceList(redis.Spec.Resource.Limits),
	// 							Requests: convertResourceList(redis.Spec.Resource.Requests),
	// 						},
	// 						Env: envs,
	// 						VolumeMounts: []corev1.VolumeMount{
	// 							{
	// 								Name:      fmt.Sprintf("%s-replica", redis.Name),
	// 								MountPath: "/data",
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	// sentinel deployment configuration
	sentinelDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sentinel", redis.Name),
			Namespace: redis.Namespace,
			Labels: map[string]string{
				"app":  redis.Name,
				"role": "sentinel",
			},
		},
		Spec: appsv1.DeploymentSpec{
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
					InitContainers: []corev1.Container{
						{
							Name:    "redis-init",
							Image:   redis.Spec.Image,
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
									Name:      fmt.Sprintf("%s-sentinel", redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "sentinel",
							Image:   redis.Spec.Image,
							Command: []string{"redis-server"},
							Args:    []string{"/data/sentinel.conf", "--sentinel"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: RedisSentinelPort},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: pointer.Int64(0),
							},
							Env: envs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf("%s-sentinel", redis.Name),
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: fmt.Sprintf("%s-sentinel", redis.Name),
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
					Name: fmt.Sprintf("%s-master", redis.Name),
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
		// replicaSts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		// 	{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name: fmt.Sprintf("%s-replica", redis.Name),
		// 		},
		// 		Spec: corev1.PersistentVolumeClaimSpec{
		// 			AccessModes: []corev1.PersistentVolumeAccessMode{
		// 				corev1.PersistentVolumeAccessMode(redis.Spec.Storage.AccessMode),
		// 			},
		// 			Resources: corev1.VolumeResourceRequirements{
		// 				Requests: corev1.ResourceList{
		// 					corev1.ResourceStorage: resource.MustParse(redis.Spec.Storage.Storage),
		// 				},
		// 			},
		// 			StorageClassName: &redis.Spec.Storage.StorageClassName,
		// 		},
		// 	},
		// }
	} else {
		masterSts.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: fmt.Sprintf("%s-master", redis.Name),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		// replicaSts.Spec.Template.Spec.Volumes = []corev1.Volume{
		// 	{
		// 		Name: fmt.Sprintf("%s-replica", redis.Name),
		// 		VolumeSource: corev1.VolumeSource{
		// 			EmptyDir: &corev1.EmptyDirVolumeSource{},
		// 		},
		// 	},
		// }
	}

	// create or update StatefulSets
	if err := r.createOrUpdate(ctx, masterSts); err != nil {
		return err
	}
	// if err := r.createOrUpdate(ctx, replicaSts); err != nil {
	// 	return err
	// }
	if err := r.createOrUpdate(ctx, sentinelDeployment); err != nil {
		return err
	}
	// create services
	if err := r.createSentinelSvcs(ctx, redis); err != nil {
		return err
	}

	return nil
}

func (r *RedisReconciler) createSentinelSvcs(ctx context.Context, redis *databasesv1.Redis) error {
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
			ClusterIP: "None", // Headless Service
			Ports: []corev1.ServicePort{
				{
					Name: "redis",
					Port: RedisPort,
				},
			},
			Selector: map[string]string{
				"app":  redis.Name,
				"role": "master",
			},
		},
	}

	// // replica Service
	// replicaSvc := &corev1.Service{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      fmt.Sprintf("%s-replica", redis.Name),
	// 		Namespace: redis.Namespace,
	// 		Labels: map[string]string{
	// 			"app":  redis.Name,
	// 			"role": "replica",
	// 		},
	// 	},
	// 	Spec: corev1.ServiceSpec{
	// 		ClusterIP: "None", // Headless Service
	// 		Ports: []corev1.ServicePort{
	// 			{
	// 				Name: "redis",
	// 				Port: RedisPort,
	// 			},
	// 		},
	// 		Selector: map[string]string{
	// 			"app":  redis.Name,
	// 			"role": "replica",
	// 		},
	// 	},
	// }

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
	// if err := r.createOrUpdate(ctx, replicaSvc); err != nil {
	// 	return err
	// }
	if err := r.createOrUpdate(ctx, sentinelSvc); err != nil {
		return err
	}

	return nil
}
