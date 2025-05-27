# Redis Operator

[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/qfzack/redis-operator)](https://goreportcard.com/report/github.com/qfzack/redis-operator)

Redis Operator 是一个基于 Kubernetes 的运维工具，用于自动化部署和管理 Redis 集群。本项目使用 [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) 构建。

## 功能特性

- 自动部署高可用的 Redis 集群
- 支持 Redis 主从复制模式
- 支持 Redis Sentinel 哨兵模式
- 支持动态扩缩容
- 自动故障转移
- 持久化存储支持

## 架构设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Kubernetes Cluster                             │
│                                                                         │
│  ┌─────────────────┐          ┌──────────────────┐                      │
│  │  Redis Operator │          │   Redis CRD      │                      │
│  │   Controller    │◄────────►│   Definition     │                      │
│  └────────┬────────┘          └──────────────────┘                      │
│           │                                                             │
│           │                    Reconcile                                │
│           ▼                                                             │
│  ┌─────────────────┐          ┌──────────────────┐                      │
│  │  Redis Master   │          │  Redis Replica   │                      │
│  │  StatefulSet    │◄────────►│   StatefulSet    │                      │
│  └────────┬────────┘          └──────────┬───────┘                      │
│           │                              │                              │
│           │         ┌──────────────────┐ │                              │
│           └────────►│ Redis Sentinel   │◄┘                              │
│                    │   Deployment      │                                │
│                    └─────────┬───────·─-┘                                │
│                              │                                          │
│                    ┌─────────▼────────┐                                 │
│                    │  Service (HA)    │                                 │
│                    └──────────────────┘                                 │
└─────────────────────────────────────────────────────────────────────────┘
```

Redis Operator 通过以下组件实现 Redis 集群的自动化管理：

- **Custom Resource Definition (CRD)**: 定义 Redis 集群的规格
- **Controller**: 监控集群状态并确保与期望状态一致
- **Redis StatefulSet**: 管理 Redis 主从节点
- **Redis Sentinel**: 提供高可用和自动故障转移

## 快速开始

### 前置条件

- Kubernetes 1.32+
- Helm 3.0+ (可选，用于 Helm 安装方式)
- Kubectl 1.23+

### 安装

#### 方式一：使用 kubectl

```bash
# 安装 CRD 和 Operator
kubectl apply -f https://raw.githubusercontent.com/qfzack/redis-operator/main/deploy/install.yaml
```

#### 方式二：使用 Helm

```bash
helm repo add redis-operator https://qfzack.github.io/redis-operator
helm install redis-operator redis-operator/redis-operator
```

### 部署 Redis 集群

创建一个简单的 Redis 集群：

```yaml
apiVersion: databases.qfzack.com/v1
kind: Redis
metadata:
  name: redis-sample
spec:
  replicas: 3
  mode: sentinel
  version: "6.2"
  persistentVolume:
    size: 1Gi
```

应用配置：

```bash
kubectl apply -f config/samples/redis-sentinel.yaml
```

### 验证部署

检查 Redis 集群状态：

```bash
kubectl get redis
kubectl get pods -l app=redis-sample
```

## 配置参考

### Redis CR 规格

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `spec.replicas` | Redis 副本数量 | 3 |
| `spec.mode` | 运行模式 (sentinel/cluster) | sentinel |
| `spec.version` | Redis 版本 | 6.2 |
| `spec.persistentVolume.size` | 存储大小 | 1Gi |

完整配置示例请参考 [配置文档](docs/configuration.md)。

## 开发指南

### 构建要求

- Go 1.22+
- Docker 17.03+
- Operator SDK v1.28.0+

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/qfzack/redis-operator.git

# 安装依赖
make deps

# 运行测试
make test

# 构建镜像
make docker-build IMG=<your-registry>/redis-operator:tag
```

详细的开发指南请参考 [开发文档](docs/development.md)。

## 路线图

- [ ] 支持 Redis Cluster 模式
- [ ] 自动备份与恢复
- [ ] 监控集成
- [ ] 升级策略优化

## 故障排除

常见问题及解决方案请参考 [故障排除指南](docs/troubleshooting.md)。

## 贡献指南

欢迎提交 Issue 和 Pull Request！详情请参考 [贡献指南](CONTRIBUTING.md)。

## 社区

- [Slack Channel](https://kubernetes.slack.com/messages/redis-operator)
- [邮件列表](https://groups.google.com/forum/#!forum/redis-operator)

## 许可证

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0