# Summary
Shim for custom prometheus service discovery based on domins list provided at endpoint.
# Input
Requires /config/endpoints.json in format below
```json
    {
        // "Project or App": "Endpint URL to scrape"
        site1: "https://vettom.github.io/api/site1.json",
        site2: "https://vettom.github.io/api/site2.json"
    }
```
# Building
docker build -t prom_sd_shim .
dk tag prom_sd_shim dennysv/prom_sd_shim:latest
docker push dennysv/prom_sd_shim:latest

# Generating requirements.txt
python -m venv venv
source venv/bin/activate
pip install prometheus_client  prometheus_flask_exporter flask requests pyyaml
pip freeze > requirements.txt
deactivate
