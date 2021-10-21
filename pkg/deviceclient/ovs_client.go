package deviceclient

import (
	corev1 "k8s.io/api/core/v1"
)

type OvsClient struct {
	Bridge string
}

func NewOvsClient(config *corev1.ConfigMap) (*OvsClient, error) {
	return &OvsClient{Bridge: config.Data["BRIDGE_NAME"]}, nil
}

func (c *OvsClient) Type() string {
	return "ovs"
}
