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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *RedisReconciler) reconcileStandalone(ctx context.Context, redis *databasesv1.Redis) error {
	// Force replicas to 1 in standalone mode as Redis standalone doesn't support replication
	standaloneReplicas := int32(1)

	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Log if user specified replicas > 1 to inform them about the override
	if redis.Spec.Replicas > 1 {
		r.Logger.Info("Forcing replicas to 1 for standalone mode",
			"requested", redis.Spec.Replicas,
			"actual", standaloneReplicas,
		)
	}

	// Create statefulset configuration
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &standaloneReplicas,
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
							Name:  redis.Name,
							Image: fmt.Sprintf(RedisImage, redis.Spec.Version),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: RedisPort,
								},
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
	} else {
		sts.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: redis.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
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
