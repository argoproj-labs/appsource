package controllers

import (
	"context"
	"errors"
	"os"
	"strings"

	argocdClientSet "github.com/argoproj/argo-cd/pkg/apiclient"
	argocd "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/kballard/go-shellquote"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	//AppSource configmap name
	appSourceCM = "argocd-appsource-cm"
)

var (
	flags map[string]string
)

type ProjectTemplate struct {
	NamePattern string                 `json:"namePattern"`
	Spec        *argocd.AppProjectSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

func getFlag(key, fallback string) string {
	val, ok := flags[key]
	if ok {
		return val
	}
	return fallback
}

func getBoolFlag(key string) bool {
	return getFlag(key, "false") == "true"
}

func loadFlags(clientOpts string) (err error) {
	opts, err := shellquote.Split(clientOpts)
	if err != nil {
		return err
	}
	flags = make(map[string]string)
	var key string
	for _, opt := range opts {
		if strings.HasPrefix(opt, "--") {
			if key != "" {
				flags[key] = "true"
			}
			key = strings.TrimPrefix(opt, "--")
		} else if key != "" {
			flags[key] = opt
			key = ""
		} else {
			return errors.New("clientOpts invalid at '" + opt + "'")
		}
	}
	if key != "" {
		flags[key] = "true"
	}

	return nil
}

func GetClientOpts(appsourceConfigMap v1.ConfigMap) (*argocdClientSet.ClientOptions, error) {
	err := loadFlags(appsourceConfigMap.Data["argocd.clientOpts"])
	if err != nil {
		return nil, err
	}

	token := os.Getenv("ARGOCD_TOKEN")

	return &argocdClientSet.ClientOptions{
		ServerAddr:        appsourceConfigMap.Data["argocd.address"],
		AuthToken:         token,
		PlainText:         getBoolFlag("plaintext"),
		Insecure:          getBoolFlag("insecure"),
		CertFile:          getFlag("server-crt", ""),
		ClientCertFile:    getFlag("client-crt", ""),
		ClientCertKeyFile: getFlag("client-crt-key", ""),
		GRPCWeb:           getBoolFlag("grpc-web"),
		GRPCWebRootPath:   getFlag("grpc-web-root-path", ""),
		PortForward:       getBoolFlag("port-forward"),
		//? How should headers be handled?
		PortForwardNamespace: getFlag("port-forward-namespace", ""),
	}, nil
}

//GetAppSourceConfigmapOrDie returns the AppSource ConfigMap defined by admins or crashes with error
func GetAppSourceConfigmap() (appsourceConfigMap *v1.ConfigMap, err error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
	//namespace, _, err := clientConfig.Namespace()
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	//Get AppSource ConfigMap
	appsourceConfigMap, err = clientset.CoreV1().ConfigMaps("argocd").Get(context.TODO(), appSourceCM, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return
}
