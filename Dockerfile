FROM golang:1.11.3-alpine3.8 AS builder

RUN apk upgrade \
    && apk add git \
    && go get -ldflags '-w -s' \
        github.com/shadowsocks/go-shadowsocks2

FROM alpine:3.8

LABEL maintainer="mritd <mritd1234@gmail.com>"

RUN apk upgrade \
    && apk add bash tzdata \
    && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/go-shadowsocks2 /usr/bin/shadowsocks

ENTRYPOINT ["shadowsocks"]
