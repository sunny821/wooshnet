package cri

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containerd/containerd/pkg/cri/server"
	gocni "github.com/containerd/go-cni"
	"google.golang.org/grpc"

	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	// criv1 "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

const (
	tokenFile            = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	rootCAFile           = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	DefautContainerdSock = "unix:///run/containerd/containerd.sock"
)

var criclient criv1.RuntimeServiceClient
var sockAddr string

func GetCriClient(target string) (criv1.RuntimeServiceClient, error) {
	if criclient != nil {
		return criclient, nil
	}

	if len(target) == 0 {
		target = DefautContainerdSock
	}

	sockAddr = target
	gc, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := criv1.NewRuntimeServiceClient(gc)
	criclient = client
	return criclient, nil
}

func GetCniConfig() (*gocni.ConfigResult, error) {
	status, err := GetCriStatus()
	if err != nil {
		return nil, err
	}
	var cniConfig gocni.ConfigResult
	err = json.Unmarshal([]byte(status.Info["cniconfig"]), &cniConfig)
	if err != nil {
		return nil, err
	}

	return &cniConfig, nil
}

func GetCriStatus() (*criv1.StatusResponse, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	req := criv1.StatusRequest{Verbose: true}
	status, err := client.Status(context.Background(), &req, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}

	return status, nil
}

func GetSandboxInfo(id string) (*server.SandboxInfo, error) {
	status, err := GetCriPodStatus(id)
	if err != nil {
		return nil, err
	}
	var sandboxInfo server.SandboxInfo
	err = json.Unmarshal([]byte(status.Info["info"]), &sandboxInfo)
	if err != nil {
		return nil, err
	}

	return &sandboxInfo, nil
}

func GetCriPodStatus(id string) (*criv1.PodSandboxStatusResponse, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	req := criv1.PodSandboxStatusRequest{PodSandboxId: id, Verbose: true}
	pod, err := client.PodSandboxStatus(context.Background(), &req, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func GetCriPodByName(namespace, podname string) (*criv1.PodSandbox, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	filter := criv1.PodSandboxFilter{}
	filter.LabelSelector = make(map[string]string)
	filter.LabelSelector["io.kubernetes.pod.namespace"] = namespace
	filter.LabelSelector["io.kubernetes.pod.name"] = podname
	req := criv1.ListPodSandboxRequest{Filter: &filter}
	pods, err := client.ListPodSandbox(context.Background(), &req, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	var podSandbox *criv1.PodSandbox
	for _, pod := range pods.Items {
		if pod == nil {
			continue
		}
		if pod.State == criv1.PodSandboxState_SANDBOX_READY {
			return pod, nil
		} else {
			podSandbox = pod
		}
	}
	if podSandbox != nil {
		return podSandbox, nil
	}

	podlist, err := ListCriPod()
	if err != nil {
		klog.Errorln(err)
		return nil, err
	}
	for _, pod := range podlist.Items {
		if pod.Metadata.Namespace == namespace && pod.Metadata.Name == podname {
			return pod, nil
		}
	}

	return nil, fmt.Errorf("pod not found, %v/%v", namespace, podname)
}

func GetCriPodByLabels(namespace string, labels map[string]string) (*criv1.PodSandbox, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	filter := criv1.PodSandboxFilter{}
	filter.LabelSelector = make(map[string]string)
	filter.LabelSelector["io.kubernetes.pod.namespace"] = namespace
	for k, v := range labels {
		filter.LabelSelector[k] = v
	}
	filter.State = &criv1.PodSandboxStateValue{State: criv1.PodSandboxState_SANDBOX_READY}
	req := criv1.ListPodSandboxRequest{Filter: &filter}
	pods, err := client.ListPodSandbox(context.Background(), &req, grpc.EmptyCallOption{})
	if err != nil {
		klog.Errorln(err)
		return nil, err
	}
	for _, pod := range pods.Items {
		if pod.State == criv1.PodSandboxState_SANDBOX_READY {
			return pod, nil
		}
	}

	return nil, fmt.Errorf("pod not found, %v %v", namespace, labels)
}

func ListCriPodByNamespace(namespace string) (*criv1.ListPodSandboxResponse, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	filter := criv1.PodSandboxFilter{}
	filter.LabelSelector = make(map[string]string)
	filter.LabelSelector["io.kubernetes.pod.namespace"] = namespace
	filter.State = &criv1.PodSandboxStateValue{State: criv1.PodSandboxState_SANDBOX_READY}
	req := criv1.ListPodSandboxRequest{Filter: &filter}

	return client.ListPodSandbox(context.Background(), &req, grpc.EmptyCallOption{})
}

func ListCriPod() (*criv1.ListPodSandboxResponse, error) {
	client, err := GetCriClient(sockAddr)
	if err != nil {
		return nil, err
	}

	filter := criv1.PodSandboxFilter{}
	filter.LabelSelector = make(map[string]string)
	filter.State = &criv1.PodSandboxStateValue{State: criv1.PodSandboxState_SANDBOX_READY}
	req := criv1.ListPodSandboxRequest{Filter: &filter}

	return client.ListPodSandbox(context.Background(), &req, grpc.EmptyCallOption{})
}
