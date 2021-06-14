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
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1/types"
	"github.com/argoproj/argo-cd/v2/pkg/clientset"
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

	//? I feel like I'm not using the reconciler to its fullest potential.
	//? The reconcile logic will always need the ArgoCD client as well as
	//? the Kubernetes client, maybe I can add them to the reconciler type?
	//? But then how will the ArgoCD client by dynamically initialized based
	//? on the address provided in the AppSource ConfigMap?
	argocd_client apiclient.Client
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
	name := req.Name
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

		//TODO Get the ArgoCD project
		closer, projectc, err := r.argocd_client.NewProjectClient()
		if err != nil {
			panic(errors.New("Unable to establish Project client."))
		}
		projquery := project.ProjectQuery{Name: namespace}
		proj, err := projectc.Get(ctx, &projquery)
		if err != nil {
			//Project should exist, is being created by admin team
			panic(errors.New("Project not found."))
		}
		closer.Close() //? Am I using this close function correctly?
		//TODO Search project for application
		closer, appc, err := r.argocd_client.NewApplicationClient()
		if err != nil {
			panic(errors.New("Unable to create Application client"))
		}
		appquery := application.ApplicationQuery{
			Name:     &name,
			Projects: []string{namespace},
		}
		app, err := appc.Get(ctx, &appquery)
		if err != nil {
			//Application does not exist, create it
			appspec := types.ApplicationSpec{
				Project: namespace,
			}
			appstatus := types.ApplicationStatus{}
			appoperations := types.Operation{}
			application := types.Application{
				Spec:      appspec,
				Status:    appstatus,
				Operation: appoperations,
			}
			appcreate := application.ApplicationCreateRequest{
				Application: application,
				Upsert:      true,
				Validate:    true,
			}
			appc.Create(ctx, appcreate)
			//? Am creating this application correctly? What defaults should I use for the app configuration?
		}
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
