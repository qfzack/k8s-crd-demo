package helper

import (
	"context"
	"fmt"

	v1 "github.com/qfzack/redis-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Get Redis pod name list from CRD instance configuration
func GetRedisPodNames(redisConfig *v1.Redis) []string {
	podNames := make([]string, redisConfig.Spec.Replicas)
	for i := 0; i < redisConfig.Spec.Replicas; i++ {
		podNames[i] = fmt.Sprintf("%s-%d", redisConfig.Name, i)
	}

	return podNames
}

// Judge whether the pod exists in k8s cluster
func IsPodExist(client client.Client, podName string, redisConfig *v1.Redis) bool {
	err := client.Get(context.Background(), types.NamespacedName{
		Namespace: redisConfig.Namespace,
		Name:      podName,
	}, &corev1.Pod{})

	return err == nil
}

func IsExistInFinalizers(podName string, redis *v1.Redis) bool {
	for _, f := range redis.Finalizers {
		if f == podName {
			return true
		}
	}
	return false
}

func CreateRedisPod(client client.Client, redisConfig *v1.Redis, podName string, scheme *runtime.Scheme) (string, error) {
	if IsPodExist(client, podName, redisConfig) {
		return "", nil
	}

	newPod := &corev1.Pod{}
	newPod.Name = podName
	newPod.Namespace = redisConfig.Namespace

	envVars := []corev1.EnvVar{}
	if redisConfig.Spec.Password != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "REDIS_PASSWORD",
			Value: redisConfig.Spec.Password,
		})
	}
	newPod.Spec.Containers = []corev1.Container{
		{
			Name:            podName,
			Image:           redisConfig.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: int32(redisConfig.Spec.Port),
				},
			},
			Env: envVars,
		},
	}

	err := controllerutil.SetControllerReference(redisConfig, newPod, scheme)
	if err != nil {
		return "", err
	}

	err = client.Create(context.Background(), newPod)
	return podName, err
}
