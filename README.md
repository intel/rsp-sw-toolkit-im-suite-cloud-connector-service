# Intel® Inventory Suite Cloud-Connector-service
[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)

Cloud connector service is a microservice for the Intel® Inventory Suite that provides a link between applications within the same network environment(on-promise) and external/cloud endpoints with secure capabilities. 

# Install and Deploy via Docker Container #

### Prerequisites ###
- Docker & make: 
```
sudo apt install -y docker.io build-essential
```

### Installation ###

Clone this repo and run:
```
sudo make build deploy
```

### API Documentation ###

Go to [https://editor.swagger.io](https://editor.swagger.io) and import cloud-connector-service.yml file.
