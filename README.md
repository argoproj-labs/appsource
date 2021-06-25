# AppSource CRD
A decentralized manager for ArgoCD — allow sub-admins to create and manage their own applications on ArgoCD.
## Summary
- Traditionally, ArgoCD applications are managed by a single entity, this formally called the multi-tenant model of ArgoCD. However, some users would like to provide their organizaitons ArgoCD as a self-serviced tool. 
- This alternative model can be called the "self-service model", where sub-admins are allowed to create and manage their own collection of applications without the need for Admin approval.
## Motivation
- Organizations would like to be able to provide development teams access to ArgoCD without needing to maintain/approve actions made to the Dev team's collection of applications. 
## Installation
- Clone this repo
- Install ArgoCD
  - Create a ArgoCD account with apiKey capabilities
  - Generate a token for that account using `argocd account generate-token --account <username>`
- Run `make install` to apply CRD to your cluster
- Create a admin ConfigMap, see `config/samples` for example manifests
- Run `make run` to start the controller
- Apply your AppSource manifests from your own project namespaces, see `config/samples` for example manfiests
  
*Please note*, currently the AppSource controller is in a proof-of-concept stage, the ArgoCD API is initialized with TLS Certificate authorization disabled.
