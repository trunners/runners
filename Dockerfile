FROM scratch
ARG TARGETPLATFORM
ENTRYPOINT ["/usr/bin/server"]
COPY $TARGETPLATFORM/server /usr/bin/