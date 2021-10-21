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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	networkv1 "wooshnet/apis/network/v1"
	"wooshnet/pkg/netclient"
	"wooshnet/pkg/wooshtools"
)

// VifPoolReconciler reconciles a VifPool object
type VifPoolReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	NetClient netclient.NetClient
}

//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=vifpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=vifpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.wooshnet.cn,resources=vifpools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VifPool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *VifPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	logger.Info(req.NamespacedName.String())

	var result ctrl.Result
	var err error
	obj := &networkv1.VifPool{}
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
		if obj.DeletionTimestamp.IsZero() && !obj.Spec.Deleted {
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
func (r *VifPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1.VifPool{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			obj := object.(*networkv1.VifPool)
			return obj.Spec.SubnetID != ""
		})).
		Complete(r)
}

// matched 检查是否需要处理
func (r *VifPoolReconciler) matched(obj *networkv1.VifPool) bool {

	return true
}

func ConvertPort(port interface{}) *networkv1.Port {
	buf, _ := json.Marshal(port)
	result := &networkv1.Port{}
	_ = json.Unmarshal(buf, result)
	return result
}

func (r *VifPoolReconciler) createOrUpdateCR(ctx context.Context, obj *networkv1.VifPool) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	if obj.Spec.NetID == "" || obj.Spec.ProjectID == "" {
		subnet, err := r.NetClient.GetSubnet(obj.Spec.SubnetID)
		if err != nil {
			logger.Error(err, msg, "FileLine", wooshtools.FileLine())
			return err
		}
		obj.Spec.NetID = subnet.NetworkID
		obj.Spec.ProjectID = subnet.ProjectID
		err = r.Update(context.Background(), obj)
		if err != nil {
			return err
		}
		return nil
	}
	if obj.Spec.Min > 0 {
		obj.Status.Min = obj.Spec.Min
	}
	if obj.Status.Min <= 0 {
		obj.Status.Min = 1
	}
	if len(obj.Status.Ports) < obj.Status.Min {
		// 创建port,并追加到obj.Status.Ports中
		port := &networkv1.Port{
			ProjectID: obj.Spec.ProjectID,
			NetworkID: obj.Spec.NetID,
			FixedIPs: []networkv1.IP{
				{
					SubnetID: obj.Spec.SubnetID,
				},
			},
		}
		port, err := r.NetClient.CreatePort("", port)
		if err != nil {
			logger.Error(err, msg)
			return err
		}
		obj.Status.Ports = append(obj.Status.Ports, port)
		err = r.Status().Update(context.Background(), obj)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				return err
			}
			key := types.NamespacedName{
				Namespace: obj.Namespace,
				Name:      obj.Name,
			}
			err = r.Get(context.Background(), key, obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *VifPoolReconciler) deleteCR(ctx context.Context, obj *networkv1.VifPool) error {
	logger := log.FromContext(ctx).WithName(wooshtools.FuncName())
	msg := obj.Namespace + "/" + obj.Name
	logger.Info(msg)
	if obj.Spec.Max > 0 {
		obj.Status.Max = obj.Spec.Max
	}
	if obj.Status.Max > 0 && len(obj.Status.Ports) > obj.Status.Max {
		// 释放多余port,并从obj.Status.Ports中删除
	}
	return nil
}
