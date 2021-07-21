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

package controllers

import (
	"context"
	"errors"
	"io"
	"regexp"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/project"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	argocdClientSet "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

type Compilers struct {
	Pattern *regexp.Regexp
}

type ProjectTemplate struct {
	NamePattern string                 `json:"namePattern"`
	Spec        *argocd.AppProjectSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// ArgoCD Project Client
	ArgoProjectClient projectTypes.ProjectServiceClient
	// ArgoCD Project Client Closer
	ArgoProjectClientCloser io.Closer
	// ArgoCD Application Client
	ArgoApplicationClient applicationTypes.ApplicationServiceClient
	// ArgoCD Application Client Closer
	ArgoApplicationClientCloser io.Closer
	// ArgoCD Project Template
	Project ProjectTemplate
	// Regex Compilers
	Compilers Compilers
	// Server Address
	ClusterHost string
	// ArgoCD Namespace
	ArgocdNS string
}

// GetCompilers returns all Regex compilers described by regex strings
// found in the appsource configuration
func GetCompilers(template ProjectTemplate) (C Compilers) {
	if template.NamePattern != "" {
		C.Pattern = regexp.MustCompile(template.NamePattern)
	}
	return C
}

// SetupConfigMap tries to initialize ArgoCD Clients using the AppSource ConfigMap
func (r *AppSourceReconciler) SetupConfigMap() error {
	if (r.ArgoApplicationClient == nil) || (r.ArgoProjectClient == nil) {
		appsourceConfigMap, err := GetAppSourceConfigmap()
		if err == nil {
			appsourceProjectTemplate := ProjectTemplate{}
			err = yaml.Unmarshal([]byte(appsourceConfigMap.Data["project.template"]), &appsourceProjectTemplate)
			if err != nil {
				return err
			}
			argocdClientOpts, err := GetClientOpts(*appsourceConfigMap)
			if err != nil {
				return err
			}
			argocdClient, err := argocdClientSet.NewClient(argocdClientOpts)
			if err != nil {
				return err
			}

			r.Project = appsourceProjectTemplate
			r.ArgoApplicationClientCloser, r.ArgoApplicationClient = argocdClient.NewApplicationClientOrDie()
			r.ArgoProjectClientCloser, r.ArgoProjectClient = argocdClient.NewProjectClientOrDie()
			r.Compilers = GetCompilers(r.Project)
			return nil
		}
	}
	return nil
}

//+kubebuilder:rbac:groups=argoproj.io,resources=appsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=appsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=argoproj.io,resources=appsources/finalizers,verbs=update

// Reconcile v1.0: Called upon AppSource creation, handles namespace validation and Project/App creation
func (r *AppSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if err := r.SetupConfigMap(); err != nil {
		return ctrl.Result{}, err
	}

	// If the clients were not set using the configmap wait until it is set
	if (r.ArgoApplicationClient == nil) || (r.ArgoProjectClient == nil) {
		return ctrl.Result{Requeue: true}, nil
	}

	var appSource argoprojv1alpha1.AppSource = argoprojv1alpha1.AppSource{}
	if err := r.Get(ctx, req.NamespacedName, &appSource); err != nil {
		//Ignore not-found errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !appSource.ObjectMeta.DeletionTimestamp.IsZero() {
		//Returns nil if nothing went wrong, non-nil err if encountered problem
		return ctrl.Result{}, r.ResolveFinalizers(ctx, &appSource)
	}

	// Create the Application if necessary
	patternMatchesNamespace := r.Compilers.Pattern.Match([]byte(req.Namespace))
	if patternMatchesNamespace {
		err := r.validateProject(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.validateApplication(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		//Name does not match namespace regex pattern.
		return ctrl.Result{}, errors.New("namespace does not match AppSource project namePattern")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojv1alpha1.AppSource{}).
		Complete(r)
}
