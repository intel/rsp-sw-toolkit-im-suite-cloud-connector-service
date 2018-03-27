FROM 280211473891.dkr.ecr.us-west-2.amazonaws.com/scratch-ca-certs:latest
ADD cloud-connector-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/cloud-connector-service","-isHealthy"]
ENTRYPOINT ["/cloud-connector-service"]
