FROM hub.docker.intel.com/rrp/scratch-ca-certs:1.0
ADD cloud-connector-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/cloud-connector-service","-isHealthy"]
ENTRYPOINT ["/cloud-connector-service"]
