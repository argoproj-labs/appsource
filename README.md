# AppSource CRD
A decentralized manager for ArgoCD — allow sub-admins to create and manage their own applications on ArgoCD.
## Installation
- Apply the AppSource CRD manifests `https://raw.githubusercontent.com/aceamarco/argocd-app-source/master/manifests/install.yaml`
- Create a secret containing your ArgoCD token named `argocd-appsource-secret`
- Configure the AppSource Controller using the [`argocd-appsource-cm` configmap](./manifests/samples/sample_admin_config.yaml)
- For a more detailed instructions, see the [Getting Started Guide](docs/GETTING_STARTED.md)
## Summary
- Traditionally, ArgoCD applications are managed by a single entity, this formally called the multi-tenant model of ArgoCD. However, some users would like to provide their organizaitons ArgoCD as a self-serviced tool. 
- This alternative model can be called the "self-service model", where sub-admins are allowed to create and manage their own collection of applications without the need for Admin approval.
## Motivation
- Organizations would like to be able to provide development teams access to ArgoCD without needing to maintain/approve actions made to the Dev team's collection of applications.

