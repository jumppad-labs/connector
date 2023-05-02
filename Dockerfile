FROM alpine:latest as base

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG BUILDPLATFORM
ARG BUILDARCH

RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM $TARGETARCH $TARGETVARIANT"  

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY linux_${TARGETARCH}/connector /connector
RUN chmod +x /connector