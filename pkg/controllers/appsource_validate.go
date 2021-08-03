package controllers

import (
	"context"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	projectTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/project"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

//validateApplication Validates the existence of ArgoCD Application specified by the AppSource request.
//If the Application does not exist, it is created
func (r *AppSourceReconciler) validateApplication(ctx context.Context, appSource *appsource.AppSource, proj *ProjectTemplate) (err error) {

	// Get the corresponding ArgoCD Application
	_, found := r.Clients.Applications.Client.Get(ctx, &applicationTypes.ApplicationQuery{Name: &appSource.Name})
	if found != nil {

		projectName, err := proj.GetProjectName(appSource)
		if err != nil {
			appSource.UpsertConditions(appsource.AppSourceCondition{
				Type:          appsource.ApplicationInvalidSpecError,
				Message:       err.Error(),
				Status:        appsource.ConditionFalse,
				LastProbeTime: metav1.Now(),
			})
			return err
		}

		appSourceDestination := v1alpha1.ApplicationDestination{
			Server:    r.ClusterHost,
			Namespace: appSource.Namespace,
		}
		err = r.validateProjectDestinations(ctx, projectName, appSourceDestination)
		if err != nil {
			appSource.UpsertConditions(appsource.AppSourceCondition{
				Type:          appsource.ApplicationCreationError,
				Message:       err.Error(),
				Status:        appsource.ConditionFalse,
				LastProbeTime: metav1.Now(),
			})
			return err
		}

		// Send request to create Application
		_, err = r.Clients.Applications.Client.Create(ctx,
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
			appSource.UpsertConditions(appsource.AppSourceCondition{
				Type:          appsource.ApplicationCreationError,
				Message:       err.Error(),
				Status:        appsource.ConditionFalse,
				LastProbeTime: metav1.Now(),
			})
			return err
		} else {
			// Application was created successfully
			appSource.UpsertConditions(appsource.AppSourceCondition{
				Type:          appsource.ApplicationCreationSuccess,
				Message:       appsource.ApplicationCreationMsg,
				Status:        appsource.ConditionTrue,
				LastProbeTime: metav1.Now(),
			})
		}
	}

	return nil
}

//validateProject Validates AppSource project against ArgoCD, empty project is created if it does not exist
func (r *AppSourceReconciler) validateProject(ctx context.Context, appSource *appsource.AppSource, proj *ProjectTemplate) (err error) {

	// Get Project name from AppSource namespace
	projectName, err := proj.GetProjectName(appSource)
	if err != nil {
		appSource.UpsertConditions(appsource.AppSourceCondition{
			Type:          appsource.ApplicationCreationError,
			Message:       err.Error(),
			Status:        appsource.ConditionFalse,
			LastProbeTime: metav1.Now(),
		})
		return err
	}

	_, projectFound := r.Clients.Projects.Client.Get(ctx, &projectTypes.ProjectQuery{Name: projectName})
	if projectFound != nil {
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
			appSource.UpsertConditions(appsource.AppSourceCondition{
				Type:          appsource.ApplicationCreationError,
				Message:       err.Error(),
				Status:        appsource.ConditionFalse,
				LastProbeTime: metav1.Now(),
			})
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
