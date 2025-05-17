package controller

import (
	"context"
	_ "embed"
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
	"k8s.io/utils/pointer"
)

const (
	clusterInitScript = "cluster-init.sh"

	clusterInitializedAnnotation = "redis.database.example.com/cluster-initialized"
)

func (r *RedisReconciler) reconcileCluster(ctx context.Context, redis *databasesv1.Redis) error {
	// TODO move to custom config
	const clusterReplicas = 1

	envs := []corev1.EnvVar{}
	for k, v := range redis.Spec.Config {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	if err := r.createConfigMap(ctx, redis); err != nil {
		r.Logger.Error(err, "Failed to create or update configmap")
		return err
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
			ServiceName: redis.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": redis.Name,
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
									Name:      redis.Name,
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
								"--cluster-enabled", "yes",
								"--dir", "/data",
								"--protected-mode", "no",
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: RedisPort},
								{ContainerPort: RedisPort + 10000}, // cluster bus port
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

	// create cronjob for redis cluster configuring
	if redis.Annotations[clusterInitializedAnnotation] != "true" {
		if err := r.createClusterManagerJob(ctx, redis, clusterReplicas); err != nil {
			r.Logger.Error(err, "Failed to create or update services")
			return err
		}
		if redis.Annotations == nil {
			redis.Annotations = make(map[string]string)
		}
		redis.Annotations[clusterInitializedAnnotation] = "true"
		if err := r.Update(ctx, redis); err != nil {
			return err
		}
	}

	// Create or update StatefulSet
	if err := r.createOrUpdate(ctx, sts); err != nil {
		r.Logger.Error(err, "Failed to create or update statefulset")
		return err
	}
	if err := r.createServices(ctx, redis); err != nil {
		r.Logger.Error(err, "Failed to create or update services")
		return err
	}
	// TODO only run once

	return nil
}

func (r *RedisReconciler) createConfigMap(ctx context.Context, redis *databasesv1.Redis) error {
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	scriptPath := filepath.Join(baseDir, "..", "resource", clusterInitScript)
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-script", redis.Name),
			Namespace: redis.Namespace,
		},
		Data: map[string]string{
			clusterInitScript: string(script),
		},
	}
	return r.createOrUpdate(ctx, cm)
}

func (r *RedisReconciler) createClusterManagerJob(ctx context.Context, redis *databasesv1.Redis, replicas int) error {
	job := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cluster-manager", redis.Name),
			Namespace: redis.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "cluster-manager",
									Image:   redis.Spec.Image,
									Command: []string{"/scripts/cluster-init.sh"},
									Env: []corev1.EnvVar{
										{
											Name:  "TOTAL_REPLICAS",
											Value: strconv.Itoa(int(redis.Spec.Replicas)),
										},
										{
											Name:  "REDIS_PORT",
											Value: strconv.Itoa(int(RedisPort)),
										},
										{
											Name:  "CLUSTER_REPLICAS",
											Value: strconv.Itoa(replicas),
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
											DefaultMode: pointer.Int32(0755),
										},
									},
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
