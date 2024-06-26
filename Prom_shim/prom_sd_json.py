#!/usr/bin/env python3
import json,requests,time,os
from flask import Flask,Response,render_template
from prometheus_flask_exporter import PrometheusMetrics
from prometheus_client import Counter, Gauge

prometheus_sd_failed_configs = Gauge('prometheus_sd_failed_configs', 'Failure to fetch from endpoint',['app_name'])
prometheus_sd_failed_total = Counter('prometheus_sd_failed', 'Total failed requests',['app_name'])
prometheus_sd_requests_total = Counter('prometheus_sd_requests', 'Total requests for SD',['app_name'])
metrics = PrometheusMetrics(app=None, path='/metrics', export_defaults = False)

app = Flask(__name__)

@app.route("/")
def home_page():
    global Endpoints
    Message = { "Endpoints Configured" : Endpoints }
    Data = "Extra text"
    json_data = json.dumps(Message, indent=2)
    # return Response(json_data, Data, mimetype='application/json')
    return  render_template("index.html",Endpoints=json_data )

@app.route('/api/v1/<appname>')
def sd_data(appname):
    # Increment request count
    prometheus_sd_requests_total.labels(appname).inc()
    app_endpoint = get_endpoint(appname)
    if app_endpoint is None:
        #  Report fail for endpoint if not configured
        prometheus_sd_failed_configs.labels(appname).set(1)
        Message = "ERROR: no matching entry found in configuration file for " + appname
        return Message, 404
    # Call respective function based on appname
    if appname == "site1":
        data = fetch_endpoint_site1(appname, app_endpoint)
        json_data = json.dumps(data, indent=2)
        return Response(json_data, mimetype='application/json')
    elif appname == "site2":
        data = fetch_endpoint_site2(appname, app_endpoint)
        json_data = json.dumps(data, indent=2)
        return Response(json_data, mimetype='application/json')
    else:
        # Set fail if no function defined.
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

def fetch_endpoint_site1(appname,Link):
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

def fetch_endpoint_site2(appname,Link):
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
    # try:
    #     with open("/config/endpoints.json", 'r') as file:
    #         Endpoints = json.load(file)
    # except Exception as e:
    #     print(f"ERROR: Failed to load endpoints.json. \n {e}")
    #     exit(1)
    # Start the HTTP server
    Endpoints = {"site1": 'https://vettom.github.io/api/site1.json', "site2": 'https://vettom.github.io/api/site2.json', }
    metrics.init_app(app)
    app.run(host='0.0.0.0',port=8080)
