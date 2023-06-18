#!/usr/bin/env python3
import json,requests,time
from flask import Flask, jsonify
from prometheus_flask_exporter import PrometheusMetrics
from prometheus_client import Counter, Gauge, Histogram

failure_counter = Counter('app_failures_total', 'Total number of failures',['app_name'])
my_counter = Counter('my_success', 'Sample by denny',['app_name'])
metrics = PrometheusMetrics(app=None, path='/metrics', export_defaults = False)

app = Flask(__name__)
@app.route("/")
def home_page():
    return 'Prometheus Service Discovery Shim'

@app.route('/api/v1/<appname>')
def sd_data(appname):
    try:
        SD_File = appname + ".json"
        with open(SD_File, 'r') as file:
            data = json.load(file)
            return jsonify(data)
    except FileNotFoundError:
        return "Data file not found"

def fetch_data():
    # Endpoints = {"cna": 'https://vettom.github.io/api/cna.json', "pis": 'https://vettom.github.io/api/pis.json' }
    try:
        with open("endpoints.json", 'r') as file:
            Endpoints = json.load(file)
    except Exception as e:
        print(f"Failed to load endpoints.json.", e)

    print(Endpoints)
    while True:
        try:
            for App, Link in Endpoints.items():
                # For each App Link, process URL
                response = requests.get(Link)
                jsonData = response.json()
                promData = {
                    "targets": jsonData['result'],
                            "labels": {
                                "__meta_application": App,
                            }
                }
                my_counter.labels(App).inc()

                # Write result to file
                with open(f"{App}.json", 'w') as file:
                    json.dump(promData, file)
                print('Data retrieved successfully')
        except Exception as e:
            failure_counter.labels(App).inc()
            print(f'Error retrieving data for {App}:', e)
        time.sleep(5)

if __name__ == '__main__':
    # Start the data retrieval in a separate thread
    import threading
    t = threading.Thread(target=fetch_data)
    t.start()

    # Start the HTTP server
    metrics.init_app(app)
    app.run(port=8085)
