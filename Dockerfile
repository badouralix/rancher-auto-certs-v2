FROM golang:1.16-alpine AS builder
# See https://github.com/go-acme/lego/issues/946
ENV GO111MODULE=on
RUN apk add --no-cache git
RUN go get -v github.com/badouralix/rancher-auto-certs-v2

FROM alpine:3.13
COPY --from=builder /go/bin/rancher-auto-certs-v2 /usr/local/bin/rancher-auto-certs-v2
RUN apk add --no-cache ca-certificates
USER 101:101
VOLUME [ "/config" ]
CMD [ "rancher-auto-certs-v2" ]
