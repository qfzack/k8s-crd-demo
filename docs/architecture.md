# Architecture

## 1.Github Project Reference

Some github repositories established with Kubebuilder.

1. [redis-operator](https://github.com/OT-CONTAINER-KIT/redis-operator)
2. [mongodb-kubernetes-operator](https://github.com/mongodb/mongodb-kubernetes-operator)

## 2.Redis Image

bitnami/redis is a Redis image provided and maintained by [Bitnami Library](https://github.com/bitnami/charts), with improvements in configurability, security, and automated deployment. It is suitable for production environments and Kubernetes deployments.

| Scenario | Bitnami Helm Chart | Redis Operator |
|---------|---------|---------|
| Automatic Failure Recovery | Requires Sentinel | CRD Support |
| Auto Scaling | :x: | CRD Support |
| Sharding | :x: | CRD Support |
| GitOps Automation | :x: | Supported |
| Multi-Instance Management | :x: | Supported |
| Auto Backup/Recovery | :x: | CRD Support |
| State Awareness | :x: | CRD Support |

## 3.Project Structure

```tree
redis-operator/
├── api/                    # API definitions (CRD specs)
│   └── v1/                 # API version
│       ├── redis_types.go  # CRD type definitions
│       └── zz_generated.*  # Auto-generated deepcopy functions
├── cmd/                    # Application entry points
│   └── main.go             # Main program
├── config/                 # Kubernetes manifests
│   ├── crd/                # CRD yaml files
│   ├── rbac/               # RBAC configurations
│   └── samples/            # Example CR yaml files
├── internal/               # Private application code
│   ├── controller/         # Controller implementation
│   └── helper/             # Shared utilities
└── pkg/                    # Public libraries (if any)
```

## 4.Operator Design

### CRD Definition

- Redis version
- replica numbers (master-slave mode/cluster mode)
- resource requests and limits (CPU and memory)
- storage configuration (PVC and emptyDir)
- backup and restore policies
- monitoring and logging

### Cluster Management

- support single-node mode / master-slave mode (Redis sentinel) / fragmented cluster mode (Redis cluster)
- capacity expension and reduction of pods
- master selection mechanism (if use sentinel, need switch from slave to master)

### High Availability And Failure Recovery

- monitor Redis pod status, and pulls up the faulty pod
- sentinel monitoring mechanism
- use statfulSet ensure stable pod network identification

### Data persistence

- support AOF and RDB
- PVC persistent storage
- handle storage expansion issues

### Configuration Management

- support custom Redis configuration by CRD (For example, redis.conf)
- compatible with different Redis version

### Monitoring And Logging

- Integrate Promethues to collect Redis metrics
- Collect logs with Fluentd/Elasticsearch
- provide `kubectl logs` easy viewing of logs

### Security

- configure password authentication
- configure RBAC to controll the Operator permission
- limit Redis access (NetworkPolicy)

### Automation And CI/CD

- operator test (E2E, unit test)
- compitable with Helm and Kustomize for installation

## Enviroment Preparation

- Operator Framework (Kubebuilder or Operator SDK)
- Go Modules (dependence management)
- CRD + K8s Controller (resource management)
- StatefulSet (Redis cluster management)
- ConfigMap & Secret (for configuration management)
- PVC (data persistence)
- RBAC (permission management)
