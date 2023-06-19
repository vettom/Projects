# Summary
Shim for custom prometheus service discovery based on domins list provided at endpoint.
# Input
Requires /config/endpoints.json in format below
```json
    {
        // "Project or App": "Endpint URL to scrape"
        "cna": "https://example.com/api/cna.json",
        "pis": "https://github.io/api/pis.json"
    }
```
# Building
docker build -t prom_sd_shim .
dk tag prom_sd_shim dennysv/prom_sd_shim:0.1.3
docker push dennysv/prom_sd_shim:0.1.3

# Generating requirements.txt
python -m venv venv
source venv/bin/activate
pip install prometheus_client  prometheus_flask_exporter flask requests
pip freeze > requirements.txt
deactivate
