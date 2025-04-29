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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecuritySpec defines the security configuration
type SecuritySpec struct {
	// TLS configuration
	EnableTLS bool `json:"enableTLS,omitempty"`
	// ServiceAccount name
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// BackupSpec defines the Redis data backup configuration
type BackupSpec struct {
	// Enable automatic backup
	Enabled bool `json:"enabled,omitempty"`
	// Automatic backup schedule
	Schedule string `json:"schedule,omitempty"`
	// Retention policy for retain backup data
	RetentionPolicy string `json:"retentionPolicy,omitempty"`
}

// MonitorSpec defines the monitoring configuration
type MonitorSpec struct {
	// Enable Promutheus monitoring
	Enabled bool `json:"enabled,omitempty"`
	// Prometheus service port
	Port int32 `json:"port,omitempty"`
}

// ResourceSpec describes compute resource requirements
type ResourceSpec struct {
	// Limits describes the maximum compute resources allowed
	Limits ResourceList `json:"limits,omitempty"`
	// Requests describes the minimum compute resources required
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList is a set of resource pairs.
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// StorageSpec defines the storage configuration
type StorageSpec struct {
	// Specify the access mode for the persistent volume
	// +kubebuilder:validation:Enum=ReadWriteOnce;ReadOnlyMany;ReadWriteMany
	AccessMode string `json:"accessMode,omitempty"`
	// Storage defines the size of persistance volumn
	Storage string `json:"storage,omitempty"`
	// StorageClassName of persistance volumn
	StorageClassName string `json:"storageClassName,omitempty"`
}

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	// Name is the part of pod name
	// +required
	Name string `json:"name,omitempty"`
	// Image define the used docker image
	// +required
	Image string `json:"image,omitempty"`
	// Replicas define the number of replicas
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas,omitempty"`

	// Mode represents the mode of Redis (standalone, sentinel, cluster)
	// +kubebuilder:validation:Enum=standalone;sentinel;cluster
	Mode string `json:"mode,omitempty"`

	// Resource defines compute resource requirement
	Resource ResourceSpec `json:"resource,omitempty"`

	// Storage confiuration for Redis pods
	Storage StorageSpec `json:"storage,omitempty"`

	// Redis configuration options
	Config map[string]string `json:"config,omitempty"`

	// Monitoring configuration
	Monitor MonitorSpec `json:"monitor,omitempty"`

	// Backup configuration
	Backup BackupSpec `json:"backup,omitempty"`

	Security SecuritySpec `json:"security,omitempty"`

	// TODO: Add other spec fields
	// User []RedisUser `json:"user,omitempty"`
}

// RedisStatus defines the observed state of Redis
type RedisStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster

	// Phase represent the current phase of Redis cluster
	// +kubebuilder:validation:Enum=Pending;Running;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of Redis state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ServiceName is the name of the service created for Redis cluster
	ServiceName string `json:"serviceName,omitempty"`

	// CurrentMaster is the current master node in the case of sentinel/replica mode
	CurrentMaster string `json:"currentMaster,omitempty"`

	// ReadyReplicas is the number of ready Redis pods
	ReadyReplicas int32 `json:"readyReplicas"`

	// LastBackupTime is the last time backup was taken
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Redis is the Schema for the redis API
// +kubebuilder:webhook:verbs=create;update,path=/validate-databases-qfzack-com-v1-redis,mutating=false,failurePolicy=fail,groups=databases.qfzack.com,resources=redises,versions=v1,name=vredis.kb.io,admissionReviewVersions=v1
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec   `json:"spec,omitempty"`
	Status RedisStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}
