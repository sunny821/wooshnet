package cniapi

import (
	"fmt"
	"log"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
)

const NETNSPATH = "/var/run/netns/"

func Add(id, ifname, netns, namespace, name, subnetid, ip string) error {
	inputData := `{"cniVersion":"0.4.0","name":"wooshcni","type":"wooshcni","ports":[{"fixed_ips":[{"subnet_id":"` + subnetid + `","ip_address":"` + ip + `"}]}]}`
	return add(id, ifname, netns, namespace, name, inputData)
}

func Del(id, ifname, netns, namespace, name, subnetid, ip string) error {
	inputData := `{"cniVersion":"0.4.0","name":"wooshcni","type":"wooshcni","ports":[{"fixed_ips":[{"subnet_id":"` + subnetid + `","ip_address":"` + ip + `"}]}]}`
	return del(id, ifname, netns, namespace, name, inputData)
}

func add(id, ifname, netns, namespace, name, inputData string) error {
	if !strings.HasPrefix(netns, "/") {
		netns = NETNSPATH + netns
	}
	var args skel.CmdArgs
	args.ContainerID = id
	args.IfName = ifname
	args.Netns = netns
	args.Path = "/opt/cni/bin"
	args.Args = fmt.Sprintf(`IgnoreUnknown=true;K8S_POD_NAMESPACE=%s;K8S_POD_NAME=%s;K8S_POD_INFRA_CONTAINER_ID=%s`, namespace, name, args.ContainerID)
	args.StdinData = []byte(inputData)
	err := CmdAdd(&args)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func del(id, ifname, netns, namespace, name, inputData string) error {
	if !strings.HasPrefix(netns, "/") {
		netns = NETNSPATH + netns
	}
	var args skel.CmdArgs
	args.ContainerID = id
	args.IfName = ifname
	args.Netns = netns
	args.Path = "/opt/cni/bin"
	args.Args = fmt.Sprintf(`IgnoreUnknown=true;K8S_POD_NAMESPACE=%s;K8S_POD_NAME=%s;K8S_POD_INFRA_CONTAINER_ID=%s`, namespace, name, args.ContainerID)
	args.StdinData = []byte(inputData)
	err := CmdDel(&args)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
