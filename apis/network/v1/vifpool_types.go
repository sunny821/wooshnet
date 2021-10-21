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

// VifPoolSpec defines the desired state of VifPool
type VifPoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ProjectID string `json:"projectId,omitempty"`
	NetID     string `json:"netId,omitempty"`
	SubnetID  string `json:"subnetId"`
	Min       int    `json:"min,omitempty"`
	Max       int    `json:"max,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

// VifPoolStatus defines the observed state of VifPool
type VifPoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Min   int     `json:"min"`
	Max   int     `json:"max"`
	Ports []*Port `json:"ports"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vpool
// +kubebuilder:printcolumn:name="projectId",type="string",JSONPath=".spec.projectId",description="projectId"
// +kubebuilder:printcolumn:name="netId",type="string",JSONPath=".spec.netId",description="netId"
// +kubebuilder:printcolumn:name="subnetId",type="string",JSONPath=".spec.subnetId",description="subnetId"
// +kubebuilder:printcolumn:name="min",type="string",JSONPath=".status.min",description="min"
// +kubebuilder:printcolumn:name="max",type="string",JSONPath=".status.max",description="max"

// VifPool is the Schema for the vifpools API
type VifPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VifPoolSpec   `json:"spec,omitempty"`
	Status VifPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VifPoolList contains a list of VifPool
type VifPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VifPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VifPool{}, &VifPoolList{})
}
