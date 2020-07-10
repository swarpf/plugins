ARG BUILDPLATFORM

FROM --platform=${BUILDPLATFORM:-linux/amd64} tonistiigi/xx:golang AS xgo
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:alpine AS build

ARG TARGETPLATFORM
ARG PLUGINNAME

RUN test -n "$PLUGINNAME"

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY https://goproxy.io
COPY --from=xgo / /

RUN go env

RUN apk --update --no-cache add \
    build-base \
    gcc \
    git \
    ca-certificates \
  && rm -rf /tmp/* /var/cache/apk/*

# Compile the cmd to a standalone binary
WORKDIR /app
COPY . .
RUN go mod vendor
RUN go build -ldflags "-s -w -extldflags '-static'" ./cmd/$PLUGINNAME


FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:latest

ARG PLUGINNAME

COPY --from=build /etc/ssl/certs/ca-certificates.crt \
     /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/$PLUGINNAME /plugin
ENTRYPOINT ["/plugin"]
