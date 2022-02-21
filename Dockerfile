# Build cadence-notification binaries
FROM golang:1.17-alpine3.13 AS builder

ARG RELEASE_VERSION

RUN apk add --update --no-cache ca-certificates make git curl mercurial unzip
RUN apk add build-base

WORKDIR /cadence-notification

# Making sure that dependency is not touched
ENV GOFLAGS="-mod=readonly"

# Copy go mod dependencies and build cache
COPY go.* ./
RUN go mod download

COPY . .
RUN rm -fr .bin .build 

ENV CADENCE_NOTIFICATION_RELEASE_VERSION=$RELEASE_VERSION

RUN CGO_ENABLED=0 make bins

# Download dockerize
FROM alpine:3.11 AS dockerize

RUN apk add --no-cache openssl

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && echo "**** fix for host id mapping error ****" \
    && chown root:root /usr/local/bin/dockerize

RUN apk add --update --no-cache ca-certificates tzdata bash curl
RUN test ! -e /etc/nsswitch.conf && echo 'hosts: files dns' > /etc/nsswitch.conf
SHELL [ "/bin/bash", "-c" ]

# Cadence server
FROM alpine AS cadence-notification-server

ENV CADENCE_NOTIFICATION_HOME=/etc/cadence-notification
RUN mkdir -p /etc/cadence-notification

COPY --from=builder /cadence-notification/cadence-notification /usr/local/bin
COPY --from=builder /cadence-notification/config /etc/cadence-notification/config
COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin

COPY /config/config_template.yaml /etc/cadence-notification/config
COPY /start.sh /start.sh

WORKDIR /etc/cadence-notification

ENV SERVICES="notifier,receiver"
RUN chmod +x /start.sh
CMD /start.sh
