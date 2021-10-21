package cniapi

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	networkv1 "wooshnet/apis/network/v1"
	"wooshnet/pkg/wooshtools"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/040"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"k8s.io/klog/v2"
)

const WooshPortDir = "/var/run/wooshnet/wooshport/"

func IPVersion(ip string) string {
	addr := net.ParseIP(ip)
	if addr != nil {
		if strings.Contains(ip, ".") {
			return "4"
		}
		if strings.Contains(ip, ":") {
			return "6"
		}
	}
	return ""
}

func CmdAdd(args *skel.CmdArgs) error {
	klog.Infof("Add start, %v, %v, %v, %v, %v\n", args.ContainerID, args.IfName, args.Args, args.Netns, args.Path)
	defer klog.Infof("Add end, %v\n", args.ContainerID)
	// klog.Infof("%s", args.StdinData)

	netConf, err := LoadNetConf(args.StdinData)
	if err != nil {
		klog.Errorln(err)
		return types.NewError(types.ErrUnknown, fmt.Sprintf("loadNetConf Error: %v", err), "")
	}
	// klog.Infof("%v %v", netConf.CNIVersion, netConf.PrevResult)
	// podName, err := parseValueFromArgs("K8S_POD_NAME", args.Args)
	// if err != nil {
	// 	return cniReq, types.NewError(types.ErrUnknown, fmt.Sprintf("get podName Error: %v", err), "")
	// }
	// podNamespace, err := parseValueFromArgs("K8S_POD_NAMESPACE", args.Args)
	// if err != nil {
	// 	return cniReq, types.NewError(types.ErrUnknown, fmt.Sprintf("get podNamespace Error: %v", err), "")
	// }
	// cniReq.PodName = podName
	// cniReq.PodNamespace = podNamespace

	kvs, err := parseExtraArgs(args.Args)
	if err != nil {
		klog.Infof("%v", err)
		return err
	}
	podName := kvs["K8S_POD_NAME"]
	podNamespace := kvs["K8S_POD_NAMESPACE"]
	if err != nil {
		klog.Errorln(err)
		return types.NewError(types.ErrUnknown, "参数读取错误 Error", err.Error())
	}

	wooshPortName := netConf.WooshPort
	var wp *networkv1.WooshPort
	for i := 30; i > 0; i-- {
		if wooshPortName != "" {
			wp, err = GetWooshPortFromCR(podNamespace, wooshPortName)
		} else {
			wp, err = GetWooshPortFromPod(podNamespace, podName)
		}
		if err != nil {
			klog.Errorln(err)
		} else {
			wooshPortName = wp.Name
			// klog.Infoln(wp.Status.DeviceReady, wp.Status.PodName, podName)
			if wp.Status.DeviceReady {
				if wp.Spec.AutoCreated || wp.Status.PodName == podName {
					break
				}
			}
		}
		if i > 0 {
			time.Sleep(time.Second)
			continue
		}
	}
	if wp == nil {
		err = fmt.Errorf("netdevice not ready")
		return types.NewError(types.ErrUnknown, "Error", err.Error())
	}
	if wp.Name != podName {
		_ = os.MkdirAll(WooshPortDir, 0755)
		err = os.WriteFile(WooshPortDir+podNamespace+"_"+podName, []byte(wp.Name), 0644)
		if err != nil {
			klog.Errorln(err)
			return types.NewError(types.ErrUnknown, "Error", err.Error())
		}
	}
	klog.Infoln("deviceReady:", wp.Status.DeviceReady)

	if !wp.Status.DeviceReady {
		if wp.Status.Message != "" {
			err = fmt.Errorf(wp.Status.Message)
		} else {
			err = fmt.Errorf("ovs device not ready")
		}
		klog.Errorln(err)
		return types.NewError(types.ErrUnknown, "Error", err.Error())
	}
	result, err := moveToNetNS(args.Netns, netConf.CNIVersion, wp)
	if err != nil {
		klog.Errorln(err)
		UpdateDeviceReady(wp.Namespace, wp.Name)
		return types.NewError(types.ErrUnknown, "Error", err.Error())
	}
	klog.Infoln(result)
	newResult := current.Result{
		CNIVersion: result.CNIVersion,
	}
	klog.Infoln(args.IfName)
	var rindex int
	for index, iface := range result.Interfaces {
		if iface.Name == args.IfName {
			newResult.Interfaces = append(newResult.Interfaces, iface)
			rindex = index
			break
		}
	}
	klog.Infoln(newResult.Interfaces)
	for _, ipconf := range result.IPs {
		if *ipconf.Interface == rindex {
			// ipconf.Interface = current.Int(0)
			newResult.IPs = append(newResult.IPs, ipconf)
		}
	}
	klog.Infoln(newResult.IPs)
	return types.PrintResult(&newResult, netConf.CNIVersion)
}

func CmdCheck(args *skel.CmdArgs) error {

	return nil
}

func CmdDel(args *skel.CmdArgs) error {
	klog.Infof("Del start, %v, %v, %v, %v, %v\n", args.ContainerID, args.IfName, args.Args, args.Netns, args.Path)
	defer klog.Infof("Del end, %v\n", args.ContainerID)
	klog.Infof("%s", args.StdinData)

	// netConf, err := LoadNetConf(args.StdinData)
	// if err != nil {
	// 	klog.Errorln(err)
	// 	return types.NewError(types.ErrUnknown, fmt.Sprintf("loadNetConf Error: %v", err), "")
	// }
	// klog.Infof("%v %v", netConf.CNIVersion, netConf.PrevResult)

	// podName, err := parseValueFromArgs("K8S_POD_NAME", args.Args)
	// if err != nil {
	// 	return cniReq, types.NewError(types.ErrUnknown, fmt.Sprintf("get podName Error: %v", err), "")
	// }
	// podNamespace, err := parseValueFromArgs("K8S_POD_NAMESPACE", args.Args)
	// if err != nil {
	// 	return cniReq, types.NewError(types.ErrUnknown, fmt.Sprintf("get podNamespace Error: %v", err), "")
	// }
	// cniReq.PodName = podName
	// cniReq.PodNamespace = podNamespace

	kvs, err := parseExtraArgs(args.Args)
	if err != nil {
		klog.Infof("%v", err)
		return err
	}
	podName := kvs["K8S_POD_NAME"]
	podNamespace := kvs["K8S_POD_NAMESPACE"]

	var wp *networkv1.WooshPort
	wpname := podName
	if wooshtools.FileExist(WooshPortDir + podNamespace + "_" + podName) {
		buf, err := os.ReadFile(WooshPortDir + podNamespace + "_" + podName)
		if err != nil {
			klog.Warningln(err)
		} else if len(buf) > 0 {
			wpname = strings.TrimSpace(string(buf))
		}
	}
	for i := 10; i > 0; i-- {
		wp, err = GetWooshPortFromCR(podNamespace, wpname)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			klog.Warningln(err)
		} else {
			break
		}
		if i > 0 {
			time.Sleep(time.Second)
			continue
		}
	}
	if wp != nil {
		wpname = wp.Name
		err = removeFromNetNS(args.Netns, wp)
		if err != nil {
			klog.Warningln(err)
		}
	} else {
		klog.Warningln(err)
	}
	klog.Infoln("wooshport:", wpname)
	err = UpdateDeviceReady(podNamespace, wpname)
	if err != nil {
		klog.Warningln(err)
	}

	return nil
}

//moveToNetNS
func moveToNetNS(netnspath, cniVersion string, obj *networkv1.WooshPort) (current.Result, error) {
	result := current.Result{CNIVersion: cniVersion}
	if netnspath == "" {
		return result, nil
	}
	var err error
	podns, err := ns.GetNS(netnspath)
	if err != nil {
		return result, err
	}
	defer podns.Close()
	for index, status := range obj.Status.PortStatus {
		result.Interfaces = append(result.Interfaces, &current.Interface{
			Name:    status.IfName,
			Mac:     status.MACAddress,
			Sandbox: netnspath,
		})
		for _, fixedIP := range status.FixedIPs {
			version := IPVersion(fixedIP.IPAddress)
			_, mask, err := net.ParseCIDR(fixedIP.Cidr)
			if err != nil {
				return result, err
			}
			var fip net.IP
			var fgw net.IP
			var route *types.Route
			switch version {
			case "4":
				fip = net.ParseIP(fixedIP.IPAddress).To4()
				fgw = net.ParseIP(fixedIP.Gateway).To4()
				route = &types.Route{
					Dst: net.IPNet{IP: net.ParseIP("0.0.0.0").To4(), Mask: net.CIDRMask(0, 32)},
					GW:  net.ParseIP(fixedIP.Gateway).To4(),
				}
			case "6":
				fip = net.ParseIP(fixedIP.IPAddress).To16()
				fgw = net.ParseIP(fixedIP.Gateway).To16()
				route = &types.Route{
					Dst: net.IPNet{IP: net.ParseIP("::").To16(), Mask: net.CIDRMask(0, 128)},
					GW:  net.ParseIP(fixedIP.Gateway).To16(),
				}
			}
			faddress := net.IPNet{IP: fip, Mask: mask.Mask}
			result.IPs = append(result.IPs, &current.IPConfig{
				Version:   version,
				Interface: current.Int(index),
				Address:   faddress,
				Gateway:   fgw,
			})
			result.Routes = append(result.Routes, route)
		}
	}
	for index, status := range obj.Status.PortStatus {
		link, err := netlink.LinkByName(status.NicName)
		if err != nil {
			continue
		}
		// 设置Mac
		macAddr, err := net.ParseMAC(status.MACAddress)
		if err != nil {
			return result, err
		}
		err = netlink.LinkSetHardwareAddr(link, macAddr)
		if err != nil {
			return result, err
		}
		err = netlink.LinkSetNsFd(link, int(podns.Fd()))
		if err != nil {
			return result, err
		}
		err = podns.Do(func(_ ns.NetNS) error {
			// 设置IP
			if status.Interface != status.NicName {
				err = netlink.LinkSetName(link, status.Interface)
				if err != nil {
					return err
				}
			}
			err = netlink.LinkSetUp(link)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Println(err)
			return result, err
		}
		for _, ipconfig := range result.IPs {
			if *ipconfig.Interface != index {
				continue
			}
			err = podns.Do(func(_ ns.NetNS) error {
				// 设置IP
				ipAddr, err := netlink.ParseAddr(ipconfig.Address.String())
				if err != nil {
					log.Println(err)
					return err
				}
				return netlink.AddrAdd(link, ipAddr)
			})
			if err != nil {
				log.Println(err)
				continue
			}
		}
		for _, route := range result.Routes {
			err = podns.Do(func(_ ns.NetNS) error {
				return netlink.RouteAdd(&netlink.Route{
					LinkIndex: link.Attrs().Index,
					Scope:     netlink.SCOPE_UNIVERSE,
					Dst:       &route.Dst,
					Gw:        route.GW,
				})
			})
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}

	return result, nil
}

//removeFromNetNS
func removeFromNetNS(netnspath string, obj *networkv1.WooshPort) error {
	if netnspath == "" {
		return nil
	}
	curns, err := ns.GetCurrentNS()
	if err != nil {
		return err
	}
	defer curns.Close()
	podns, err := ns.GetNS(netnspath)
	if err != nil {
		return err
	}
	defer podns.Close()
	for _, portStatus := range obj.Status.PortStatus {
		err = podns.Do(func(_ ns.NetNS) error {
			link, err := netlink.LinkByName(portStatus.Interface)
			if err != nil {
				return err
			}
			err = netlink.LinkSetDown(link)
			if err != nil {
				return err
			}
			err = netlink.LinkSetName(link, portStatus.NicName)
			if err != nil {
				return err
			}
			return netlink.LinkSetNsFd(link, int(curns.Fd()))
		})
		if err != nil {
			continue
		}
	}
	return nil
}
