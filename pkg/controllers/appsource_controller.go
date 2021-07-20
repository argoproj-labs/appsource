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
	"regexp"

	applicationTypes "github.com/argoproj/argo-cd/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/pkg/apiclient/project"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

type Compilers struct {
	Pattern *regexp.Regexp
}

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// ArgoCD Project Client
	ArgoProjectClient projectTypes.ProjectServiceClient
	// ArgoCD Application Client
	ArgoApplicationClient applicationTypes.ApplicationServiceClient
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

// Reconcile v1.0: Called upon AppSource creation, handles namespace validation and Project/App creation
func (r *AppSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var appSource *argoprojv1alpha1.AppSource
	if err := r.Get(ctx, req.NamespacedName, appSource); err != nil {
		//Ignore not-found errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !appSource.ObjectMeta.DeletionTimestamp.IsZero() {
		//Returns nil if nothing went wrong, non-nil err if encountered problem
		return ctrl.Result{}, r.ResolveFinalizers(ctx, appSource)
	}

	// Create the Application if necessary
	patternMatchesNamespace := r.Compilers.Pattern.Match([]byte(req.Namespace))
	if patternMatchesNamespace {
		err := r.validateProject(ctx, req)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		err = r.validateApplication(ctx, req)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	} else {
		//Name does not match namespace regex pattern.
		return ctrl.Result{Requeue: true}, errors.New("namespace does not match AppSource project namePattern")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojv1alpha1.AppSource{}).
		Complete(r)
}
