# wooshnet

An implement of CNI depending on neutron API

## build

```shell
# 生成crd,和deepcopy
make manifests generate
# 编译
# make build image
make wooshnet

# 生成部署yaml
TAG=dev-001 REPO=127.0.0.1/woosh sh tools/genyamls.sh

# configmap
kubectl apply -f yamls/woosh-config.yaml

# wooshnet
kubectl apply -f yamls/dev-001/wooshnet-controller.yaml
kubectl apply -f yamls/dev-001/wooshnet-daemon.yaml

```
