# Getting Started
## 1. Install ArgoCD
Follow these steps if you __want to get ArgoCD running on a local cluster__, 
or follow the [Getting Started](https://argo-cd.readthedocs.io/en/stable/getting_started/) guide from ArgoCD if you want to install it to some specific needs.

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
argocd login localhost:8080 --username admin --insecure
```
Use the password from the previous section.

## 2. Create AppSource account
Open the `argocd-cm` config map
```shell
kubectl edit configmap argocd-cm -n argocd
```
Add `appsource` to the list of accounts by editing the `data` field.
```yaml
data:
  accounts.appsource: apiKey, login
```
### Optional: Update appsource password and disable admin account
#### Update appsource password
Use _admin password_ when prompted for the `current password`
```shell
argocd account update-password --account appsource
```
#### Disable admin account
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
### Log in with appsource account
Use the admin or updated (optional step above) password to log in.
```shell
argocd login localhost:8080 --insecure --username appsource
```
### Give appsource account necessary API permissions
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
## 3. Install AppSource
Prior to installing the AppSource controller, you need to create the admin configuration and ArgoCD API token secret for the controller.
### Create Admin ConfigMap
Here is what a minimal admin configmap looks like:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-appsource-cm
  namespace: argocd
data:
  argocd.address: localhost:8080
  argocd.clientOpts: "--insecure"
  project.profiles: |
    - default:
        namePattern: .*
        spec:
          description: Default AppSource project
          sourceRepos:
            - '*'
```
### Create Secret containing a API Token for AppSource ArgoCD account
```shell
export ARGOCD_TOKEN=$(argocd account generate-token --account appsource)
kubectl -n argocd create secret generic argocd-appsource-secret --from-literal argocd-token=$ARGOCD_TOKEN
```
This creates a secret containing a newly generated API token for the `appource` account
### Install AppSource CRD and controller
```shell
kubectl apply -f https://raw.githubusercontent.com/arogproj-labs/appsource/master/manifests/install.yaml -n argocd
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
