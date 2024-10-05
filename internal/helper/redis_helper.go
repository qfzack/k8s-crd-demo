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

// Get pod names
func GetRedisPodNames(redisConfig *v1.Redis) []string {
	podNames := make([]string, redisConfig.Spec.Replicas)
	fmt.Printf("%+v", redisConfig)

	for i := 0; i < redisConfig.Spec.Replicas; i++ {
		podNames[i] = fmt.Sprintf("%s-%d", redisConfig.Name, i)
	}
	fmt.Println("Podnames: ", podNames)
	return podNames
}

// Judge the pod exist
func IsExistPod(podName string, redisConfig *v1.Redis, client client.Client) bool {
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
	if IsExistPod(podName, redisConfig, client) {
		return "", nil
	}

	newPod := &corev1.Pod{}
	newPod.Name = podName
	newPod.Namespace = redisConfig.Namespace
	newPod.Spec.Containers = []corev1.Container{
		{
			Name:            podName,
			Image:           "redis:5-alpine",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: int32(redisConfig.Spec.Port),
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(redisConfig, newPod, scheme)
	if err != nil {
		return "", err
	}

	err = client.Create(context.Background(), newPod)
	return podName, err
}
