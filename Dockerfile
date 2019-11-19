FROM alpine:latest as builder

RUN apk --update add ca-certificates

# temporary folder for application to read/write files
RUN mkdir -p /tmp /logs


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chown=2000:2000 /logs /logs
COPY --from=builder --chown=2000:2000 /tmp /tmp

ADD cloud-connector-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/cloud-connector-service","-isHealthy"]

USER 2000
ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/cloud-connector-service"]
