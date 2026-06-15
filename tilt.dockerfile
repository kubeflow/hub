FROM alpine:3.24

WORKDIR /

COPY devenv/bin/model-registry /model-registry

ENTRYPOINT ["/model-registry"]
