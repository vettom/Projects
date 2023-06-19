#!/usr/bin/env python3
import json,requests,time,os
from flask import Flask, jsonify
from prometheus_flask_exporter import PrometheusMetrics
from prometheus_client import Counter, Gauge

prometheus_sd_failed_configs = Gauge('prometheus_sd_failed_configs', 'SD configuration status',['app_name'])
prometheus_sd_failed_total = Gauge('prometheus_sd_failed', 'Total failed requests',['app_name'])
metrics = PrometheusMetrics(app=None, path='/metrics', export_defaults = False)

app = Flask(__name__)
@app.route("/")
def home_page():
    return 'Prometheus Service Discovery'

@app.route('/api/v1/<appname>')
def sd_data(appname):
    try:
        SD_File = "data/" + appname + ".json"
        with open(SD_File, 'r') as file:
            data = json.load(file)
            return jsonify(data)
    except Exception as e:
        return "Page not found", 404


def fetch_data():
    Endpoints = {"cna": 'https://vettom.github.io/api/cna.json', "pis": 'https://vettom.github.io/api/pis.json' }
    # try:
    #     with open("/config/endpoints.json", 'r') as file:
    #         Endpoints = json.load(file)
    # except Exception as e:
    #     print(f"Failed to load endpoints.json.", e)
    while True:
        try:
            for App, Link in Endpoints.items():
                response = requests.get(Link)
                jsonData = response.json()
                promData = {
                    "targets": jsonData['result'],
                            "labels": {
                                "__meta_application": App,
                            }
                }
                prometheus_sd_failed_configs.labels(App).set(0)
                # Write result to file
                with open(f"data/{App}.json", 'w') as file:
                    json.dump(promData, file)
                print(f'INFO: Data retrieved for {App}')
        except Exception as e:
            prometheus_sd_failed_configs.labels(App).set(1)
            prometheus_sd_failed_total.labels(App).inc()
            print(f'ERROR: Failed to retrieve data for {App}:', e)
        time.sleep(5)
if __name__ == '__main__':
    if not os.path.exists("data"):
        os.makedirs("data")
    # Start the data retrieval in a separate thread
    import threading
    t = threading.Thread(target=fetch_data)
    t.start()

    # Start the HTTP server
    metrics.init_app(app)
    app.run(port=8080)
