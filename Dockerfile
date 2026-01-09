FROM gcr.io/distroless/static
ARG TARGETPLATFORM
ENTRYPOINT ["/usr/bin/server"]
COPY $TARGETPLATFORM/server /usr/bin/