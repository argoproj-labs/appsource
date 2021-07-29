package controllers

import (
	"context"
	"errors"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
					appSource.Status.History = append(appSource.Status.History, &appsource.AppSourceCondition{
						Type:       appsource.ApplicationDeletionError,
						Message:    err.Error(),
						Status:     appsource.ConditionFalse,
						ObservedAt: metav1.Now(),
					})
					return err
				}
				appSource.Status.History = append(appSource.Status.History, &appsource.AppSourceCondition{
					Type:       appsource.ApplicationDeletionSuccess,
					Message:    appsource.ApplicationDeletionMsg,
					Status:     appsource.ConditionTrue,
					ObservedAt: metav1.Now(),
				})
				controllerutil.RemoveFinalizer(appSource, finalizer)
				if err = r.Update(ctx, appSource); err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}
