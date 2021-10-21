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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WooshPortSpec defines the desired state of WooshPort
type WooshPortSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NodeName    string `json:"nodeName,omitempty"`
	PodName     string `json:"podName,omitempty"`
	Ports       []Port `json:"ports,omitempty"`
	AutoCreated bool   `json:"autoCreated,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
}

// WooshPortStatus defines the observed state of WooshPort
type WooshPortStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	NodeName       string       `json:"nodeName,omitempty"`
	NodeIP         string       `json:"nodeIP,omitempty"`
	PortReady      bool         `json:"portReady"`
	DeviceReady    bool         `json:"deviceReady"`
	PodName        string       `json:"podName,omitempty"`
	PodPid         uint32       `json:"podPid,omitempty"`
	PodNetns       string       `json:"podNetns,omitempty"`
	PodRuntimeType string       `json:"podRuntimeType,omitempty"`
	PodReady       bool         `json:"podReady"`
	Ready          bool         `json:"ready"`
	Message        string       `json:"message,omitempty"`
	PortStatus     []PortStatus `json:"portStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vport
// +kubebuilder:printcolumn:name="nodeIP",type="string",JSONPath=".status.nodeIP",description="nodeIP"
// +kubebuilder:printcolumn:name="portReady",type="string",JSONPath=".status.portReady",description="portReady"
// +kubebuilder:printcolumn:name="deviceReady",type="string",JSONPath=".status.deviceReady",description="deviceReady"
// +kubebuilder:printcolumn:name="podReady",type="string",JSONPath=".status.podReady",description="podReady"
// +kubebuilder:printcolumn:name="ready",type="string",JSONPath=".status.ready",description="ready"
// +kubebuilder:printcolumn:name="deleted",type="boolean",JSONPath=".spec.deleted",description="deleted"

// WooshPort is the Schema for the wooshports API
type WooshPort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WooshPortSpec   `json:"spec,omitempty"`
	Status WooshPortStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WooshPortList contains a list of WooshPort
type WooshPortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WooshPort `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WooshPort{}, &WooshPortList{})
}
