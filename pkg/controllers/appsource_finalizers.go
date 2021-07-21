package controllers

import (
	"context"
	"errors"

	applicationTypes "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
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

func (r *AppSourceReconciler) ResolveFinalizers(ctx context.Context, appsource *argoprojv1alpha1.AppSource) (err error) {
	for _, appsourceFinalizer := range appsource.GetFinalizers() {
		for _, finalizer := range finalizers {
			if appsourceFinalizer == finalizer {
				switch finalizer {
				case "application-finalizer.appsource.argoproj.io":
					_, err = r.ArgoApplicationClient.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
						Name:    &appsource.Name,
						Cascade: &cascadeFalse,
					})
				case "application-finalizer.appsource.argoproj.io/cascade":
					_, err = r.ArgoApplicationClient.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
						Name:              &appsource.Name,
						Cascade:           &cascadeTrue,
						PropagationPolicy: &background,
					})
				default:
					err = errors.New("invalid finalizer")
				}
				if err != nil {
					return err
				}
				controllerutil.RemoveFinalizer(appsource, finalizer)
				if err = r.Update(ctx, appsource); err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}
