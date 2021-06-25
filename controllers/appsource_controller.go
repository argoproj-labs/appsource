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
	applicationTypes "github.com/argoproj/argo-cd/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/api/v1alpha1"
)

// AppSourceReconciler reconciles a AppSource object
type AppSourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ArgoProjectClient     projectTypes.ProjectServiceClient
	ArgoApplicationClient applicationTypes.ApplicationServiceClient
	PatternRegexCompiler  *regexp.Regexp
	ClusterHost           string
	ArgocdNS              string
}

//+kubebuilder:rbac:groups=argoproj.io,resources=appsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=appsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=argoproj.io,resources=appsources/finalizers,verbs=update

// Reconcile v1.0: Called upon AppSource creation, handles namespace validation and Project/App creation
func (r *AppSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	patternMatchesNamespace := r.PatternRegexCompiler.Match([]byte(req.Namespace))
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
		return ctrl.Result{Requeue: true}, errors.New("namespace does not match AppSource config project.pattern")
	}

	return ctrl.Result{}, nil
}

//GetProjectName returns the first capturing group named "project" a namespace, defaults to first capturing group
//Looks for the left-most match to a named capture group called project (case-sensitive), i.e (?P<project>.*)
//If the named group is not found, it will grab the first capture group present, i.e (.*)
func (r *AppSourceReconciler) GetProjectName(namespace string) (result string, err error) {
	matches := r.PatternRegexCompiler.FindStringSubmatch(namespace)
	if len(matches) < 2 {
		return "", errors.New("no capturing groups found")
	}
	matchMap := make(map[string]string)
	//Map potentially named groups to submatch
	for i, subMatch := range r.PatternRegexCompiler.SubexpNames() {
		if (i != 0) && (subMatch != "") {
			matchMap[subMatch] = matches[i]
		}
	}
	match, ok := matchMap["project"]
	if !ok {
		//First capturing group
		match = matches[1]
	}
	return match, nil
}

//validateProject Validates AppSource project against ArgoCD, empty project is created if it does not exist
func (r *AppSourceReconciler) validateProject(ctx context.Context, req ctrl.Request) (err error) {
	projectName, err := r.GetProjectName(req.Namespace)
	if err != nil {
		return err
	}
	_, err = r.ArgoProjectClient.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	if err != nil {
		//Project was not found
		appProject := v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name: projectName,
			},
		}
		_, err = r.ArgoProjectClient.Create(ctx, &projectTypes.ProjectCreateRequest{
			Project: &appProject,
			Upsert:  false,
		})
		return err
	}
	return err
}

//validateApplication Validates the existence of ArgoCD Application specified by the AppSource request.
//If the Application does not exist, it is created
func (r *AppSourceReconciler) validateApplication(ctx context.Context, req ctrl.Request) (err error) {
	//Search for Application
	_, err = r.ArgoApplicationClient.Get(ctx, &applicationTypes.ApplicationQuery{Name: &req.Name})
	if err != nil {
		//Application not found, create it
		projectName, err := r.GetProjectName(req.Namespace)
		if err != nil {
			return err
		}
		appSource := &argoprojv1alpha1.AppSource{}
		err = r.Get(ctx, req.NamespacedName, appSource)
		if err != nil {
			return err
		}
		err = r.validateProjectSourceRepos(ctx, projectName, appSource)
		if err != nil {
			return err
		}
		appSourceDestination := v1alpha1.ApplicationDestination{
			Server:    r.ClusterHost,
			Namespace: req.Namespace,
		}
		err = r.validateProjectDestinations(ctx, projectName, appSourceDestination)
		if err != nil {
			return err
		}
		_, err = r.ArgoApplicationClient.Create(ctx,
			&applicationTypes.ApplicationCreateRequest{
				Application: v1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      req.Name,
						Namespace: r.ArgocdNS},
					Spec: v1alpha1.ApplicationSpec{
						Source: v1alpha1.ApplicationSource{
							RepoURL: appSource.Spec.RepoURL,
							Path:    appSource.Spec.Path,
						},
						Destination: appSourceDestination,
						Project:     projectName,
					},
				}})
		return err
	}
	return nil
}

//validateProjectSourceRepos Validates the existence of the Applications source repo within the AppProject SourceRepo list
//Appends the Applications source repo if it is not present already
func (r *AppSourceReconciler) validateProjectSourceRepos(ctx context.Context, projectName string, appSource *argoprojv1alpha1.AppSource) (err error) {
	appProject, err := r.ArgoProjectClient.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	if err != nil {
		//Project should exist already
		return err
	}
	for _, sourceRepo := range appProject.Spec.SourceRepos {
		if sourceRepo == appSource.Spec.RepoURL {
			//Source Repo already present in project
			return nil
		}
	}
	//Source Repo not present, add to the list of sourceRepos
	appProject.Spec.SourceRepos = append(appProject.Spec.SourceRepos, appSource.Spec.RepoURL)
	_, err = r.ArgoProjectClient.Update(ctx, &projectTypes.ProjectUpdateRequest{Project: appProject})
	return err
}

//validateProjectDestinations Validates the existence of Application destination within AppProject Destinations list
//Appends the destination in question if it is not present already
func (r *AppSourceReconciler) validateProjectDestinations(ctx context.Context, projectName string, appSourceDestination v1alpha1.ApplicationDestination) (err error) {
	appProject, err := r.ArgoProjectClient.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	if err != nil {
		//Project should exist already
		return err
	}
	for _, destination := range appProject.Spec.Destinations {
		if appSourceDestination == destination {
			//App destination already present in project
			return nil
		}
	}
	//App destination does not exist already
	appProject.Spec.Destinations = append(appProject.Spec.Destinations, appSourceDestination)
	_, err = r.ArgoProjectClient.Update(ctx, &projectTypes.ProjectUpdateRequest{Project: appProject})
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojv1alpha1.AppSource{}).
		Complete(r)
}
