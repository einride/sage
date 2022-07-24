ARG esp_version=2
FROM gcr.io/endpoints-release/endpoints-runtime-serverless:${esp_version}
USER root
ENV ENDPOINTS_SERVICE_PATH /etc/endpoints/service.json
COPY ./service.json ${ENDPOINTS_SERVICE_PATH}
RUN chown -R envoy:envoy ${ENDPOINTS_SERVICE_PATH} && chmod -R 755 ${ENDPOINTS_SERVICE_PATH}
USER envoy
ENTRYPOINT ["/env_start_proxy.py"]
