# Summary
Shim for custom prometheus service discovery based on domins list provided at endpoint.

## Access App
<URL>/api/v1/app1

# Input
Requires /config/endpoints.yaml in format below
```yaml
        "app1": "https://vettom.github.io/api/app1.json",
        "app2": "https://vettom.github.io/api/app2.json"
```


# Building
docker build --platform linux/amd64 -t prom_sd_shim .
dk tag prom_sd_shim dennysv/prom_sd_shim:v1.0.4
docker push dennysv/prom_sd_shim:v1.0.4

# Generating requirements.txt
python -m venv venv
source venv/bin/activate
pip install prometheus_client  prometheus_flask_exporter flask requests pyyaml
pip freeze > requirements.txt
deactivate
