#!/usr/bin/env python3
import json,requests,yaml
from flask import Flask,Response
from prometheus_flask_exporter import PrometheusMetrics
from prometheus_client import Counter, Gauge

prometheus_sd_failed_configs = Gauge('prometheus_sd_failed_configs', 'Failure to fetch from endpoint',['app_name'])
prometheus_sd_failed_total = Counter('prometheus_sd_failed', 'Total failed requests',['app_name'])
prometheus_sd_requests_total = Counter('prometheus_sd_requests', 'Total requests for SD',['app_name'])
metrics = PrometheusMetrics(app=None, path='/metrics', export_defaults = False)

app = Flask(__name__)

@app.route('/', methods=['GET'])
# For health check  and return configured endpoints
def home_page():
    global Endpoints
    yaml_data = yaml.dump(Endpoints, sort_keys=False, indent=2)
    return Response(yaml_data,mimetype='text/plain')


@app.route('/api/v1/<appname>')
def sd_data(appname):
    prometheus_sd_requests_total.labels(appname).inc()
    app_endpoint = get_endpoint(appname)
    if app_endpoint is None:
        prometheus_sd_failed_configs.labels(appname).set(1)
        Message = "ERROR: no matching entry found in configuration file for " + appname
        return Message, 404
    # Call respective function based on appname
    if appname == "app1":
        data = fetch_endpoint_app1(appname, app_endpoint)
        json_data = json.dumps(data, indent=2)
        return Response(json_data, mimetype='application/json')
    if appname == "app2":
        data = fetch_endpoint_app2(appname, app_endpoint)
        json_data = json.dumps(data, indent=2)
        return Response(json_data, mimetype='application/json')
    else:
        prometheus_sd_failed_configs.labels(appname).set(1)
        Message = "ERROR: Please add function for "  + appname
        return Message, 404

def get_endpoint(appname):
    global Endpoints
    try:
        for App, Link in Endpoints.items():
            if appname == App:
                return Link
    except:
        return None

def fetch_endpoint_app1(appname,Link):
    try:
        response = requests.get(Link)
        jsonData = response.json()
        promData = {
            "targets": jsonData['result'],
                    "labels": {
                        "__meta_application": appname,
                    }
        }
    except Exception as e:
        print(f"ERROR: Failed to retrieve data for {appname} and link: {Link}")
        promData = "ERROR: Failed to retrieve data for " +  appname + " and link:" + Link
        prometheus_sd_failed_configs.labels(appname).set(1)
    return promData

def fetch_endpoint_app2(appname,Link):
    try:
        response = requests.get(Link)
        jsonData = response.json()
        promData = {
            "targets": jsonData['result'],
                    "labels": {
                        "__meta_application": appname,
                    }
        }
    except Exception as e:
        print(f"ERROR: Failed to retrieve data for {appname} and link: {Link}")
        promData = "ERROR: Failed to retrieve data for " +  appname + " and link:" + Link
        prometheus_sd_failed_configs.labels(appname).set(1)
    return promData

if __name__ == '__main__':
    try:
        yaml_file = '/config/endpoints.yaml'
        with open(yaml_file, 'r') as file:
            Endpoints = yaml.safe_load(file)
        print(Endpoints)
    except Exception as e:
        print(f"ERROR: Failed to load endpoints.yaml. \n {e}")
        exit(1)
    # Start the HTTP server
    metrics.init_app(app)
    app.run(host='0.0.0.0',port=8080)