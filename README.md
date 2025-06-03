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
- Helm >= 3.0 (optional)
- Kubectl >= 1.20

### Installation

#### Using Helm (Recommended)

```bash
# Add Helm repository
helm repo add redis-operator https://qfzack.github.io/redis-operator
helm repo update

# Install Redis Operator
helm install redis-operator redis-operator/redis-operator
```

#### Option 2: Using kubectl

```bash
kubectl apply -k ./config/default
```

## Configuration

### Redis CR Specification

| Parameter | Description | Default |
|-----------|-------------|---------|
| `spec.mode` | Redis deployment mode (standalone/sentinel/cluster) | standalone |
| `spec.version` | Redis version | 7.0.0 |
| `spec.replicas` | Number of Redis nodes | 3 |
| `spec.resource.requests.cpu` | CPU request quota | 100m |
| `spec.resource.requests.memory` | Memory request quota | 128Mi |
| `spec.resource.limits.cpu` | CPU limit quota | 200m |
| `spec.resource.limits.memory` | Memory limit quota | 256Mi |
| `spec.storage.accessMode` | Storage access mode | ReadWriteOnce |
| `spec.storage.storage` | Storage size | 100Mi |
| `spec.storage.storageClassName` | Storage class name | standard |
| `spec.config.REDIS_PASSWORD` | Redis access password | - |
| `spec.backup.enabled` | Enable backup | false |
| `spec.backup.schedule` | Backup schedule (Cron expression) | 0 0 * * * |
| `spec.backup.retention` | Backup retention days | 7 |
| `spec.security.enableTLS` | Enable TLS | false |

For detailed configuration options, see the [Configuration Guide](docs/configuration.md).

## Documentation

- [Architecture Overview](docs/architecture.md)
- [User Guide](docs/user-guide.md)
- [Developer Guide](docs/development.md)
- [Troubleshooting](docs/troubleshooting.md)

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

# Install dependencies
make deps

# Run tests
make test

# Run operator locally
make run
```

See [Developer Guide](docs/development.md) for detailed instructions.

## Roadmap

- [x] Basic Redis deployment support
- [x] Sentinel mode support
- [x] Cluster mode support
- [x] Prometheus monitoring
- [ ] Backup and restore
- [ ] Automated scaling
- [ ] Enhanced security features
- [ ] Cross-cluster deployment

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on how to submit pull requests.

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
