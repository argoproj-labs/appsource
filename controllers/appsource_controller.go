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
	argocd_apiclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const appsource_cm_name = "argocd-sourc-cm"

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

	// req currently contains the name and namespace of the AppSource instance being reconciled.

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	client := clientset.CoreV1()

	// Collect argocd-source-cm ConfigMap
	configmap, err := client.ConfigMaps("").Get(ctx, appsource_cm_name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Extract name and namespace from AppSource request
	namespace := req.Namespace
	//TODO Compare AppSource namespace+name against AppSourceConfigMap.data.pattern (regular expression)
	pattern := configmap.Data["project.pattern"]

	var pattern_matches_namespace bool
	pattern_matches_namespace, err = regexp.MatchString(pattern, namespace)
	if err != nil {
		panic(err.Error())
	}

	if pattern_matches_namespace {
		//? Check if ArgoCD Application referenced by req exists
		//TODO Get the AppSource Object using req
		appsource := &clusterv1.AppSource{}
		_ = r.Get(ctx, req.NamespacedName, appsource)

		//TODO Make an ArgoCD project client
		client := argocd_apiclient.NewClient()

		//TODO Get the ArgoCD project
		//TODO If project does not exist then create it
		//TODO
		//TODO Search for req.name within existing applications
		//TODO If err, then app does not exist therefore we should create it
	} else {
		//? Name does not match namespace regex pattern.
		panic(errors.New("Namespace does not match AppSource Project Pattern."))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.AppSource{}).
		Complete(r)
}
