# Building
docker build -t flask-app .
dk tag flask-app dennysv/flask-app
docker push dennysv/flask-app


# Generating requirements.txt
python -m venv venv
source venv/bin/activate
pip install prometheus_client  prometheus_flask_exporter flask requests
pip freeze > requirements.txt
deactivate
