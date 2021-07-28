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
	ApplicationCreationSuccesfulMsg  AppConditionMessage = "ArgoCD Application was successfully created"
	ApplicationExistsMsg             AppConditionMessage = "ArgoCD Application exists"
	ApplicationCreationSuccessfulMsg AppConditionMessage = "ArgoCD Application was succesfully deleted"
)

type AppSourceConditionType = string

const (
	// ApplicationConditionCreationErro indicates an unknown controller error
	ApplicationConditionCreationError AppSourceConditionType = "ApplicationCreationError"
	// ApplicationConditionCreationSuccessful indicates that the controller was able to create the ArgoCD Application
	ApplicationConditionCreationSuccessful AppSourceConditionType = "ApplicationCreationSuccesful"
	// ApplicationConditionDeletionError indicates that controller failed to delete application
	ApplicationConditionDeletionError AppSourceConditionType = "ApplicationDeletionError"
	// ApplicationConditionDeletionSuccessful indicates that the controller was able to delete the ArgoCD Application
	ApplicationConditionDeletionSuccessful AppSourceConditionType = "ApplicationDeletionSuccessful"
	// ApplicationConditionInvalidSpecError indicates that application source is invalid
	ApplicationConditionInvalidSpecError AppSourceConditionType = "InvalidSpecError"
	// ApplicationConditionUnknownError indicates an unknown controller error
	ApplicationConditionUnknownError AppSourceConditionType = "UnknownError"
	// ProjectCondtionCreationError indicates the controller was unable to create the ArgoCD Project
	ProjectConditonCreationError AppSourceConditionType = "ProjectCreationError"
	// ApplicationExists indicates that the controller found the application referenced by the AppSource Spec
	ApplicationExists AppSourceConditionType = "ApplicationExists"
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
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
}

type OperationType string

const (
	ArgoCDAppCreation     OperationType = "Creating ArgoCD Application"
	ArgoCDProjectCreation OperationType = "Creating ArgoCD Project"
	ArgoCDAppDeletion     OperationType = "Deleting ArgoCD Application"
)

type OperationPhase string

const (
	OperationRunning   OperationPhase = "Running"
	OperationError     OperationPhase = "Error"
	OperationSucceeded OperationPhase = "Succeeded"
)

// AppSourceOperation indicates the current ongoing AppSource operation
type Operation struct {
	Type  OperationType  `json:"appSourceOperationType,omitempty"`
	Phase OperationPhase `json:"appSourcePhase,omitempty"`
	// StartedAt contains time of operation start
	StartedAt *metav1.Time `json:"startedAt" protobuf:"bytes,6,opt,name=startedAt"`
	// FinishedAt contains time of operation completion
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,7,opt,name=finishedAt"`
	// RetryCount contains time of operation retries
	RetryCount int64 `json:"retryCount,omitempty" protobuf:"bytes,8,opt,name=retryCount"`
}

// AppSourceStatus defines the observed state of AppSource
type AppSourceStatus struct {
	// OperationState contains information about any ongoing operations, such as a ApplicationCreation
	Operation Operation `json:"operationState,omitempty"`
	// Conditions is the condition of the AppSource instance
	Condition *AppSourceCondition `json:"condition,omitempty"`
	// ReconciledAt indicates when the appsource instance was last reconciled
	ReconciledAt metav1.Time `json:"reconciledAt,omitempty"`
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
