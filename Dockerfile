FROM scratch
COPY taskctl /
ENTRYPOINT ["/taskctl"]
