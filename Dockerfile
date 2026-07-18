FROM alpine:latest
ARG TARGETPLATFORM=.
COPY ${TARGETPLATFORM}/bin/taskctl /
ENTRYPOINT ["/taskctl"]
