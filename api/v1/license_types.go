/*
Copyright 2024.

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

// LicenseSpec defines the desired state of License
type LicenseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of available licenses by this name
	LicenseCount int `json:"license_count,omitempty"`
	// InUseLicenseCount is used by tools to query the license server.
	// We will add the LicenseStatus.Available to this and subtract it from LicenseCount to get the RealAvailable Licenses.
	InUseLicenseCount int `json:"inuse_license_count,omitempty"`
}

// LicenseStatus defines the observed state of License
type LicenseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The number of licenses we believe are still available
	Available int `json:"available,omitempty"`
	// The number of licenses we believe are available after accounting for External InUseLicenseCount
	RealAvailable int `json:"real_available,omitempty"`
	// The list of pods that are using licenses, and how many, in the format ns::name::count
	UsedBy  []string `json:"used_by,omitempty"`
	Queuing []string `json:"queuing,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// License is the Schema for the licenses API
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec,omitempty"`
	Status LicenseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LicenseList contains a list of License
type LicenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []License `json:"items"`
}

func init() {
	SchemeBuilder.Register(&License{}, &LicenseList{})
}
