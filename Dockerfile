FROM golang:1.12-alpine AS builder
RUN apk add --no-cache git
RUN go get -v github.com/badouralix/rancher-auto-certs-v2

FROM alpine:3.9
COPY --from=builder /go/bin/rancher-auto-certs-v2 /usr/local/bin/rancher-auto-certs-v2
RUN apk add --no-cache ca-certificates
VOLUME [ "/config" ]
CMD [ "rancher-auto-certs-v2" ]
