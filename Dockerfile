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

# Cadence server
FROM alpine AS cadence-notification-server

ENV CADENCE_HOME /etc/cadence-notification
RUN mkdir -p /etc/cadence-notification

COPY --from=builder /cadence-notification/cadence-notification /usr/local/bin
COPY --from=builder /cadence-notification/config /etc/cadence-notification/config

WORKDIR /etc/cadence-notification

ENV SERVICES="notifier,receiver"
CMD ["cadence-notification", "start"]
