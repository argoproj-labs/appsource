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
	appclientset "github.com/argoproj/argo-cd/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	//? I feel like I'm not using the reconciler to its fullest potential.
	//? The reconcile logic will always need the ArgoCD client as well as
	//? the Kubernetes client, maybe I can add them to the reconciler type?
	//? But then how will the ArgoCD client by dynamically initialized based
	//? on the address provided in the AppSource ConfigMap?
	ArgoAppClientset appclientset.Interface
	KubeClientset    kubernetes.Interface
}

const (
	appsource_cm_name = "argocd-sourc-cm"
	argocd_namespace  = "argocd"
)

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

	pattern_matches_namespace, err := r.ValidateNamespacePattern(ctx, req)
	if err != nil {
		panic(err)
	}

	if pattern_matches_namespace {
		err = r.ValidateProject(ctx, req)
		if err != nil {
			panic(err)
		}
		err = r.ValidateApplication(ctx, req)
		if err != nil {
			panic(err)
		}
	} else {
		//Name does not match namespace regex pattern.
		panic(errors.New("Namespace does not match AppSource Project Pattern."))
	}

	return ctrl.Result{}, nil
}

func (r *AppSourceReconciler) ValidateApplication(ctx context.Context, req ctrl.Request) (err error) {
	//Search for application
	appclient := r.ArgoAppClientset.ArgoprojV1alpha1().Applications(req.Namespace)
	_, err = appclient.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		//Application not found, create it
		appsource := &clusterv1.AppSource{}
		_ = r.Get(ctx, req.NamespacedName, appsource)
		_, err = appclient.Create(ctx,
			appsource.ApplicationFromSource(req), metav1.CreateOptions{})
	} //? Why is the linter compaling that I can't use *v1alpha1.Application when that is
	//? the argument type that it takes in?
	return
}

func (r *AppSourceReconciler) ValidateProject(ctx context.Context, req ctrl.Request) (err error) {
	appproject_client := r.ArgoAppClientset.ArgoprojV1alpha1().AppProjects(argocd_namespace)
	_, err = appproject_client.Get(ctx, req.Namespace, metav1.GetOptions{})
	//TODO Implement project creation logic, see commented out section below.
	// if err != nil {
	// 	//Project was not found, therefore we should create it
	// 	appproject_req := v1alpha1.AppProject{}
	// 	_, err = r.ArgoAppClientset.ArgoprojV1alpha1().AppProjects(argocd_namespace).Create(
	// 		ctx,
	// 		&v1alpha1.AppProject{ObjectMeta: metav1.ObjectMeta{Name: req.Namespace}},
	// 		metav1.CreateOptions{})
	// }
	return
}

// Returns whether requested AppSource object namespace matches allowed project pattern
func (r *AppSourceReconciler) ValidateNamespacePattern(ctx context.Context, req ctrl.Request) (pattern_matches_namespace bool, err error) {
	// Collect argocd-source-cm ConfigMap
	configmap, err := r.KubeClientset.CoreV1().ConfigMaps("").Get(ctx, appsource_cm_name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	// Extract name and namespace from AppSource request
	namespace := req.Namespace
	pattern := configmap.Data["project.pattern"]

	pattern_matches_namespace, err = regexp.MatchString(pattern, namespace)
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.AppSource{}).
		Complete(r)
}
