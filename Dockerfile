FROM golang:1.12.4-alpine3.9 AS builder

RUN apk upgrade \
    && apk add git \
    && go get -ldflags '-w -s' \
        github.com/shadowsocks/go-shadowsocks2

FROM alpine:3.9 AS dist

LABEL maintainer="mritd <mritd@linux.com>"

RUN apk upgrade \
    && apk add tzdata \
    && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/go-shadowsocks2 /usr/bin/shadowsocks

ENTRYPOINT ["shadowsocks"]
