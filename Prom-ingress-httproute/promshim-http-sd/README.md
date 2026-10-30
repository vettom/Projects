# promshim-http-sd

Prometheus HTTP service discovery shim application written in Go. It retrieves Kubernetes Ingress and Gateway API HTTPRoute resources across all namespaces and exposes them in the Prometheus HTTP Service Discovery format.

## Endpoints

- `GET /` - Overview page listing available discovery endpoints and a table of discovered HTTP routes with clickable links
- `GET /ingress-sd` - Prometheus HTTP service discovery targets derived from Kubernetes Ingress resources
- `GET /httproutes-sd` - Prometheus HTTP service discovery targets derived from Gateway API HTTPRoute resources
- `GET /healthz` - Health check endpoint (returns `{"status":"ok"}`)

## Requirements

The application runs inside a Kubernetes cluster using in-cluster configuration (or locally using `~/.kube/config`).
The Pod's ServiceAccount requires RBAC permissions to:

- `get`, `list`, `watch` on `networking.k8s.io/ingresses`
- `get`, `list` on `gateway.networking.k8s.io/httproutes`

## Local Development & Testing

Run unit tests:

```bash
go test -v ./...
```

Build binary:

```bash
go build -o promshim-http-sd .
```

Run locally (uses your `~/.kube/config`):

```bash
./promshim-http-sd
```

## Container Build

The application uses a multi-stage Dockerfile to produce a minimal, secure container image:

```bash
# Build with Docker or Podman
VERSION="v1.0.4"
podman build --arch amd64 -t dennysv/promshim-http-sd:$VERSION .
podman tag dennysv/promshim-http-sd:$VERSION dennysv/promshim-http-sd:latest
podman push dennysv/promshim-http-sd:$VERSION
podman push dennysv/promshim-http-sd:latest

# ARM64
podman build --arch arm64 -t dennysv/promshim-http-sd-arm:$VERSION .
podman tag dennysv/promshim-http-sd-arm:$VERSION dennysv/promshim-http-sd-arm:latest
podman push dennysv/promshim-http-sd-arm:$VERSION
podman push dennysv/promshim-http-sd-arm:latest
```

### Build and Push to ECR

```bash
export AWS_PROFILE=cor-audio
aws ecr get-login-password --profile labs--region eu-west-2 | podman login --username AWS --password-stdin <aws-account-number>.dkr.ecr.eu-west-2.amazonaws.com
podman build --arch amd64 . -t <aws-account-number>.dkr.ecr.eu-west-2.amazonaws.com/promshim-http-sd:v<version>
podman push <aws-account-number>.dkr.ecr.eu-west-2.amazonaws.com/promshim-http-sd:<version>
```
