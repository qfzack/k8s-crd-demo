# Redis Operator

[![Docker Image](https://github.com/qfzack/redis-operator/actions/workflows/docker-image-build.yml/badge.svg?branch=redis-operator)](https://github.com/qfzack/redis-operator/actions/workflows/docker-image-build.yml)
[![Helm Chart](https://github.com/qfzack/redis-operator/actions/workflows/helm-chart-build.yml/badge.svg?branch=redis-operator)](https://github.com/qfzack/redis-operator/actions/workflows/helm-chart-build.yml)
[![Lint](https://github.com/qfzack/redis-operator/actions/workflows/golangci-lint.yml/badge.svg?branch=redis-operator)](https://github.com/qfzack/redis-operator/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/qfzack/redis-operator)](https://goreportcard.com/report/github.com/qfzack/redis-operator)
[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

Redis Operator is a Kubernetes operator built with [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) that automates the deployment, scaling, and management of Redis instances in Kubernetes clusters.

## Features

- Automated deployment and management of Redis instances
- Multiple deployment modes:
  - Standalone: Single Redis instance
  - Sentinel: High availability with master-slave replication
  - Cluster: Sharded cluster with automatic failover
- Dynamic scaling capabilities
- Automated backup and recovery
- Monitoring integration with Prometheus
- Persistent storage support
- Automated failover and high availability
- Configuration management via CRD

## Getting Started

### Prerequisites

- Kubernetes >= 1.20
- Kubectl >= 1.20
- Helm >= 3.0

### Installation

```bash
helm repo add redis-operator https://qfzack.github.io/redis-operator
helm install redis-operator redis-operator/redis-operator -n <namespace>
```

## Configuration

### Helm Chart Values Configuration

View all configurable parameters:

```bash
helm repo add redis-operator https://qfzack.github.io/redis-operator
helm show values redis-operator/redis-operator
```

And then you can customize the Redis Operator installation by modifying the chart values in several ways:

1. Using `--set` parameters:

```bash
helm install redis-operator redis-operator/redis-operator \
  --set redis.mode=sentinel \
  --set redis.replicas=3 \
  -n <namespace>
```

2. Using a custom values file refer to [values.yaml](./charts/redis-operator/values.yaml):

```bash
helm install redis-operator redis-operator/redis-operator \
  -f values.yaml \
  -n <namespace>
```

## Development

### Requirements

- Go >= 1.20
- Docker >= 20.10
- Operator SDK >= 1.28.0
- Kubebuilder >= 3.0.0

### Local Development

```bash
# Clone repository
git clone https://github.com/qfzack/redis-operator.git
cd redis-operator

# Generate CRD configurations
make manifests

# Install CRD to k8s cluster
make install

# Create custom resource
kubectl apply -f ./config/samples/databases_v1_redis.yaml -n <namespace>

# Run operator locally
make run
```

See [how to start](docs/how_to_start.md) for detailed instructions.

## Roadmap

- [x] Basic Redis deployment support
- [x] Multiple Deployment Modes
- [x] Persistent Storage Support
- [x] Dynamic Scaling
- [x] Automated Failover
- [x] Helm Chart Deployment
- [ ] Automated Backup and Restore
- [ ] Enhanced security features (TLS, ServiceAccount)
- [ ] Advanced alerting and event notifications
- [ ] Multi-tenancy support
- [ ] Cross-cluster deployment

## Community

- [Slack Channel](https://kubernetes.slack.com/messages/redis-operator)
- [GitHub Issues](https://github.com/qfzack/redis-operator/issues)
- [Mailing List](https://groups.google.com/forum/#!forum/redis-operator)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
