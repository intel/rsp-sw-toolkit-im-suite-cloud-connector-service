DISCONTINUATION OF PROJECT. 

This project will no longer be maintained by Intel.

This project has been identified as having known security escapes.

Intel has ceased development and contributions including, but not limited to, maintenance, bug fixes, new releases, or updates, to this project.  

Intel no longer accepts patches to this project.
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
