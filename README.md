# argocd-app-manager
A decentralized manager for ArgoCD — allow sub-admins to create and manage their own applications on ArgoCD.
## Summary
- Traditionally, ArgoCD applications are managed by a single entity, this formally called the multi-tenant model of ArgoCD. However, some users would like to provide their organizaitons ArgoCD as a self-serviced tool. 
- This alternative model can be called the "self-service model", where sub-admins are allowed to create and manage their own collection of applications without the need for Admin approval.
## Motivation
- Organizations would like to be able to provide development teams access to ArgoCD without needing to maintain/approve actions made to the Dev team's collection of applications.
## Installation
- argo-app-manager should be installed into your cluster by adding a ConfigMap resource to your cluster that points to the controller.
## Workflow
### Admin (Installer)
- Create a ConfigMap instance with 
  - Name argo-appsource-sm
  - ArgoCD server host URL and port
  - ArgoCD access token
  - Naming convention for applications users will be allowed to create
### Sub-admin (User)
- Create a AppSource instance with
  - Application name
  - ArgoCD Git Repo URL
  - Path to application within ArgoCD Git Repo


TODO: Add example images and documentation images.