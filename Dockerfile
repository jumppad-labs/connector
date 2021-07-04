FROM alpine:latest as base

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

# Copy AMD binaries
FROM base AS image-amd64-

COPY connector_linux_amd64/connector /connector
RUN chmod +x /connector


# Copy Arm 8 binaries
FROM base AS image-arm64-

COPY connector_linux_arm64/connector /connector
RUN chmod +x /connector

# Copy Arm 7 binaries
FROM base AS image-arm-v7

COPY connector_linux_arm_7/connector /connector
RUN chmod +x /connector

# Copy Arm 6 binaries
FROM base AS image-arm-v6

COPY connector_linux_arm_6/connector /connector
RUN chmod +x /connector

FROM image-${TARGETARCH}-${TARGETVARIANT}

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG BUILDPLATFORM
ARG BUILDARCH

RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM $TARGETARCH $TARGETVARIANT"  