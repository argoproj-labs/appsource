package controllers

import (
	"context"
	"errors"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"

	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

var (
	finalizers = []string{
		"application-finalizer.appsource.argoproj.io",
		"application-finalizer.appsource.argoproj.io/cascade",
	}
	cascadeFalse bool   = false
	cascadeTrue  bool   = true
	background   string = "background"
)

func (r *AppSourceReconciler) ResolveFinalizers(ctx context.Context, appSource *appsource.AppSource) (err error) {
	for _, appSourceFinalizer := range appSource.GetFinalizers() {
		for _, finalizer := range finalizers {
			if appSourceFinalizer == finalizer {

				if (appSource.Status.Operation.Type == appsource.ArgoCDAppDeletion) && (appSource.Status.Operation.FinishedAt == nil) {
					if err = r.RetryOperation(ctx, appSource); err != nil {
						return err
					}
				} else {
					if err = r.NewOperation(ctx, appSource, appsource.ArgoCDAppDeletion); err != nil {
						return err
					}
				}

				switch finalizer {
				case "application-finalizer.appsource.argoproj.io":
					_, err = r.Clients.Applications.Client.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
						Name:    &appSource.Name,
						Cascade: &cascadeFalse,
					})
				case "application-finalizer.appsource.argoproj.io/cascade":
					_, err = r.Clients.Applications.Client.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
						Name:              &appSource.Name,
						Cascade:           &cascadeTrue,
						PropagationPolicy: &background,
					})
				default:
					err = errors.New("invalid finalizer")
				}

				if err != nil {
					if err = r.FinishOperation(ctx, appSource, &appsource.AppSourceCondition{
						Type:    appsource.ApplicationConditionDeletionError,
						Message: err.Error(),
					}); err != nil {
						return err
					}
					return err
				}
				if err = r.FinishOperation(ctx, appSource, nil); err != nil {
					return err
				}
				if err = r.Update(ctx, appSource); err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}
