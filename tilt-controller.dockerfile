FROM alpine:3.24

WORKDIR /

COPY devenv/bin/manager /manager

ENTRYPOINT ["/manager"]
