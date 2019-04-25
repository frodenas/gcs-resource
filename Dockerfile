FROM alpine:edge
RUN apk add --no-cache bash tzdata ca-certificates unzip zip gzip tar

ADD assets/ /opt/resource/
