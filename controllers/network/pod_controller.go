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
	"strconv"
	"strings"
	"time"
	"wooshnet/pkg/wooshtools"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1 "wooshnet/apis/network/v1"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	netdef "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	SystemNamespace string
}

//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.6/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	logger.Info(req.NamespacedName.String())

	var result ctrl.Result
	var err error
	obj := &corev1.Pod{}
	err = r.Get(context.Background(), req.NamespacedName, obj)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info(req.NamespacedName.String() + " not found")
		} else {
			result.Requeue = true
			result.RequeueAfter = time.Second
			return result, err
		}
	} else if r.matched(obj) {
		if obj.DeletionTimestamp.IsZero() {
			// DeletionTimestamp 为空时,为创建或更新事件
			err = r.createOrUpdateCR(ctx, obj)
			if err != nil {
				result.Requeue = true
				result.RequeueAfter = time.Second
				return result, err
			}
		} else {
			// DeletionTimestamp 不为空时,为删除事件
			err = r.deleteCR(ctx, obj)
			if err != nil {
				result.Requeue = true
				result.RequeueAfter = time.Second
				return result, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			pod := object.(*corev1.Pod)
			return !pod.Spec.HostNetwork
		})).
		Complete(r)
}

// matched 检查是否需要处理
func (r *PodReconciler) matched(pod *corev1.Pod) bool {

	return true
}

// createOrUpdate 检查是否需要处理
func (r *PodReconciler) createOrUpdateCR(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := pod.Namespace + "/" + pod.Name
	logger.Info(msg)
	if pod.Spec.HostNetwork {
		return nil
	}
	annotations := pod.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if annotations[WooshPortName] == "" {
		// 未指定WooshPort CR, 创建pod同名WooshPort CR
		ports, err := r.getPortsFromPodAnnotation(ctx, pod)
		if err != nil {
			return err
		}
		ok := false
		for _, port := range ports {
			if len(port.FixedIPs) > 0 {
				ok = true
				break
			}
		}
		if !ok {
			nsd := types.NamespacedName{
				Name: pod.Namespace,
			}
			var ns corev1.Namespace
			err = r.Get(context.Background(), nsd, &ns)
			if err != nil {
				return err
			}
			ports, err = r.getPortsFromNamespaceAnnotation(&ns)
			if err != nil {
				return err
			}
		}
		err = r.createWooshPortCR(ctx, pod, ports)
		if err != nil {
			logger.Error(err, pod.Namespace+"/"+pod.Name)
			return err
		}
	} else {
		// 指定WooshPort CR, 不做操作
		name := annotations[WooshPortName]
		wp, err := r.getWooshPortCR(ctx, pod.Namespace, name)
		if err != nil {
			return err
		}
		if wp.Spec.PodName != pod.Name {
			wp.Spec.PodName = pod.Name
			err = r.Update(context.TODO(), wp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// delete 检查是否需要处理
func (r *PodReconciler) deleteCR(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := pod.Namespace + "/" + pod.Name
	logger.Info(msg)
	if pod.Spec.HostNetwork {
		return nil
	}
	annos := pod.Annotations
	if annos == nil {
		annos = make(map[string]string)
	}
	if annos[WooshPortName] == "" {
		// 未指定WooshPort CR, 修改pod同名WooshPort CR的状态为delete
		err := r.deleteWooshPortCR(ctx, pod.Namespace, pod.Name)
		if err != nil {
			logger.Error(err, pod.Namespace+"/"+pod.Name)
			return err
		}
	} else {
		// 指定WooshPort CR, 不做操作
		name := annos[WooshPortName]
		wp, err := r.getWooshPortCR(ctx, pod.Namespace, name)
		if err != nil {
			return err
		}
		if wp.Spec.PodName == pod.Name {
			wp.Spec.PodName = ""
			// wp.Status.Ready = false
			// wp.Status.PodReady = false
			// wp.Status.DeviceReady = false
			err = r.Update(context.TODO(), wp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getPortsFromPodAnnotation
func (r *PodReconciler) getPortsFromPodAnnotation(ctx context.Context, pod *corev1.Pod) ([]networkv1.Port, error) {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := pod.Namespace + "/" + pod.Name
	annotations := pod.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	statusAnno := annotations[NetworkStatusAnnot]
	if statusAnno == "" {
		statusAnno = "[]"
	}
	portsAnno := annotations[WooshPortsAnnot]
	var annoPorts []networkv1.Port
	if portsAnno != "" {
		err := json.Unmarshal([]byte(portsAnno), &annoPorts)
		if err != nil {
			klog.Errorln(err)
			return nil, err
		}
	}
	logger.Info(msg, "annoPorts", annoPorts)

	var defaultType string
	var defaultPorts []networkv1.Port
	if annotations[MultusDefaultNetworkAttachmentAnnot] != "" {
		def := annotations[MultusDefaultNetworkAttachmentAnnot]
		klog.Infoln(def)
		var nad *netdef.NetworkAttachmentDefinition
		var err error
		defs := strings.Split(def, "/")
		if len(defs) == 1 {
			nad, err = r.getNetAttachDef(ctx, r.SystemNamespace, defs[0])
		} else if len(defs) == 2 {
			// 读multus cr
			nad, err = r.getNetAttachDef(ctx, defs[0], defs[1])
		}
		if err != nil {
			klog.Error(err, def)
			return nil, err
		}
		cniConfig := nad.Spec.Config
		netConf, err := LoadNetConf([]byte(cniConfig))
		if err != nil {
			klog.Error(err, def)
			return nil, err
		}
		defaultType = netConf.Type
		defaultPorts = netConf.Ports
		logger.Info(msg, "defaultType", defaultType)
		logger.Info(msg, "defaultPorts", defaultPorts)
	}
	var ports []networkv1.Port
	if defaultType == WooshNetType || defaultType == "" {
		if annotations[WooshNetSubnetID] != "" {
			port := networkv1.Port{
				Name:      WooshNetIFNamePrefix + "0",
				ProjectID: annotations[WooshNetProjectID],
				NetworkID: annotations[WooshNetNetID],
			}
			if annotations[WooshNetSecurityGroups] != "" {
				// '["aaa","bbb"]'
				var securityGroups []string
				err := json.Unmarshal([]byte(annotations[WooshNetSecurityGroups]), &securityGroups)
				if err != nil {
					klog.Errorln(err)
					return nil, err
				}
				port.SecurityGroups = securityGroups
			}
			if annotations[WooshNetSubnetID] != "" {
				port.FixedIPs = []networkv1.IP{
					{
						SubnetID:  annotations[WooshNetSubnetID],
						IPAddress: annotations[WooshNetIPAddress],
					},
				}
			}
			ports = append(ports, port)
		}
		for _, port := range defaultPorts {
			port.Name = WooshNetIFNamePrefix + "0"
			if len(ports) > 0 {
				port.Name += strconv.Itoa(len(ports))
			}
			ports = append(ports, port)
		}
		for _, port := range annoPorts {
			port.Name = WooshNetIFNamePrefix + "0"
			if len(ports) > 0 {
				port.Name += strconv.Itoa(len(ports))
			}
			ports = append(ports, port)
		}
	}
	logger.Info(msg, "ports", ports)
	// 兼容multus注解
	multusAnno := annotations[netdef.NetworkAttachmentAnnot]
	if len(multusAnno) > 0 {
		for index, def := range strings.Split(multusAnno, ",") {
			klog.Infoln(def)
			var nad *netdef.NetworkAttachmentDefinition
			var err error
			defs := strings.Split(def, "/")
			if len(defs) == 1 {
				nad, err = r.getNetAttachDef(ctx, pod.Namespace, defs[0])
			} else if len(defs) == 2 {
				// 读multus cr
				nad, err = r.getNetAttachDef(ctx, defs[0], defs[1])
			}
			if err != nil {
				klog.Error(err, def)
				continue
			}
			cniConfig := nad.Spec.Config
			netConf, err := LoadNetConf([]byte(cniConfig))
			if err != nil {
				klog.Error(err, def)
				return nil, err
			}
			if netConf.Type == WooshNetType {
				for pindex, port := range netConf.Ports {
					port.Name = MultusPrefix + strconv.Itoa(index+1)
					if pindex > 0 {
						port.Name += strconv.Itoa(pindex)
					}
					ports = append(ports, port)
				}
			}
		}
	}
	logger.Info(msg, "ports", ports)

	return ports, nil
}

// getPortsFromNamespaceAnnotation
func (r *PodReconciler) getPortsFromNamespaceAnnotation(ns *v1.Namespace) ([]networkv1.Port, error) {
	annotations := ns.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	// 未指定WooshPort CR, 创建pod同名WooshPort CR
	statusAnno := annotations[NetworkStatusAnnot]
	if statusAnno == "" {
		statusAnno = "[]"
	}
	portsAnno := annotations[WooshPortsAnnot]
	var ports []networkv1.Port
	if portsAnno != "" {
		err := json.Unmarshal([]byte(portsAnno), &ports)
		if err != nil {
			klog.Errorln(err)
			return nil, err
		}
	}
	if annotations[WooshNetSubnetID] != "" {
		ports = append(ports, networkv1.Port{
			ProjectID: annotations[WooshNetProjectID],
			NetworkID: annotations[WooshNetNetID],
			FixedIPs: []networkv1.IP{
				{
					SubnetID: annotations[WooshNetSubnetID],
				},
			},
		})
	}

	return ports, nil
}

// getWooshPortCR
func (r *PodReconciler) getWooshPortCR(ctx context.Context, namespace, name string) (*networkv1.WooshPort, error) {
	nsd := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var wp networkv1.WooshPort
	err := r.Get(context.Background(), nsd, &wp)
	if err != nil {
		return nil, err
	}

	return &wp, nil
}

// getNetAttachDef
func (r *PodReconciler) getNetAttachDef(ctx context.Context, namespace, name string) (*netdef.NetworkAttachmentDefinition, error) {
	nsd := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var nad netdef.NetworkAttachmentDefinition
	err := r.Get(context.Background(), nsd, &nad)
	if err != nil {
		return nil, err
	}

	return &nad, nil
}

// createWooshPortCR
func (r *PodReconciler) createWooshPortCR(ctx context.Context, pod *corev1.Pod, ports []networkv1.Port) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	if len(ports) == 0 {
		return nil
	}
	nsd := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	var wp networkv1.WooshPort
	err := r.Get(context.Background(), nsd, &wp)
	if err != nil && client.IgnoreNotFound(err) == nil {
		wp := networkv1.WooshPort{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			Spec: networkv1.WooshPortSpec{
				PodName:     pod.Name,
				Ports:       ports,
				AutoCreated: true,
			},
		}
		err := r.Create(context.TODO(), &wp)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			logger.Error(err, pod.Namespace+"/"+pod.Name)
			return err
		}
	}

	return nil
}

// deleteWooshPortCR
func (r *PodReconciler) deleteWooshPortCR(ctx context.Context, namespace, name string) error {
	nsd := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var wp networkv1.WooshPort
	err := r.Get(context.Background(), nsd, &wp)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}
	if wp.Spec.Deleted {
		return nil
	}
	wp.Spec.Deleted = true
	err = r.Update(context.TODO(), &wp)
	if err != nil {
		return err
	}

	return nil
}
