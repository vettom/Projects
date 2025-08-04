FROM python:3.13.5-slim-bookworm

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY ./src/* /app

# Install any needed packages specified in requirements.txt
RUN pip install --trusted-host pypi.python.org -r requirements.txt

# Make port 9113 available to the world outside this container
EXPOSE 9113
# Run app.py when the container launches

CMD ["flask", "run", "--host=0.0.0.0", "--port=9113"]
