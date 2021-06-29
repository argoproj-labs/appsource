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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argocdClientSet "github.com/argoproj/argo-cd/pkg/apiclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoprojv1alpha1 "github.com/argoproj-labs/argocd-app-source/api/v1alpha1"
	"github.com/argoproj-labs/argocd-app-source/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	//AppSource configmap name
	appSourceCM = "argocd-appsource-cm"
	//In-cluster server address
	clusterServerName = "https://kubernetes.default.svc"
	//ArgoCD namespace
	argocdNamespace = "argocd"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(argoprojv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

//GetAppSourceConfigmapOrDie returns the AppSource ConfigMap defined by admins or crashes with error
func getAppSourceConfigmapOrDie() (appSourceConfigmap *v1.ConfigMap) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
	//namespace, _, err := clientConfig.Namespace()
	config, err := clientConfig.ClientConfig()
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes cluster config")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes clientset")
		os.Exit(1)
	}
	//Get AppSource ConfigMap
	appSourceConfigmap, err = clientset.CoreV1().ConfigMaps("argocd").Get(context.TODO(), appSourceCM, metav1.GetOptions{})
	if err != nil {
		setupLog.Error(err, "failed to get appSource configmap")
		os.Exit(1)
	}
	return
}

func getTokenSecretOrDie() (token string) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
	//namespace, _, err := clientConfig.Namespace()
	config, err := clientConfig.ClientConfig()
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes cluster config")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes clientset")
		os.Exit(1)
	}
	secret, err := clientset.CoreV1().Secrets("argocd").Get(context.TODO(), "argocd-appsource-secret", metav1.GetOptions{})
	if err != nil {
		setupLog.Error(err, "unable to get ArgoCD token secret")
		os.Exit(1)
	}
	return secret.StringData["argocd-token"]
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "02ff6e16.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//AppSourceReconciler attribute initialization
	appSourceConfigmap := getAppSourceConfigmapOrDie()
	argocdClient, err := argocdClientSet.NewClient(
		&argocdClientSet.ClientOptions{
			ServerAddr: appSourceConfigmap.Data["argocd.address"],
			AuthToken:  appSourceConfigmap.Data["argocd.token"],
			//TODO Add cli flags to determine insecure connection
			Insecure: true,
		})
	if err != nil {
		//! Hmm, this is weird — the error showing up in logs has to do with the argocd token, but this error message
		//! isn't showing up in the logs
		setupLog.Error(err, fmt.Sprintf("Unable to start ArgoCD client, token %s", appSourceConfigmap.Data["argocd.token"]))
		os.Exit(1)
	}
	closer, argocdApplicationClient := argocdClient.NewApplicationClientOrDie()
	defer closer.Close()
	closer, argocdProjectClient := argocdClient.NewProjectClientOrDie()
	defer closer.Close()

	setupLog.Info(fmt.Sprintf("Created ArgoCD Clients with token: %s", appSourceConfigmap.Data["argocd.token"]))
	setupLog.Info(fmt.Sprintf("$TOKEN: %s", os.Getenv("TOKEN")))
	setupLog.Info(fmt.Sprintf("Token secret value is: %s", getTokenSecretOrDie()))

	//AppSourceReconciler Initialization
	if err = (&controllers.AppSourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		//AppSourceReconciler specific Attributes
		ArgoApplicationClient: argocdApplicationClient,
		ArgoProjectClient:     argocdProjectClient,
		PatternRegexCompiler:  regexp.MustCompile(appSourceConfigmap.Data["project.pattern"]),
		ClusterHost:           clusterServerName,
		ArgocdNS:              argocdNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AppSource")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
