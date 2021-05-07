FROM golang:1.15.10-alpine as builder
COPY . /go/src/github.com/frodenas/gcs-resource
ENV CGO_ENABLED 0
RUN go build -o /assets/in github.com/frodenas/gcs-resource/cmd/in
RUN go build -o /assets/out github.com/frodenas/gcs-resource/cmd/out
RUN go build -o /assets/check github.com/frodenas/gcs-resource/cmd/check

FROM alpine:edge AS resource
RUN apk add --no-cache bash tzdata ca-certificates unzip zip gzip tar
COPY --from=builder assets/ /opt/resource/
RUN chmod +x /opt/resource/*

FROM resource
