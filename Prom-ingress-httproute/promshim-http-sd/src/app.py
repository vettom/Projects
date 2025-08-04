# Prometheus shim application to expose Kubernetes Ingress resources as Prometheus service discovery targets
# This application runs inside a Kubernetes cluster and uses the in-cluster configuration to access the Kubernetes API.
# It retrieves all Ingress resources across all namespaces and formats them as Prometheus service discovery
from kubernetes import client, config
from flask import Flask, jsonify

app = Flask(__name__)

def get_ingress_targets():
    # Load in-cluster configuration
    load_kube_config()
    api = client.NetworkingV1Api()

    ingress_list = api.list_ingress_for_all_namespaces().items
    targets = []

    for ingress in ingress_list:
        namespace = ingress.metadata.namespace
        name = ingress.metadata.name
        rules = ingress.spec.rules or []

        for rule in rules:
            host = rule.host
            # This section if ignoring the paths in ingress rules.
            targets.append({
                "targets": [host],
                "labels": {
                    "namespace": namespace,
                    "ingress": name
                }
            })            

            # Uncomment the following section if you want to include paths in the targets
            paths = rule.http.paths if rule.http else []
            # for path_obj in paths:
            #     path = path_obj.path or "/"
            #     url = f"https://{host}{path}"  
            #     targets.append({
            #         "targets": [url],
            #         "labels": {
            #             "namespace": namespace,
            #             "ingress": name
            #         }
            #     })
    return targets

def get_http_route_target():
    load_kube_config()

    # Create API instance
    api = client.CustomObjectsApi()
    http_routes = api.list_cluster_custom_object(group="gateway.networking.k8s.io",
                                             version="v1",
                                             plural="httproutes")
    hostnames = []
    for route in http_routes['items']:
        if "hostnames" in route["spec"]:
            for hostname in route["spec"]["hostnames"]:
                hostnames.append(hostname)
    promData = {
        "targets": list(set(hostnames)),
        "labels": {
            "__meta_application": "httproutes",
            "__meta_source": "prom_shim_http_routes",
            "__meta_origin": "eks_cluster",
        }
    }
    promHttpSD= [promData]
    return promHttpSD

# Function to load Kubernetes configuration. If running inside a cluster, it will use in-cluster config,
# otherwise it will load the kubeconfig file from the default location allowing local development.
def load_kube_config():
    try:
        config.load_incluster_config()
    except config.ConfigException:
        config.load_kube_config()

# Default route to check if the application is running
@app.route("/")
def index():
    return """
    <html>
        <body>
            <h1>Prometheus Ingress and httproutes Service Discovery.</h1>
            <h3> Available endpoints: </h3>
            <ul>
                <li><a href="/ingress-sd">/ingress-sd</a> - Get Ingress Service Discovery targets</li>
                <li><a href="/httproutes-sd">/httproutes-sd</a> - Get HTTP Routes Service Discovery targets</li>
            </ul>
        </body>
    </html>
    """

# Endpoints to retrieve Ingress service discovery targets
@app.route("/ingress-sd")
def ingress_sd():
    return jsonify(get_ingress_targets())
# Endpoint to retrieve HTTP Routes service discovery targets
@app.route("/httproutes-sd")
def httproute_sd():
    return jsonify(get_http_route_target())

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9113)
