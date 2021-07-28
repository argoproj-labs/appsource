package controllers

import (
	"context"
	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *AppSourceReconciler) NewOperation(ctx context.Context, appSource *appsource.AppSource, op appsource.OperationType) error {
	startTime := metav1.Now()
	appSource.Status.Operation = appsource.Operation{
		Type:       op,
		Phase:      appsource.OperationRunning,
		StartedAt:  &startTime,
		RetryCount: 0,
	}
	return r.Status().Update(context.Background(), appSource)
}

func (r *AppSourceReconciler) RetryOperation(ctx context.Context, appSource *appsource.AppSource) error {
	appSource.Status.Operation.RetryCount++
	return r.Status().Update(context.Background(), appSource)
}

func (r *AppSourceReconciler) FinishOperation(ctx context.Context, appSource *appsource.AppSource, condition *appsource.AppSourceCondition) error {
	if condition != nil {
		switch condition.Status {
		case appsource.ConditionFalse:
			appSource.Status.Operation.Phase = appsource.OperationError

		case appsource.ConditionTrue:
			appSource.Status.Operation.Phase = appsource.OperationSucceeded
		}
	}
	condition.LastTransitionTime = metav1.Now()
	appSource.Status.Condition = condition
	finishTime := metav1.Now()
	appSource.Status.Operation.FinishedAt = &finishTime

	return r.Status().Update(context.Background(), appSource)
}

func (r *AppSourceReconciler) SetCondition(ctx context.Context, appSource *appsource.AppSource, condition *appsource.AppSourceCondition) error {
	condition.LastTransitionTime = metav1.Now()
	condition.LastTransitionTime = metav1.Now()
	appSource.Status.Condition = condition
	return r.Status().Update(context.Background(), appSource)
}
