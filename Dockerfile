FROM alpine:latest

COPY /connector /connector

ENTRYPOINT [ "/connector" ]