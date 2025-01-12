FROM alpine:latest
COPY ./bin/taskctl /
ENTRYPOINT ["/taskctl"]
