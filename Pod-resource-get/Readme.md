# Pod Resource monitoring.

## Summary
 Purpose of this pod is to retrieve Pod's resource information from KubeAPI based on provided *label* and return it as a list.

## Installation
```bash
  # Helm deploy
  helm install podman ./podmon-helm

  # Kubectl deploy
  kubectl apply -f podmon-kubectl/podmon.yml

```

## Building image
```bash
  cd Build
  pip3 freeze > requirements.txt
  docker buildx build -t {{REPO}}/podmonitor_amd:latest -o type=image --platform=linux/amd64 .
  docker scan {{REPO}}/podmonitor_amd
  docker push {{REPO}}/podmonitor_amd:latest

```

*Testing*
```bash
	curl -L podmon-svc.monitoring.svc.cluster.local:8000/container-resources?pod-label=app.kubernetes.io%2Fcomponent%3Dtesting

```
* Sample output
```json
        [
      {
        "container_name": "jenkins-container",
        "pod_name": "automation",
        "namespace": "jenkinspod",
        "mem_limit": "128Mi",
        "mem_req": "64Mi",
        "cpu_limit": "500m",
        "cpu_req": "250m"
      }
    ]
```

# Design
- A dedicated namespace called **monitoring** created
- Create cluster role with [list, get ] permissions on pods
- Cluster role binding mapping in **monitoring** namespace
- Deploy Pod to **monitoring** namespace
	- Python script with Kubernetes module
	- Flask app to front end
- Create Cluster IP service for the pod


# About
 - Applications is written in python using **kubernetes** module. 
 - Cluster role  allowing pods in *monitoring* namespace to query kube-api
 - Flask used as front end.