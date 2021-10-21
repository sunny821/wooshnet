/*
Copyright 2021 xiayuhai.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/opencontainers/runtime-spec/specs-go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	networkv1 "wooshnet/apis/network/v1"
	"wooshnet/pkg/cri"
	"wooshnet/pkg/deviceclient"
	"wooshnet/pkg/netclient"
	"wooshnet/pkg/wooshtools"
)

// WooshPortReconciler reconciles a WooshPort object
type WooshPortReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	NetClient    netclient.NetClient
	DeviceClient deviceclient.DeviceClient
	Mode         string
	NodeName     string
}

//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=wooshports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=wooshports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=wooshports/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WooshPort object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.port/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *WooshPortReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	logger.Info(req.NamespacedName.String())

	var result ctrl.Result
	var err error
	obj := &networkv1.WooshPort{}
	err = r.Get(context.Background(), req.NamespacedName, obj)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info(req.NamespacedName.String() + " not found")
		} else {
			result.Requeue = true
			result.RequeueAfter = time.Millisecond * 200
			return result, err
		}
	} else if r.matched(obj) {
		if obj.DeletionTimestamp.IsZero() && !obj.Spec.Deleted {
			// DeletionTimestamp 为空时,为创建或更新事件
			if r.Mode == "daemon" {
				err = r.handlerUpdateCR(ctx, obj)
			} else {
				err = r.createOrUpdateCR(ctx, obj)
			}
			if err != nil {
				result.Requeue = true
				result.RequeueAfter = time.Millisecond * 200
				return result, err
			}
		} else {
			// DeletionTimestamp 不为空时,为删除事件
			if r.Mode == "daemon" {
				err = r.handlerDeleteCR(ctx, obj)
			} else {
				err = r.deleteCR(ctx, obj)
			}
			if err != nil {
				result.Requeue = true
				result.RequeueAfter = time.Millisecond * 200
				return result, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WooshPortReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Mode == "daemon" {
		return ctrl.NewControllerManagedBy(mgr).
			For(&networkv1.WooshPort{}).
			WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
				obj := object.(*networkv1.WooshPort)
				return obj.Status.NodeName == r.NodeName && len(obj.Status.PortStatus) > 0
			})).
			Complete(r)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1.WooshPort{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			obj := object.(*networkv1.WooshPort)
			return obj.Name != ""
		})).
		Complete(r)
}

// matched 检查是否需要处理
func (r *WooshPortReconciler) matched(obj *networkv1.WooshPort) bool {

	return true
}

// needUpdate 检查是否需要处理Update
func (r *WooshPortReconciler) needUpdate(oldobj, obj *networkv1.WooshPort) bool {
	ok := (obj.Status.DeviceReady != oldobj.Status.DeviceReady) ||
		(obj.Status.PortReady != oldobj.Status.PortReady) ||
		(obj.Status.NodeName != oldobj.Status.NodeName) ||
		(obj.Status.NodeIP != oldobj.Status.NodeIP) ||
		(obj.Status.Message != oldobj.Status.Message) ||
		(obj.Status.PodName != oldobj.Status.PodName) ||
		(obj.Status.PodPid != oldobj.Status.PodPid) ||
		(obj.Status.PodRuntimeType != oldobj.Status.PodRuntimeType) ||
		(obj.Status.PodNetns != oldobj.Status.PodNetns) ||
		(obj.Status.PodReady != oldobj.Status.PodReady) ||
		(len(obj.Status.PortStatus) != len(oldobj.Status.PortStatus))
	if ok {
		return ok
	}
	for index, status := range obj.Status.PortStatus {
		oldstatus := oldobj.Status.PortStatus[index]
		ok = ok ||
			(status.DeviceReady != oldstatus.DeviceReady) ||
			(status.PortReady != oldstatus.PortReady) ||
			(status.PortID != oldstatus.PortID) ||
			(status.ID != oldstatus.ID)
	}
	return ok
}

func (r *WooshPortReconciler) createOrUpdateCR(ctx context.Context, obj *networkv1.WooshPort) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	if obj.Spec.Deleted {
		return nil
	}
	oldobj := obj.DeepCopy()
	obj.Status.NodeName = obj.Spec.NodeName
	if obj.Spec.PodName != "" {
		obj.Status.PodName = obj.Spec.PodName
	}
	if obj.Status.PodName != "" {
		pod, err := r.getPod(obj.Namespace, obj.Status.PodName)
		if err != nil {
			if client.IgnoreNotFound(err) == nil && obj.Spec.AutoCreated {
				obj.Spec.Deleted = true
				return r.Update(context.TODO(), obj)
			}
			return err
		}
		obj.Status.NodeName = pod.Spec.NodeName
		obj.Status.NodeIP = pod.Status.HostIP
	} else {
		pod, err := r.getPod(obj.Namespace, obj.Name)
		if err != nil {
			if client.IgnoreNotFound(err) == nil && obj.Spec.AutoCreated {
				obj.Spec.Deleted = true
				return r.Update(context.TODO(), obj)
			}
			return err
		}
		obj.Status.NodeName = pod.Spec.NodeName
		obj.Status.NodeIP = pod.Status.HostIP
	}
	if obj.Status.NodeName != "" {
		node, err := r.getNode(obj.Status.NodeName)
		if err != nil {
			return err
		}
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				obj.Status.NodeIP = addr.Address
				break
			}
		}
	}
	count := 0
	for _, port := range obj.Spec.Ports {
		if len(port.FixedIPs) == 0 && port.NetworkID == "" {
			continue
		} else {
			count++
		}
	}
	if count == 0 {
		err := fmt.Errorf("no device specified")
		obj.Status.Message = err.Error()
		if r.needUpdate(oldobj, obj) {
			err := r.Status().Update(context.TODO(), obj)
			if err != nil {
				logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
				return err
			}
		}
		return err
	}

	var err error
	if len(obj.Spec.Ports) != len(obj.Status.PortStatus) {
		var statuses []networkv1.PortStatus
		for index, port := range obj.Spec.Ports {
			if port.NetworkID == "" && len(port.FixedIPs) == 0 {
				continue
			}

			var dsPtr *networkv1.PortStatus
			for _, status := range obj.Status.PortStatus {
				if index == status.Index && status.NetworkID != "" && len(status.FixedIPs) > 0 {
					dsPtr = &status
					break
				}
			}
			if dsPtr != nil {
				statuses = append(statuses, *dsPtr)
				continue
			}

			portStatus := networkv1.PortStatus{
				Port:   port,
				Index:  index,
				IfName: port.Name,
			}
			if len(portStatus.FixedIPs) == 0 {
				// 根据networkId查询subnet,将第一个subnetId添加到FixedIPs
				subnets, err := r.NetClient.ListSubnet(portStatus.NetworkID)
				if err != nil {
					logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
					return err
				}
				if len(subnets) > 0 {
					portStatus.FixedIPs = append(portStatus.FixedIPs, networkv1.IP{
						SubnetID: subnets[0].NetworkID,
					})
				}
			}
			logger.Info(msg, "portStatus", portStatus)
			statuses = append(statuses, portStatus)
		}
		obj.Status.PortStatus = statuses
	}
	// logger.Info(msg, "wooshPort.PortStatus", len(obj.Status.PortStatus))

	subnetMap := make(map[string]*subnets.Subnet)
	for index := range obj.Status.PortStatus {
		portStatus := &obj.Status.PortStatus[index]
		if len(portStatus.FixedIPs) == 0 {
			continue
		}
		if portStatus.ID != "" {
			portStatus.PortReady = true
			continue
		}
		if len(portStatus.FixedIPs) > 0 {
			// 根据subnetId查询netId,cidr和gateway,补充信息
			for i := 0; i < len(portStatus.FixedIPs); i++ {
				if portStatus.ProjectID != "" && portStatus.NetworkID != "" {
					continue
				}
				var err error
				var subnet *subnets.Subnet
				if sub, ok := subnetMap[portStatus.FixedIPs[i].SubnetID]; ok {
					subnet = sub
				} else {
					subnet, err = r.NetClient.GetSubnet(portStatus.FixedIPs[i].SubnetID)
					if err != nil {
						logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
						return err
					}
				}
				portStatus.NetworkID = subnet.NetworkID
				portStatus.ProjectID = subnet.ProjectID
				if subnet != nil {
					subnetMap[portStatus.FixedIPs[i].SubnetID] = subnet
				}
			}
		}
		// 创建Port
		if len(portStatus.FixedIPs) > 1 || portStatus.QoSPolicyID != "" || len(portStatus.SecurityGroups) > 0 || portStatus.FixedIPs[0].IPAddress != "" {
			// 多子网,或者指定IP时,调接口直接创建port
			port, err := r.NetClient.CreatePort(portStatus.IfaceID, &portStatus.Port)
			if err != nil {
				logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
				return err
			}
			portStatus.Port = *ConvertPort(port)
			portStatus.PortID = port.ID
		} else {
			// 单子网,自动分配IP时,从port池中取
			key := types.NamespacedName{
				Namespace: obj.Namespace,
				Name:      portStatus.FixedIPs[0].SubnetID,
			}
			vifpool := &networkv1.VifPool{}
			err = r.Get(ctx, key, vifpool)
			if err != nil {
				if client.IgnoreNotFound(err) == nil {
					err = r.createVifPoolCR(ctx, obj.Namespace, portStatus.ProjectID, portStatus.NetworkID, portStatus.FixedIPs[0].SubnetID)
					if err != nil {
						logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
						return err
					}
				}
				logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
				return err
			}
			// logger.Info(msg, "vifpool.Ports", len(vifpool.Status.Ports))
			for {
				if len(vifpool.Status.Ports) > 0 {
					portStatus.Port = *vifpool.Status.Ports[0]
					portStatus.PortID = portStatus.ID
					vifpool.Status.Ports = vifpool.Status.Ports[1:]
					err = r.Status().Update(ctx, vifpool)
					if err != nil {
						if client.IgnoreNotFound(err) == nil {
							logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
							return err
						}
						time.Sleep(time.Millisecond * 10)
						err = r.Get(ctx, key, vifpool)
						if err != nil {
							logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
							return err
						}
						continue
					}
				}
				break
			}
		}
		if len(portStatus.FixedIPs) > 0 {
			// 根据subnetId查询netId,cidr和gateway,补充信息
			for i := 0; i < len(portStatus.FixedIPs); i++ {
				if portStatus.FixedIPs[i].Gateway != "" && portStatus.FixedIPs[i].Cidr != "" {
					continue
				}
				var err error
				var subnet *subnets.Subnet
				if sub, ok := subnetMap[portStatus.FixedIPs[i].SubnetID]; ok {
					subnet = sub
				} else {
					subnet, err = r.NetClient.GetSubnet(portStatus.FixedIPs[i].SubnetID)
					if err != nil {
						logger.Info(msg, "error", err, "FileLine", wooshtools.FileLine())
						return err
					}
				}
				portStatus.FixedIPs[i].Gateway = subnet.GatewayIP
				portStatus.FixedIPs[i].Cidr = subnet.CIDR
			}
		}
		if portStatus.ID != "" {
			portStatus.PortReady = true
			if portStatus.IfName == "" {
				portStatus.IfName = WooshNetIFNamePrefix + strconv.Itoa(index)
			}
			portStatus.IfaceID = "c_" + portStatus.ID[:12]
			portStatus.NicName = "c_" + portStatus.ID[:12]
			portStatus.Interface = "c_" + portStatus.ID[:12]
		}
	}
	obj.Status.PortReady = true
	for _, status := range obj.Status.PortStatus {
		if status.ID == "" {
			obj.Status.PortReady = false
			break
		}
	}
	logger.Info(msg, "PortReady", obj.Status.PortReady)
	if r.needUpdate(oldobj, obj) {
		err := r.Status().Update(context.TODO(), obj)
		if err != nil {
			logger.Info(msg, "error", err.Error())
			return err
		}
		if obj.Status.PortReady && obj.Status.PodName != "" {
			pod, err := r.getPod(obj.Namespace, obj.Status.PodName)
			if err != nil {
				logger.Info(msg, "error", err.Error())
				obj.Status.Message = err.Error()
			} else {
				if pod.Annotations == nil {
					pod.Annotations = make(map[string]string)
				}
				statusAnno, err := json.Marshal(obj.Status.PortStatus)
				if err != nil {
					logger.Error(err, msg)
					obj.Status.Message = err.Error()
				}
				pod.Annotations[WooshPortStatus] = string(statusAnno)
				// pod.Annotations[NetworkStatusAnnot] = string(statusAnno)
				// pod.Annotations[OldNetworkStatusAnnot] = string(statusAnno)
				err = r.Update(context.TODO(), pod)
				if err != nil {
					logger.Error(err, msg)
					obj.Status.Message = err.Error()
				}
			}
		}
	}
	if !obj.Status.PortReady {
		return fmt.Errorf("Port not ready (%s)", msg)
	}
	return nil
}

func (r *WooshPortReconciler) deleteCR(ctx context.Context, obj *networkv1.WooshPort) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	oldnd := obj.DeepCopy()
	if !obj.Spec.Deleted {
		logger.Info(msg, "event", "delete CR")
	} else {
		logger.Info(msg, "event", "delete")
		var err error
		if obj.Status.DeviceReady {
			var err error
			if obj.Status.DeviceReady && obj.Status.NodeName != "" {
				_, err = r.getNode(obj.Status.NodeName)
			}
			if obj.Status.NodeName == "" || client.IgnoreNotFound(err) == nil {
				obj.Status.DeviceReady = false
			}
		} else {
			if obj.Status.PortReady {
				for index := range obj.Status.PortStatus {
					portStatus := &obj.Status.PortStatus[index]
					// 删除Port
					if len(portStatus.FixedIPs) > 1 || portStatus.QoSPolicyID != "" || len(portStatus.SecurityGroups) > 0 || portStatus.FixedIPs[0].IPAddress != "" {
						// 多子网,或者指定IP时,调接口直接删除port
						err = r.NetClient.DeletePort(portStatus.PortID)
						if err != nil {
							logger.Info(msg, "error", err.Error())
							return err
						}
						portStatus.PortID = ""
					} else {
						// 单子网,自动分配IP时,回收到port池中
						key := types.NamespacedName{
							Namespace: obj.Namespace,
							Name:      portStatus.FixedIPs[0].SubnetID,
						}
						vifpool := &networkv1.VifPool{}
						err = r.Get(ctx, key, vifpool)
						if err != nil {
							logger.Info(msg, "error", err.Error())
							return err
						}
						for {
							vifpool.Status.Ports = append(vifpool.Status.Ports, portStatus.Port.DeepCopy())
							err = r.Status().Update(ctx, vifpool)
							if err != nil {
								if client.IgnoreNotFound(err) == nil {
									return err
								}
								err = r.Get(ctx, key, vifpool)
								if err != nil {
									logger.Error(err, msg)
									return err
								}
								continue
							}
							break
						}
					}
					portStatus.PortReady = false
				}
				obj.Status.PortReady = false
			}
		}

		if !obj.Status.Ready && !obj.Status.PortReady && !obj.Status.DeviceReady {
			err := r.Delete(context.TODO(), obj)
			if err != nil {
				logger.Info(msg, "error", err.Error())
				return err
			}
		} else if r.needUpdate(oldnd, obj) {
			err := r.Status().Update(context.TODO(), obj)
			if err != nil {
				logger.Info(msg, "error", err.Error())
				return err
			}
		}
	}
	return nil
}

func (r *WooshPortReconciler) handlerUpdateCR(ctx context.Context, obj *networkv1.WooshPort) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	if obj.Spec.Deleted {
		return nil
	}
	if !obj.Status.PortReady {
		return nil
	}
	oldobj := obj.DeepCopy()
	// 开始创建ovs port和虚拟网卡
	var err error
	for index := range obj.Status.PortStatus {
		portStatus := &obj.Status.PortStatus[index]
		portStatus.DeviceReady = false
		if portStatus.IfaceID != "" {
			_, err = r.DeviceClient.CreatePort(portStatus.IfaceID, portStatus)
			if err != nil {
				logger.Info(msg, "error", err.Error())
				return err
			}
			portStatus.DeviceReady = true
		}
	}
	if err != nil {
		obj.Status.Message = err.Error()
		logger.Error(err, msg)
	}
	obj.Status.DeviceReady = true
	for _, status := range obj.Status.PortStatus {
		obj.Status.DeviceReady = true && status.DeviceReady
	}
	if r.needUpdate(oldobj, obj) {
		err := r.Status().Update(ctx, obj)
		if err != nil {
			logger.Error(err, msg)
			return err
		}
	}
	if !obj.Status.DeviceReady {
		return fmt.Errorf("Device not ready (%s)", msg)
	}
	if obj.Status.PodName == "" {
		return nil
	}
	if !obj.Status.PodReady {
		// 获取netns, 并移入netns
		podSandbox, err := cri.GetCriPodByName(obj.Namespace, obj.Status.PodName)
		if err != nil {
			logger.Info(msg, "error", err)
			obj.Status.PodReady = false
		} else {
			info, err := cri.GetSandboxInfo(podSandbox.Id)
			if err != nil {
				logger.Info(msg, "error", err, "id", podSandbox.Id)
				return err
			}
			logger.Info(msg, "runtimeType", info.RuntimeType)
			logger.Info(msg, "pid", info.Pid)
			// logger.Info(msg, "ns", info.RuntimeSpec.Linux.Namespaces)
			var netns string
			for _, ns := range info.RuntimeSpec.Linux.Namespaces {
				if ns.Type != specs.NetworkNamespace {
					continue
				}
				netns = ns.Path
				break
			}
			logger.Info(msg, "netns", netns)
			obj.Status.PodPid = info.Pid
			obj.Status.PodRuntimeType = info.RuntimeType
			obj.Status.PodNetns = netns
			obj.Status.PodReady = true
		}
		if r.needUpdate(oldobj, obj) {
			err := r.Status().Update(context.TODO(), obj)
			if err != nil {
				logger.Info(msg, "error", err)
				return err
			}
		}
		if !obj.Status.PodReady {
			return fmt.Errorf("Pod not ready (%s)", msg)
		}
	}

	return nil
}

func (r *WooshPortReconciler) handlerDeleteCR(ctx context.Context, obj *networkv1.WooshPort) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	oldobj := obj.DeepCopy()
	if !obj.Spec.Deleted {
		logger.Info(msg, "event", "delete CR")
	} else {
		logger.Info(msg, "event", "delete")
		var err error
		if obj.Status.PodReady {
			if obj.Status.PodNetns != "" {
				curns, err := ns.GetCurrentNS()
				if err != nil {
					logger.Error(err, msg, "error", "GetCurrentNS failed")
					return err
				}
				defer curns.Close()
				podns, err := ns.GetNS(obj.Status.PodNetns)
				if err != nil {
					logger.Error(err, msg, "error", fmt.Errorf("GetNS %s failed", obj.Status.PodNetns))
					return err
				}
				defer podns.Close()
				for _, portStatus := range obj.Status.PortStatus {
					err = podns.Do(func(_ ns.NetNS) error {
						link, err := netlink.LinkByName(portStatus.NicName)
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
						logger.Error(err, msg, "interface", portStatus.Interface)
					}
				}
			}
			obj.Status.PodReady = false
		} else {
			if !obj.Status.DeviceReady {
				return nil
			}
			for index := range obj.Status.PortStatus {
				portStatus := &obj.Status.PortStatus[index]
				err = r.DeviceClient.DeletePort(portStatus.IfaceID)
				if err != nil {
					logger.Error(err, msg, "IfaceID", portStatus.IfaceID)
				}
				portStatus.DeviceReady = false
			}
			obj.Status.DeviceReady = false
		}
		if r.needUpdate(oldobj, obj) {
			err := r.Status().Update(context.TODO(), obj)
			if err != nil {
				logger.Error(err, msg)
				return err
			}
		}
	}
	return nil
}

func (r *WooshPortReconciler) getPod(namespace, name string) (*v1.Pod, error) {
	nsd := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var pod v1.Pod
	err := r.Get(context.Background(), nsd, &pod)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *WooshPortReconciler) getNode(name string) (*v1.Node, error) {
	nsd := types.NamespacedName{
		Name: name,
	}
	var node v1.Node
	err := r.Get(context.Background(), nsd, &node)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// createWooshPortCR
func (r *WooshPortReconciler) createVifPoolCR(ctx context.Context, namespace, projectId, netId, subnetId string) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	vifPool := networkv1.VifPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subnetId,
			Namespace: namespace,
		},
		Spec: networkv1.VifPoolSpec{
			ProjectID: projectId,
			NetID:     netId,
			SubnetID:  subnetId,
		},
	}
	err := r.Create(context.TODO(), &vifPool)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		logger.Error(err, namespace+"/"+subnetId)
		return err
	}

	return nil
}
