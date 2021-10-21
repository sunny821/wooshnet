#!/bin/sh

echo $SHELL
echo "BUILD_TIMESTAMP: ${BUILD_TIMESTAMP}"
export GOARCH=${GOARCH:-amd64}
export VERSION=${VERSION:-v0.1.7-${GOARCH}}
if [ -n "${BUILD_TIMESTAMP}" ];then
    export VERSION=${VERSION}-${BUILD_TIMESTAMP}
fi
export TAG=${TAG:-tag}

export REPO=${REPO:-"127.0.0.1/woosh"}
export K8SMASTER=${K8SMASTER:-"127.0.0.1"}

export OVN_NB_DB=${OVN_NB_DB:-"tcp:127.0.0.1:6641"}
export OVN_SB_DB=${OVN_SB_DB:-"tcp:127.0.0.1:6642"}
export CNI_CONF_PATH=${CNI_CONF_PATH:-"/etc/cni/net.d"}
export CNI_BIN_PATH=${CNI_BIN_PATH:-"/opt/cni/bin"}
export IAAS_VPC_CIDR=${IAAS_VPC_CIDR:-"172.31.0.0/16"}
export IAAS_SVC_CIDR=${IAAS_SVC_CIDR:-"11.254.0.0/16"}
export RS_NS=${RS_NS:-"kube-system"}

# OVN configs
export OVN_COUNT=${OVN_COUNT:-"3"}
export OVN_DB_NODES=${OVN_DB_NODES:-"127.0.0.1"}  # K8s nodes names, comma separated, e.g.: node1,node2,node3
export OVN_DB_IPS=${OVN_DB_IPS:-"127.0.0.1"}
export OVN_DB_PATH=${OVN_DB_PATH:-"/vdata/spider/ovn"}
export OVN_LOG=${OVN_LOG:-"/var/log/ovn"}
export OVS_LOG=${OVS_LOG:-"/var/log/openvswitch"}
export OVS_ETC=${OVS_ETC:-"/etc/openvswitch"}

export OVNOVSVERSION=${OVNOVSVERSION:-2.15.1_ubuntu_20.12.1_1}

mkdir -p yamls/$TAG/

envtotext(){
    ./tools/envtotext -i ./yamls/template/$1.yaml -o ./yamls/$TAG/$1.yaml
}

if [ $# -eq 0 ];then
    envtotext wooshnet-controller
    envtotext wooshnet-daemon
    envtotext ovn
fi

if [ $# -eq 1 ];then
    envtotext $1
fi
