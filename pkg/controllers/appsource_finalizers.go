package controllers

import (
	"context"
	"errors"

	applicationTypes "github.com/argoproj/argo-cd/pkg/apiclient/application"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

var (
	finalizers = []string{
		"argoproj.io/appsource-finalizer-fg",
		"argoproj.io/appsource-finalizer-bg",
		"argoproj.io/appsource-finalizer-fg-cascade",
		"argoproj.io/appsource-finalizer-bg-cascade",
	}
	cascadeFalse bool = false
	cascadeTrue  bool = true
)

func (r *AppSourceReconciler) ResolveFinalizers(ctx context.Context, appsource *argoprojv1alpha1.AppSource) (err error) {
	for _, appsourceFinalizer := range appsource.GetFinalizers() {
		for _, finalizer := range finalizers {
			if appsourceFinalizer == finalizer {
				switch finalizer {
				case "argoproj.io/appsource-finalizer-fg":
					err = r.deleteApplicationFG(ctx, appsource, &cascadeFalse)
				case "argoproj.io/appsource-finalizer-bg":
					ch := make(chan error)
					go r.deleteApplicationBG(ctx, appsource, &cascadeFalse, ch)
					err = <-ch
				case "argoproj.io/appsource-finalizer-fg-cascade":
					err = r.deleteApplicationFG(ctx, appsource, &cascadeTrue)
				case "argoproj.io/appsource-finalizer-bg-cascade":
					ch := make(chan error)
					go r.deleteApplicationBG(ctx, appsource, &cascadeFalse, ch)
					err = <-ch
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

func (r *AppSourceReconciler) deleteApplicationBG(ctx context.Context, appsource *argoprojv1alpha1.AppSource, cascade *bool, ch chan error) {
	_, err := r.ArgoApplicationClient.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
		Name:    &appsource.Name,
		Cascade: cascade,
	})
	ch <- err
}

func (r *AppSourceReconciler) deleteApplicationFG(ctx context.Context, appsource *argoprojv1alpha1.AppSource, cascade *bool) (err error) {
	_, err = r.ArgoApplicationClient.Delete(ctx, &applicationTypes.ApplicationDeleteRequest{
		Name:    &appsource.Name,
		Cascade: cascade,
	})
	return
}
