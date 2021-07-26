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

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/project"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

type ApplicationClient struct {
	Client applicationTypes.ApplicationServiceClient
	Closer io.Closer
}

type ProjectClient struct {
	Client projectTypes.ProjectServiceClient
	Closer io.Closer
}

type ArgoCDClients struct {
	Projects     ProjectClient
	Applications ApplicationClient
}

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// AppSource ConfigMap
	ConfigMap *v1.ConfigMap
	// ArgoCD Resource Clients
	Clients ArgoCDClients
	// ArgoCD Project Template
	ProjectProfiles []map[string]*ProjectTemplate
	// Server Address
	ClusterHost string
	// ArgoCD Namespace
	ArgocdNS string
}

// Reconcile v1.0: Called upon AppSource creation, handles namespace validation and Project/App creation
func (r *AppSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Get the requested AppSource
	var appSource appsource.AppSource = appsource.AppSource{}
	if err := r.Get(ctx, req.NamespacedName, &appSource); err != nil {
		//Ignore not-found errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	appSource.Status.ReconciledAt = metav1.Now()

	if ok, err := r.UpsertAppSourceConfig(); err != nil {
		if ok {
			return ctrl.Result{Requeue: true}, errors.New("appsource configmap not created yet")
		}
		return ctrl.Result{}, err
	} else {
		defer r.Clients.Projects.Closer.Close()
		defer r.Clients.Applications.Closer.Close()
	}

	if !appSource.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.ResolveFinalizers(ctx, &appSource); err != nil {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}
	}

	// Create the Application if necessary
	proj, err := r.FindProject(req.Namespace)
	if err != nil {
		if ok := r.SetCondition(ctx, &appSource, &appsource.AppSourceCondition{
			Type:    appsource.ApplicationConditionInvalidSpecError,
			Message: err.Error(),
		}); ok != nil {
			return ctrl.Result{}, ok
		}
		return ctrl.Result{}, err
	}

	err = r.validateProject(ctx, &appSource, proj)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.validateApplication(ctx, &appSource, proj)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsource.AppSource{}).
		Complete(r)
}
