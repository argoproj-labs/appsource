# How the AppSource test environment works

The AppSource test environment requires three things

- A kubernetes cluster w/ the AppSource CRD installed
- An ArgoCD instance
- A AppSource controller instance

For the Kubernetes cluster and AppSource CRD, our test framework uses [Kubebuilder's envtest framework](https://book.kubebuilder.io/reference/envtest.html), with the USE_EXISTING_CLUSTER constant set to true.

The exisitng cluster is a k3s lightweight cluster, in it is installed ArgoCD and our AppSource CRD/Controller.

Before we can start testing the appsource controller, we need to do a few things.

- Create k3s cluster
- Install ArgoCD
- Port-forward the ArgoCD server to localhost:8080
- Create our test `appsource` argocd account with API capabilities
- Give the `appsource` account permission to manage our stub ArgoCD instances resources
- Generate an API token an store it in a secret for the controller to use
- Create ArgoCD configmap
- Install the Controller CRD and controller

## Creating the k3s cluster

Within the `.github/workflows/ci.yaml` file, in the `teste2e` job, we create a k3s instance by using the `[debianmaster/actions-k3s@master`](https://github.com/marketplace/actions/actions-k3s) Github action that creates a k3s cluster for us.

## Installing ArgoCD

Since this is a kubernetes cluster we can now run `kubectl` commmands. So installing ArgoCD is done the same way you would on any other cluster.

## Port-forward the ArgoCD server to localhost:8080

Using `kubectl wait --for=condition=Ready deploy/argocd-server` we can wait for the ArgoCD server to finish spinning up before we can prot-forward the argocd-server pod to localhost:8080

## Setting up our AppSource ArgoCD account

Using bash scripts, this whole process has been automated, we can pipe the data fields needed to the end of the necessary ArgoCD configmaps