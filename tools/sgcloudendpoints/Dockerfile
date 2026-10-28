ARG esp_version=2
FROM gcr.io/endpoints-release/endpoints-runtime-serverless:${esp_version}

# The CA bundle is pre-filtered by sage to distrust the eMudhra intermediate CAs with compromised keys,
# and the roots chaining to them. See https://emudhra.we-have-a-problem.com/
# The base image is distroless (no shell, no Python), so the bundle can't be filtered in the image build.
COPY --chown=root:root --chmod=644 ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV ENDPOINTS_SERVICE_PATH=/etc/endpoints/service.json
COPY --chown=nonroot:nonroot --chmod=755 ./service.json ${ENDPOINTS_SERVICE_PATH}
