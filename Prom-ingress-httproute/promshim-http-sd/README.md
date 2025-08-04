# Promshim  Ingress
Prometheus shim application to retrieve list of all Ingress and HTTPRoutes. then present it in HTTP_SD (Service definition) format for Prometheus to monitor.

## Requirements
App uses prometheus service account role permission to list Ingresses and Http routes. During deployment, you must ensure that the ServiceAccount used by Pod can list all Ingresses  across all namespaces.

## Build and push to registry
```bash
podman build --arch amd64 .  -t dennysv/promshim-http-sd:latest
podman push dennysv/promshim-http-sd:latest
```

## App design details
**Authentication** : Pod is expecting to be running on EKS cluster with SA having necessary permissions. First it will try of In_cluster authentication, if not will try to load Kubeconfig to authenticate with cluster.

Ingress list : <URL>:<port>/ingress-sd
HTTP Routes list : <URL>:<port>/httproutes-sd

