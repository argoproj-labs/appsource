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

	clusterv1 "github.com/argoproj-labs/argocd-app-source/api/v1"

	//?Are these imports correct? They seem to be throwing an error.
	argocdClientSet "github.com/argoproj/argo-cd/pkg/apiclient"
	applicationTypes "github.com/argoproj/argo-cd/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/pkg/apiclient/project"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ArgoAppClientset argocdClientSet.Client

	PatternRegexCompiler           *regexp.Regexp
	ProjectGroupRegexCompiler      *regexp.Regexp
	FirstCaptureGroupRegexCompiler *regexp.Regexp
}

//+kubebuilder:rbac:groups=cluster.my.domain,resources=appsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.my.domain,resources=appsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.my.domain,resources=appsources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppSource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AppSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	patternMatchesNamespace, err := r.validateNamespacePattern(ctx, req)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	if patternMatchesNamespace {
		err = r.validateProject(ctx, req)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		err = r.validateApplication(ctx, req)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	} else {
		//Name does not match namespace regex pattern.
		return ctrl.Result{Requeue: true}, errors.New("namespace does not match AppSource config project.pattern")
	}

	return ctrl.Result{}, nil
}

func (r *AppSourceReconciler) validateApplication(ctx context.Context, req ctrl.Request) (err error) {
	//Search for application
	closer, appClient, err := r.ArgoAppClientset.NewApplicationClient()
	if err != nil {
		return err
	}
	defer closer.Close()
	_, err = appClient.Get(ctx, &applicationTypes.ApplicationQuery{Name: &req.Name})
	if err != nil {
		//Application not found, create it
		appSource := &clusterv1.AppSource{}
		_ = r.Get(ctx, req.NamespacedName, appSource)
		_, err = appClient.Create(ctx,
			&applicationTypes.ApplicationCreateRequest{Application: appSource.ApplicationFromSource(req)})
	}
	return
}

func (r *AppSourceReconciler) GetProjectName(namespace string) (result string, err error) {
	match := r.ProjectGroupRegexCompiler.Find([]byte(namespace))
	if match == nil {
		match = r.FirstCaptureGroupRegexCompiler.Find([]byte(namespace))
	}
	if match == nil {
		return "", errors.New("project name could not be found from appsource namespace")
	}
	return string(match), nil
}

func (r *AppSourceReconciler) validateProject(ctx context.Context, req ctrl.Request) (err error) {
	closer, appProjectClient, err := r.ArgoAppClientset.NewProjectClient()
	if err != nil {
		return err
	}
	defer closer.Close()
	projectName, err := r.GetProjectName(req.Namespace)
	if err != nil {
		return err
	}
	_, err = appProjectClient.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	//TODO Implement project creation logic, see commented out section below.
	// if err != nil {
	// 	//Project was not found, therefore we should create it
	// 	appproject_req := v1alpha1.AppProject{}
	// 	_, err = r.ArgoAppClientset.ArgoprojV1alpha1().AppProjects(argocdNS).Create(
	// 		ctx,
	// 		&v1alpha1.AppProject{ObjectMeta: metav1.ObjectMeta{Name: req.Namespace}},
	// 		metav1.CreateOptions{})
	// }
	return
}

// Returns whether requested AppSource object namespace matches allowed project pattern
func (r *AppSourceReconciler) validateNamespacePattern(ctx context.Context, req ctrl.Request) (patternMatchesNamespace bool, err error) {
	// // Collect argocd-source-cm ConfigMap

	patternMatchesNamespace = r.PatternRegexCompiler.Match([]byte(req.Namespace))
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.AppSource{}).
		Complete(r)
}
