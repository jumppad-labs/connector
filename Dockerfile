FROM alpine:latest as base

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

# Copy AMD binaries
FROM base AS image-amd64-

COPY connector_linux_amd64/connector /connector

# Copy Arm binaries
FROM base AS image-arm-v7

COPY connector_linux_arm_7/connector /connector

FROM image-${TARGETARCH}-${TARGETVARIANT}

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG BUILDPLATFORM
ARG BUILDARCH

RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM $TARGETARCH $TARGETVARIANT"  

ENTRYPOINT [ "/connector" ]
