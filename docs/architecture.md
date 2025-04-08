# Architecture

## Operator Design

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

### Data Persistance

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
- PVC (data persistance)
- RBAC (permission management)