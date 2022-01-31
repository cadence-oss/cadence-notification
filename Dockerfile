# Build cadence-notification binaries
FROM golang:1.17-alpine3.13 AS builder

ARG RELEASE_VERSION

RUN apk add --update --no-cache ca-certificates make git curl mercurial unzip

WORKDIR /cadence-notification

# Making sure that dependency is not touched
ENV GOFLAGS="-mod=readonly"

# Copy go mod dependencies and build cache
COPY go.* ./
RUN go mod download

COPY . .
RUN rm -fr .bin .build

ENV CADENCE_NOTIFICATION_RELEASE_VERSION=$RELEASE_VERSION

RUN CGO_ENABLED=0 make cadence-notification

# Download dockerize
FROM alpine:3.11 AS dockerize

RUN apk add --no-cache openssl

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && echo "**** fix for host id mapping error ****" \
    && chown root:root /usr/local/bin/dockerize

# Alpine base image
FROM alpine:3.11 AS alpine

RUN apk add --update --no-cache ca-certificates tzdata bash curl

# set up nsswitch.conf for Go's "netgo" implementation
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-424546457
RUN test ! -e /etc/nsswitch.conf && echo 'hosts: files dns' > /etc/nsswitch.conf

SHELL ["/bin/bash", "-c"]

# Cadence-notification server
FROM alpine AS cadence-notification

ENV CADENCE_NOTIFICATION_HOME /etc/cadence-notification
RUN mkdir -p /etc/cadence-notification

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin
COPY --from=builder /cadence-notification /usr/local/bin

WORKDIR /etc/cadence-notification

ENV SERVICES="notifier,receiver"

EXPOSE 7933 7934 7935 7939

CMD /cadence-notification start

FROM cadence-notification
