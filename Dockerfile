FROM alpine:latest
COPY taskctl /
ENTRYPOINT ["/taskctl"]
