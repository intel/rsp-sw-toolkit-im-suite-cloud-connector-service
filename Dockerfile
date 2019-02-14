FROM amr-registry.caas.intel.com/rrp/rrp-scratch:latest
ADD cloud-connector-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/cloud-connector-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/cloud-connector-service"]
