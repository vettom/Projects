# Use the official Python base image with Alpine Linux
FROM python:3.9-alpine

# Set the working directory in the container
WORKDIR /app

# Copy the requirements file to the working directory
COPY requirements.txt .

# Install the Python dependencies
RUN pip install --no-cache-dir -r requirements.txt && \
    apk add curl

# Copy the application code to the working directory
COPY . /app/
COPY templates /app/

# Expose the desired port (change if necessary)
EXPOSE 8085

# Set the entry point command to run the Flask application
CMD ["python", "prom_sd.py"]
