#!/usr/bin/env bash
# set -euo pipefail

TAG=${TAG:-tag}

export OVN_LOG=${OVN_LOG:-"/var/log/ovn"}
export OVS_LOG=${OVS_LOG:-"/var/log/openvswitch"}
export OVS_ETC=${OVS_ETC:-"/etc/openvswitch"}
export WOOSHNET_CONTROLLER_COUNT=${WOOSHNET_CONTROLLER_COUNT:-"1"}

echo "[Step 0] Deploy wooshnet-controller"
echo "Deploy wooshnet-controller"
kubectl apply -f yamls/${TAG}/wooshnet-controller.yaml
kubectl rollout status deployment/wooshnet-controller -n kube-system
kubectl scale --replicas=${WOOSHNET_CONTROLLER_COUNT} deployment/wooshnet-controller -n kube-system
echo "-------------------------------"
echo ""

echo "[Step 0] Deploy wooshnet-daemon"
echo "Deploy wooshnet-daemon"
kubectl apply -f yamls/${TAG}/wooshnet-daemon.yaml
kubectl rollout status daemonset/wooshnet-daemon -n kube-system
echo "-------------------------------"
echo ""

echo "Deploy Success !"

