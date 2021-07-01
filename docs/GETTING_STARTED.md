# Getting Started
## 1. Install and Configure ArgoCD
Follow these steps if you __want to get ArgoCD running on a local cluster__, 
or follow the [Getting Started](https://argo-cd.readthedocs.io/en/stable/getting_started/) guide from ArgoCD if you want to configure it for some specific need.

Skip to Step 2 to install AppSource
### Apply ArgoCD Manifest
```shell
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```
### Port-forward ArgoCD Server to localhost
```shell
kubectl port-forward svc/argocd-server -n argocd 8080:443
```
### Create an ArgoCD Service Account to generate API Token
#### Get first-time login admin password
```shell
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
```
#### Log in to admin account
```shell
argocd login localhost:8080 --insecure
```
Use the username `admin` and the password from the previous section.
#### Create appsource account
Open the `argocd-cm` config map
```shell
kubectl edit configmap argocd-cm -n argocd
```
Add `appsource` to the list of accounts by editing the `data` field.
```yaml
data:
  accounts.appsource: apiKey, login
```
##### Optional: Update appsource password and diable admin account
###### Update appsource password
Use _admin password_ when prompted for the `current password`
```shell
argocd account update-password --account appsource
```
###### Disable admin account
Per ArgoCD Guidlines, you should disable the `admin` account after creating a ArgoCD user account.
```shell
kubectl edit configmap argocd-cm -n argocd
```
Disable the admin account within the `data` field
```yaml
data:
  accounts.appsource: apiKey, login
  admin.enabled: "false"
```
###### Log in with appsource account
Use the updated password to login.
```shell
argocd login localhost:8080 --insecure --account appsource
```
#### Give appsource account necessary API permissions
Open the `argocd-rbac-cm` config map
```shell
kubectl edit configmap argocd-rbac-cm -n argocd
```
Create appsource role and necessary permissions, give appsource account the appsource role.
```yaml
data:
  policy.csv: |
    p, role:appsource, applications, *, */*, allow
    p, role:appsource, projects, *, *, allow
    p, role:appsource, repositories, *, *, allow
    p, role:appsource, cluster, *, *, allow
    p, role:appsource, clusters, *, *, allow
    g, appsource, role:appsource
```
## 2. Install AppSource
Prior to installing the AppSource controller, you need to create the admin configuration and ArgoCD API token secret for the controller.
### Create Admin ConfigMap
A minimal admin configmap looks like:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-appsource-cm
  namespace: argocd
data:
  #Use localhost:8080 ArgoCD was installed locally
  argocd.address: 172.17.0.6:8080 # Argo CD server hostname and port
  project.pattern: '(?P<project>.*)-us-(west|east)-(\d.*)'
```
### Create Secret containing a API Token for AppSource ArgoCD account
```shell
kubectl create secret generic argocd-appsource-secret --from-literal argocd-token=$(argocd account generate-token --account appsource)
```
This creates a secret containing a newly generated API token for the `appource` account
### Install AppSource CRD and controller
```shell
kubectl apply -f https://raw.githubusercontent.com/aceamarco/argocd-app-source/dev/manifests/install.yaml -n argocd
```
This will create a AppSource custom resource definition, deployment, service account, role, and rolebinding for the AppSource controller.
### Optional: Open AppSource controller logs
If you'd like to follow the AppSource controller logs, in a new terminal run:
```shell
kubectl logs --follow deploy/argocd-appsource-controller -n argocd
```
__Users can now create AppSource instances within their own project namespaces__
# FAQ
#### My logs are not showing up, what happened?
The install manifest creates a deployment for the latest AppSource controller image, the deployment then creates a ReplicaSet and Pod where the deployment will run.

To view the state of the deployment, run:
```shell
kubectl describe deploymeny argocd-appsource-controller -n argocd
```

To view the state of the replicaset, run:
```shell
kubectl describe replicaset argocd-appsource-controller -n argocd
```

To the view the state of the pod, run:
```shell
kubectl describe pod argocd-appsource-controller -n argocd
```