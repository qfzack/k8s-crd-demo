# How to create a CRD project

## What is Kubebuilder

Kubebuilder is a framework for building Kubernetes APIs using [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/).

It can easily generate a basic project for you to create your Kubernetes CRDs, so that you only need to focus on the implementation of the functionality without wasting time on the framework of the project.

## Start with Kubebuilder

Install Kubebuilder refer to [Kubebuilder-quick-start](https://book.kubebuilder.io/quick-start.html#installation).

**1.Create a golang project**

```golang
mkdir <project-name>
cd <project-name>
go mod init <module-name>
```

Code changes refer to [commit](https://github.com/qfzack/redis-operator/commit/477e045e7ddd246ecc2ade5c149a1d98c60201cc).

**2.Init project with Kubebuilder**

```shell
kubebuilder init --domain <domain-name>
```

Code changes refer to [commit](https://github.com/qfzack/redis-operator/commit/337aa140b0bb76bb56828163e894b5927a3c8b77).

> eg: `kubebuilder init --domain qfzack.com`

**3.Create Kubernetes API**

```shell
kubebuilder create api --group <group-name> --version <version-name> --kind <kind-name>
```

Code changes refer to [commit](https://github.com/qfzack/redis-operator/commit/e724fffea79fd5379099507f3ec1fc164a7ffffe).

> eg: `kubebuilder create api --group databases --version v1 --kind Redis`

**4.Feature implementation**

- `api/vi/<kind-name>_types.go` is to define custom CRD fields.
- `internal/controller/<kind-name>_controller.go` is to implement Kubernetes API for **custom resource (CR)** management.

## Apply CRD resources to cluster

**1.Generate CRD configuration**

Generate CRD configurations from code.

```shell
make manifests
```

**2.Apply CRD to Kubernetes cluster**

> Kubebuilder also generate a **Makefile** that contains common operations for CRD, such as: CRD updates, operator service startup, etc.

```shell
make install
```

It will generate the CRD yaml file in directory `config/crd/bases` and apply it to Kubernetes cluster.

**3.Create custom resource**

After CRD appled to Kubernetes cluster, it is equivalent defined a new resource (source name specified with --kind <kind-name>), and then we are abled to creat this kind custom resource.

[databases_v1_redis.yaml](../config/samples/databases_v1_redis.yaml) is an example could be used to deployed in Kubernetes cluster with:

```shell
kubectl apply -f ./config/samples/databases_v1_redis.yaml -n <namespace>
```

**4.Run Operator as CRD controller**

After we created CR in Kubernetes cluster, it is just created a resource instance (like deployment in K8s) but not create pod.

And then need operator to monitor our configuration changes, and we can achieve the resource management functionality we want by calling the k8s API.

```shell
make run
```

**5.Build redis manager image**

Build redis manager image with dockerfile and push to docker hub:

```shell
docker build -t zhangqf29/redis-manager:<tag> .
docker push zhangqf29/redis-manager:<tag>
```

Run as redis operator in local:

```shell
docker run -v ~/.kube/config:/.kube/config -e KUBECONFIG=/.kube/config zhangqf29/redis-manager:<tag>
```

## Configure monitoring

[prometheus config](../config/prometheus/) is used to configure the Prometheus monitoring, which is a resource of type CRD `ServiceMonitor`, so it neccessary to install this CRD before use monitoring:

```shell
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack -n <namespace>
```

and then apply redis prometheus monitoring configs with:

```shell
kubectl apply -k ./config/prometheus
```

check ServiceMonitor CR status with:

```shell
kubectl get crd
kubectl get servicemonitors.monitoring.coreos.com -n monitoring
```

## Update Helm Chart

use kustomize and helmify to update configs to helm chart.

```shell
kustomize build config/default | helmify charts/redis-operator
```