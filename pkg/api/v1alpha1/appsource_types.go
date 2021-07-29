/*
Copyright 2021.

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

package v1alpha1

import (
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//In-cluster server address
	ClusterServerName = "https://kubernetes.default.svc"
	//ArgoCD namespace
	ArgocdNamespace = "argocd"
)

type AppConditionMessage = string

const (
	ApplicationExistsMsg   AppConditionMessage = "ArgoCD Application exists"
	ApplicationDeletionMsg AppConditionMessage = "ArgoCD Application was succesfully deleted"
	ApplicationCreationMsg AppConditionMessage = "ArgoCD Application was successfully created"
)

type AppSourceConditionType = string

const (
	// ApplicationCreationError indicates an unknown controller error
	ApplicationCreationError AppSourceConditionType = "ApplicationCreationError"
	// ApplicationCreationSuccess indicates that the controller was able to create the ArgoCD Application
	ApplicationCreationSuccess AppSourceConditionType = "ApplicationCreationSuccess"
	// ApplicationDeletionError indicates that controller failed to delete application
	ApplicationDeletionError AppSourceConditionType = "ApplicationDeletionError"
	// ApplicationDeletionSuccess indicates that the controller was able to delete the ArgoCD Application
	ApplicationDeletionSuccess AppSourceConditionType = "ApplicationDeletionSuccess"
	// ApplicationInvalidSpecError indicates that application source is invalid
	ApplicationInvalidSpecError AppSourceConditionType = "InvalidSpecError"
	// ApplicationUnknownError indicates an unknown controller error
	ApplicationUnknownError AppSourceConditionType = "UnknownError"
)

type ConditionStatus = string

const (
	ConditionTrue  = "True"
	ConditionFalse = "False"
)

// AppSourceCondition holds the latest information about the AppSource conditions
type AppSourceCondition struct {
	// Type is an application condition type
	Type AppSourceConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Boolean status describing if the conditon is currently true
	Status ConditionStatus `json:"status,string"`
	// Message contains human-readable message indicating details about condition
	Message string `json:"message" protobuf:"bytes,2,opt,name=message"`
	// LastTransitionTime is the time the condition was last observed
	ObservedAt metav1.Time `json:"observedAt,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
}

// AppSourceStatus defines the observed state of AppSource
type AppSourceStatus struct {
	// History is a list of observed AppSource conditions
	History []AppSourceCondition `json:"history,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AppSource is the Schema for the appsources API
type AppSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   argocd.ApplicationSource `json:"spec,omitempty"`
	Status AppSourceStatus          `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AppSourceList contains a list of AppSource
type AppSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppSource{}, &AppSourceList{})
}
