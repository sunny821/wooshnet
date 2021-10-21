#!/bin/bash
set -euo pipefail

OVN_DB_IPS=${OVN_DB_IPS:-}
NODE_IP=${POD_IP:-}
if [ -z $NODE_IP ];then 
    echo "环境变量POD_IP为空"
    exit 1
fi

UUID=$(uuidgen --sha1 --namespace @dns --name ${NODE_IP})
echo $NODE_IP $UUID

# Check required kernel module
modinfo openvswitch
modinfo geneve

# https://bugs.launchpad.net/neutron/+bug/1776778
if grep -q "3.10.0-862" /proc/version
then
    echo "kernel version 3.10.0-862 has a nat related bug that will affect ovs function, please update to a version greater than 3.10.0-898"
    exit 1
fi

# https://bugs.launchpad.net/ubuntu/+source/linux/+bug/1794232
if [ ! -f "/proc/net/if_inet6" ] && grep -q "3.10" /proc/version ; then
    echo "geneve requires ipv6, please add ipv6.disable=0 to kernel follow the instruction below:"
    echo "
vi /etc/default/grub
find GRUB_CMDLINE_LINUX=  and change ipv6.disable=1 to ipv6.disable=0
grub2-mkconfig -o /boot/grub2/grub.cfg
reboot
cat /proc/cmdline"
    exit 1
fi

function quit {
	/usr/share/openvswitch/scripts/ovs-ctl stop
	exit 0
}
trap quit EXIT

# Start ovsdb
/usr/share/openvswitch/scripts/ovs-ctl restart --no-ovs-vswitchd --system-id=${UUID}
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-handler-threads=10
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-revalidator-threads=10

# Start vswitchd
/usr/share/openvswitch/scripts/ovs-ctl restart --no-ovsdb-server --system-id=${UUID}
/usr/share/openvswitch/scripts/ovs-ctl --protocol=udp --dport=6081 enable-protocol

chmod 600 /etc/openvswitch/*
tail -f /var/log/openvswitch/ovs-vswitchd.log
