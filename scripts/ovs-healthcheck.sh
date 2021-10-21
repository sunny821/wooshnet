#!/bin/bash
set -euo pipefail
shopt -s expand_aliases

OVN_DB_IPS=${OVN_DB_IPS:-}

function gen_conn_str {
  t=$(echo -n "${OVN_DB_IPS}" | sed 's/[[:space:]]//g' | sed 's/,/ /g')
  x=$(for i in ${t}; do echo -n "tcp:[$i]:$1",; done| sed 's/,$//')
  echo "$x"
}

echo Connecting OVN SB "$(gen_conn_str 6642)"
ovsdb-client list-dbs "$(gen_conn_str 6642)"

alias ovs-ctl='/usr/share/openvswitch/scripts/ovs-ctl'

ovs-ctl status
