package netclient

import (
	corev1 "k8s.io/api/core/v1"
)

type OvnClient struct {
}

func NewOvnClient(config *corev1.ConfigMap) (*OvnClient, error) {

	return &OvnClient{}, nil
}

func (c *OvnClient) Type() string {
	return "ovn"
}
