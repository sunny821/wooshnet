#!/usr/bin/env bash
# set -euo pipefail

export TAG=${TAG:-tag}
# OVN configs
export OVN_DB_NODES=${OVN_DB_NODES}  # K8s nodes names, comma separated, e.g.: node1,node2,node3
export OVN_LOG=${OVN_LOG:-"/var/log/ovn"}
export OVS_LOG=${OVS_LOG:-"/var/log/openvswitch"}
export OVS_ETC=${OVS_ETC:-"/etc/openvswitch"}

echo "[Step 0] Create Namespace kube-system"
kubectl create ns kube-system

echo "Deploy multus"
kubectl apply -f yamls/multus.yaml

# 生成yaml文件
sh genyamls.sh

# apply yamls
sh applyovn.sh
sh applyapps.sh

echo "deploy complete."
tail -f /dev/null
