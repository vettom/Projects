#!/usr/bin/env python3
import json
import requests
import time
from flask import Flask, jsonify

app = Flask(__name__)
@app.route("/")
def home_page():
        return 'Test landing page. It worked'

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
    Endpoints = {"cna": 'https://vettom.github.io/api/mixed.json', "pis": 'https://vettom.github.io/api/mixed.json' }
    while True:
        for App, Link in Endpoints.items():
            try:
                # For each App Link, process URL
                response = requests.get(Link)
                jsonData = response.json()
                promData = {
                    "targets": jsonData['result'],
                            "labels": {
                                "__meta_application": App,
                            }
                }
            except Exception as e:
                print('Error retrieving data:', e)
            try:
                # Write result to file
                with open(f"{App}.json", 'w') as file:
                    json.dump(promData, file)
                print('Data retrieved successfully')
            except Exception as e:
                print('Error writing data to json:', e)

        time.sleep(10)

if __name__ == '__main__':
    # Start the data retrieval in a separate thread
    import threading
    t = threading.Thread(target=fetch_data)
    t.start()

    # Start the HTTP server
    app.run(port=8085)
