# How to create a CRD project
## What is Kubebuilder
Kubebuilder is a framework for building Kubernetes APIs using [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/).

It can easily generate a basic project for you to create your Kubernetes CRDs, so that you only need to focus on the implementation of the functionality without wasting time on the framework of the project.

## How to use Kubebuilder
Install Kubebuilder refer to [Kubebuilder-quick-start](https://book.kubebuilder.io/quick-start.html#installation).

**1. Create a golang project**
```golang
mkdir <project-name>
cd <project-name>
go mod init <module-name>
```
Code changes refer to [commit](https://github.com/qfzack/k8s-crd-demo/commit/477e045e7ddd246ecc2ade5c149a1d98c60201cc).

**2. Init project with Kubebuilder**
```shell
kubebuilder init --domain <domain-name>
```
Code changes refer to [commit](https://github.com/qfzack/k8s-crd-demo/commit/337aa140b0bb76bb56828163e894b5927a3c8b77).
> eg: `kubebuilder init --domain qfzack.com`

**3. Create Kubernetes API**
```shell
kubebuilder create api --group <group-name> --version <version-name> --kind <kind-name>
```
Code changes refer to [commit](https://github.com/qfzack/k8s-crd-demo/commit/e724fffea79fd5379099507f3ec1fc164a7ffffe).
> eg: `kubebuilder create api --group databases --version v1 --kind Redis`

**4. Functional implementation**
- Complete file `api/vi/<kind-name>_types.go` to define your CRD fields.
- Complete file `internal/controller/<kind-name>_controller.go` to implement your Kubernetes API for **custom resource (CR)** control logic.

**5. Apply CRD to Kubernetes cluster**
> Kubebuilder also generates a **Makefile** that contains common operations such as CRD updates, operator service startup, etc.
```shell
make install
```
It will generate a CRD yaml file in `config/crd/bases` path and apply it to Kubernetes cluster.

**6. Run operator for CRD monitoring**
```shell
make run
```
After we define the fields of the CRD, we can create the corresponding CR, but this is just a yaml file to put in k8s.

So we need operator to monitor our configuration changes, and we can achieve the resource management functionality we want by calling the k8s api.
