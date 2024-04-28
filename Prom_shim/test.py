from flask import Flask, Response
import yaml

app = Flask(__name__)

@app.route('/', methods=['GET'])
def get_yaml_data():
    # Specify the path to your YAML file
    yaml_file = './endpoints.yaml'

    # Read the YAML file
    with open(yaml_file, 'r') as file:
        data = yaml.safe_load(file)

    # Convert the data to YAML format with pretty printing
    yaml_data = yaml.dump(data, sort_keys=False, indent=2)

    # Return the YAML response
    return Response(yaml_data, mimetype='text/plain')

if __name__ == '__main__':
    app.run()
