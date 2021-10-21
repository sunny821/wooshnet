#!/usr/bin/env bash
# set -euo pipefail

TAG=${TAG:-tag}
# OVN configs
OVN_DB_NODES=${OVN_DB_NODES}  # K8s nodes names, comma separated, e.g.: node1,node2,node3

echo "[Step 0] Create Namespace kube-system"
kubectl create ns kube-system

echo "label node"
for node in $(echo $OVN_DB_NODES | sed $'s/,/\\\n/g')
do
  if [ -z "$node" ];
  then
    continue
  fi
  kubectl label --overwrite no $node network/ovn=master
done

echo "[Step 5] Deploy OVN"
kubectl apply -f yamls/${TAG}/ovn.yaml
kubectl rollout status deployment/ovn-central -n kube-system
