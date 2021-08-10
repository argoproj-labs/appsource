#!/usr/bin/env bash

KUBECTL=${which kubectl}

# Install the ArgoCD CLI
curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x /usr/local/bin/argocd

ARGOCD=${which argocd}

# Install ArgoCD
${KUBECTL} create namespace argocd
${KUBECTL} apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Port forward ArgoCD Server to localhost:8080
${KUBECTL} wait --timeout=10m --for=condition=Ready pod argocd-server -n argocd
${KUBECTL} port-forward svc/argocd-server -n argocd 8080:443

# Log in to admin account
ADMIN_OTP=${kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo}
ARGOCD=${which argocd}
echo ADMIN_OTP | ${argocd} login localhost:8080 --username admin --insecure

# Create ArgoCD account with API capabilities
touch tests/manifests/argocd_appsource_account.yaml
touch tests/manifests/argocd_appsource_rbac.yaml
${KUBECTL} describe configmap argocd-cm -n argocd > tests/manifests/argocd_appsource_account.yaml
${KUBECTL} describe configmap argocd-rbac-cm -n argocd > tests/manifests/argocd_appsource_rbac.yaml

cat tests/manifests/appsource_account_data.yaml >> tests/manifests/argocd_appsource_account.yaml
cat tests/manifests/appsource_rbac_data.yaml >> tests/manifests/argocd_appsource_rbac.yaml
${KUBECTL} apply -f tests/manifests/argocd_appsource_account.yaml
${KUBECTL} apply -f tests/manifests/argocd_appsource_rbac.yaml

# Create ArgoCD API token
ARGOCD_TOKEN=${ARGOCD account generate-token --account appsource}
${KUBECTL} -n ${ARGOCD} create secret generic argocd-appsource-secret --from-literal argocd-token=${ARGOCD_TOKEN}

# Install AppSource CRD and controller
${KUBECTL} -n argocd apply -f https://raw.githubusercontent.com/argoproj-labs/appsource/master/manifests/install.yaml
