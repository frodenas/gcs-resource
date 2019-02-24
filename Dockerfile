FROM alpine as certs
RUN apk update && apk add ca-certificates

FROM busybox:1.30.1
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

ADD assets/ /opt/resource/
