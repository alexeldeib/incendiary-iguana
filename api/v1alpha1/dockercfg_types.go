/*
Copyright 2019 Alexander Eldeib.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//  DockerConfigSpec defines the desired state of DockerConfig
type DockerConfigSpec struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Vault    string `json:"vault"`
	Email    string `json:"email"`
	Server   string `json:"server"`
}

//  DockerConfigStatus defines the observed state of DockerConfig
type DockerConfigStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:singular=dockerconfig,path=dockerconfigs,shortName=dockercfg,categories=all

//  DockerConfig is the Schema for the docker config API
type DockerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DockerConfigSpec   `json:"spec,omitempty"`
	Status DockerConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

//  DockerConfigList contains a list of  DockerConfig
type DockerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DockerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DockerConfig{}, &DockerConfigList{})
}
