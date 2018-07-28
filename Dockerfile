FROM golang:1.10.3-alpine3.8 AS builder

RUN apk upgrade --update \
    && apk add git \
    && go get github.com/riobard/go-shadowsocks2

FROM alpine:3.8

LABEL maintainer="mritd <mritd1234@gmail.com>"

RUN apk upgrade --update \
    && apk add bash tzdata \
    && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/go-shadowsocks2 /usr/bin/go-shadowsocks2

CMD ["go-shadowsocks2"]
