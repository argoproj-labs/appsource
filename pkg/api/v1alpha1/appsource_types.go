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
	"reflect"
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

type AppSourceReason = string

const (
	// ApplicationCreationError indicates an unknown controller error
	ApplicationCreationError AppSourceReason = "ApplicationCreationError"
	// ApplicationCreationSuccess indicates that the controller was able to create the ArgoCD Application
	ApplicationCreationSuccess AppSourceReason = "ApplicationCreationSuccess"
	// ApplicationDeletionError indicates that controller failed to delete application
	ApplicationDeletionError AppSourceReason = "ApplicationDeletionError"
	// ApplicationInvalidSpecError indicates that application source is invalid
	ApplicationInvalidSpecError AppSourceReason = "InvalidSpecError"
	// ApplicationUnknownError indicates an unknown controller error
	ApplicationUnknownError AppSourceReason = "UnknownError"
)

type ConditionStatus = string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type AppSourceConditionType = string

const (
	Ready AppSourceConditionType = "Steady"
)

// AppSourceCondition holds the latest information about the AppSource conditions
//TODO Test this new interface with kubectl wait
type AppSourceCondition struct {
	// Last time we probed the condition.
	LastProbeTime metav1.Time `json:"lastProbeTime"`
	// Last time the condition transitioned from one status to another.
	LastTansitionTime metav1.Time `json:"lastTransitionTime"`
	// Human-readable message indicating details about last transition.
	Message AppConditionMessage `json:"message"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	Reason AppSourceReason `json:"reason"`
	// Status is the status of the condition. Can be True, False, Unknown.
	Status ConditionStatus `json:"status"`
	// Type is an application condition type
	Type AppSourceConditionType `json:"type"`
}

// AppSourceStatus defines the observed state of AppSource
type AppSourceStatus struct {
	// Conditions is a list of observed AppSource conditions
	Conditions []AppSourceCondition `json:"conditions,omitempty"`
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

func ConditionIsEqual(a, b AppSourceCondition) bool {
	aValue := reflect.ValueOf(a)
	aValues := make([]interface{}, aValue.NumField())
	bValue := reflect.ValueOf(b)
	bValues := make([]interface{}, bValue.NumField())

	if aValue.NumField() != bValue.NumField() {
		return false
	}
	for i := 0; i < bValue.NumField(); i++ {
		if aValues[i] != bValues[i] {
			return false
		}
	}
	return true

}

func IsEqual(a, b []AppSourceCondition) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if !ConditionIsEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (a *AppSource) UpsertConditions(newCondition AppSourceCondition) {
	for i, _ := range a.Status.Conditions {
		if a.Status.Conditions[i].Type == newCondition.Type {
			// Update condition
			a.Status.Conditions[i] = newCondition
			return
		}
	}
	// Condition not found, insert it
	newCondition.LastTansitionTime = metav1.Now()
	a.Status.Conditions = append(a.Status.Conditions, newCondition)
}

func init() {
	SchemeBuilder.Register(&AppSource{}, &AppSourceList{})
}
