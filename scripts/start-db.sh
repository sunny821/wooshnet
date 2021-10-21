#!/bin/bash
set -eo pipefail

DB_NB_ADDR=${DB_NB_ADDR:-::}
DB_NB_PORT=${DB_NB_PORT:-6641}
DB_SB_ADDR=${DB_SB_ADDR:-::}
DB_SB_PORT=${DB_SB_PORT:-6642}

function gen_conn_str {
  t=$(echo -n "${OVN_DB_IPS}" | sed 's/[[:space:]]//g' | sed 's/,/ /g')
  x=$(for i in ${t}; do echo -n "tcp:[$i]:$1",; done| sed 's/,$//')
  echo "$x"
}

function get_leader_ip {
  # If no available leader the first ip will be the leader
  t=$(echo -n "${OVN_DB_IPS}" | sed 's/[[:space:]]//g' | sed 's/,/ /g')
  echo -n "${t}" | cut -f 1 -d " "
}

function quit {
    /usr/share/ovn/scripts/ovn-ctl stop_northd
    exit 0
}
trap quit EXIT

if [[ ! "$OVN_DB_IPS" =~ "$POD_IP" ]]; then
  echo "ERROR! host ip $POD_IP not in env OVN_DB_IPS $OVN_DB_IPS"
  exit 1
fi
/usr/share/ovn/scripts/ovn-ctl stop_northd

nb_leader_ip=$(get_leader_ip nb)
sb_leader_ip=$(get_leader_ip sb)
if [[ "$nb_leader_ip" == "${POD_IP}" ]]; then
    # Start ovn-northd, ovn-nb and ovn-sb
    /usr/share/ovn/scripts/ovn-ctl \
        --db-nb-create-insecure-remote=yes \
        --db-sb-create-insecure-remote=yes \
        --db-nb-cluster-local-addr="[${POD_IP}]" \
        --db-sb-cluster-local-addr="[${POD_IP}]" \
        --db-nb-addr=[::] \
        --db-sb-addr=[::] \
        --ovn-northd-nb-db="$(gen_conn_str 6641)" \
        --ovn-northd-sb-db="$(gen_conn_str 6642)" \
        start_northd
    ovn-nbctl --no-leader-only set-connection ptcp:"${DB_NB_PORT}":0.0.0.0
    ovn-nbctl --no-leader-only set Connection . inactivity_probe=180000
    ovn-sbctl --no-leader-only set-connection ptcp:"${DB_SB_PORT}":0.0.0.0
    ovn-sbctl --no-leader-only set Connection . inactivity_probe=180000
else
    # Start ovn-northd, ovn-nb and ovn-sb
    /usr/share/ovn/scripts/ovn-ctl \
        --db-nb-create-insecure-remote=yes \
        --db-sb-create-insecure-remote=yes \
        --db-nb-cluster-local-addr="[${POD_IP}]" \
        --db-sb-cluster-local-addr="[${POD_IP}]" \
        --db-nb-cluster-remote-addr="[${nb_leader_ip}]" \
        --db-sb-cluster-remote-addr="[${sb_leader_ip}]" \
        --db-nb-addr=[::] \
        --db-sb-addr=[::] \
        --ovn-northd-nb-db="$(gen_conn_str 6641)" \
        --ovn-northd-sb-db="$(gen_conn_str 6642)" \
        start_northd
fi

chmod 600 /etc/ovn/*
tail -f /var/log/ovn/ovn-northd.log
