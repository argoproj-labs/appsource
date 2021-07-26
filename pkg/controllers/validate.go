package controllers

import (
	"context"
	"errors"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

//validateApplication Validates the existence of ArgoCD Application specified by the AppSource request.
//If the Application does not exist, it is created
func (r *AppSourceReconciler) validateApplication(ctx context.Context, appSource *appsource.AppSource) (err error) {

	// Get the corresponding ArgoCD Application
	application, found := r.Clients.Applications.Client.Get(ctx, &applicationTypes.ApplicationQuery{Name: &appSource.Name})
	if found != nil {
		//Create ArgoCD Application

		if ok := r.NewOperation(ctx, appSource, appsource.ArgoCDAppCreation); ok != nil {
			return ok
		}

		projectName, err := r.getProjectName(ctx, appSource)
		if err != nil {
			if ok := r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
				Type:    appsource.ApplicationConditionInvalidSpecError,
				Message: err.Error(),
			}); ok != nil {
				return ok
			}
			return err
		}

		appSourceDestination := v1alpha1.ApplicationDestination{
			Server:    r.ClusterHost,
			Namespace: appSource.Namespace,
		}
		err = r.validateProjectDestinations(ctx, projectName, appSourceDestination)
		if err != nil {
			if ok := r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
				Type:    appsource.ApplicationConditionCreationError,
				Message: err.Error(),
			}); ok != nil {
				return ok
			}
			return err
		}

		// Send request to create Application
		application, err = r.Clients.Applications.Client.Create(ctx,
			&applicationTypes.ApplicationCreateRequest{
				Application: v1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      appSource.Name,
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
		if err != nil {
			// Application could not be created
			if ok := r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
				Type:    appsource.ApplicationConditionCreationError,
				Message: err.Error(),
			}); ok != nil {
				return ok
			}
			return err
		} else {
			// Application was created successfully
			if ok := r.FinishOperation(ctx, appSource, nil); ok != nil {
				return ok
			}
		}
	}

	// Update the ArgoCD Application Status with found or created application
	return r.UpdateArgoCDApplicationStatus(ctx, appSource, &application.Status)
}

//validateProject Validates AppSource project against ArgoCD, empty project is created if it does not exist
func (r *AppSourceReconciler) validateProject(ctx context.Context, appSource *appsource.AppSource) (err error) {

	// Get Project name from AppSource namespace
	projectName, err := r.getProjectName(ctx, appSource)
	if err != nil {
		if ok := r.SetCondition(ctx, appSource, &appsource.AppSourceCondition{
			Type:    appsource.ApplicationConditionInvalidSpecError,
			Message: err.Error(),
		}); ok != nil {
			return ok
		}
		return err
	}

	_, projectFound := r.Clients.Projects.Client.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	if projectFound != nil {
		// Project not found, create a new ArgoCD Project

		if ok := r.NewOperation(ctx, appSource, appsource.ArgoCDProjectCreation); ok != nil {
			return ok
		}

		var proj *ProjectTemplate
		if proj, err = r.FindProject(projectName); err != nil {
			if ok := r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
				Type:    appsource.ProjectConditonCreationError,
				Message: err.Error(),
			}); ok != nil {
				return ok
			}
			return err
		}

		// Create ArgoCD Project
		if _, err = r.Clients.Projects.Client.Create(ctx, &projectTypes.ProjectCreateRequest{
			Project: &v1alpha1.AppProject{
				ObjectMeta: metav1.ObjectMeta{
					Name: projectName,
				},
				Spec: *proj.Spec,
			},
			Upsert: false,
		}); err != nil {
			// Project Creation failed
			if ok := r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
				Type:    appsource.ProjectConditonCreationError,
				Message: err.Error(),
			}); ok != nil {
				return ok
			}
		} else {
			// Project created successfully
			if ok := r.FinishOperation(ctx, appSource, nil); ok != nil {
				return ok
			}
		}
		return err
	}
	return err
}

//validateProjectDestinations Validates the existence of Application destination within AppProject Destinations list
//Appends the destination in question if it is not present already
func (r *AppSourceReconciler) validateProjectDestinations(ctx context.Context, projectName string, appSourceDestination v1alpha1.ApplicationDestination) (err error) {
	appProject, err := r.Clients.Projects.Client.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
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
	_, err = r.Clients.Projects.Client.Update(ctx, &projectTypes.ProjectUpdateRequest{Project: appProject})
	return err
}

//getProjectName returns the first capturing group named "project" a namespace, defaults to first capturing group
//Looks for the left-most match to a named capture group called project (case-sensitive), i.e (?P<project>.*)
//If the named group is not found, it will grab the first capture group present, i.e (.*)
func (r *AppSourceReconciler) getProjectName(ctx context.Context, appSource *appsource.AppSource) (result string, err error) {
	proj, err := r.FindProject(appSource.Namespace)
	if err != nil {
		return "", err
	}
	matches := proj.PatternCompiler.FindStringSubmatch(appSource.Namespace)
	if len(matches) < 2 {
		// Project name could not be extracted
		return "", errors.New("no capturing groups found")
	}
	matchMap := make(map[string]string)
	//Map potentially named groups to submatch
	for i, subMatch := range proj.PatternCompiler.SubexpNames() {
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
