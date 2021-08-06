package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appsource "github.com/argoproj-labs/argocd-app-source/pkg/api/v1alpha1"
)

var _ = Describe("AppSource controller", func() {

	const (
		AppSourceName = "sample1"
		AppSourceNameSpace = "defualt"
		
		timeout = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating AppSource Status", func() {
		It("Should Change the AppSource Status Condition type", func(){
			By("Creating a new ArgoCD Application", func() {
				ctx := context.Background()
				appSource := appsource.AppSource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: ,
					},
				}
			})
		})
	})
})
