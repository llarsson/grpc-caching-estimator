FROM golang:latest AS build-env
# Build stand-alone (independent of libc implementation)
ENV CGO_ENABLED 0
ADD . /go/src/github.com/llarsson/grpc-caching-estimator
WORKDIR /go/src/github.com/llarsson/grpc-caching-estimator
RUN go get ./... && go build -o /grpc-caching-estimator
# Multi-stage!
FROM alpine
WORKDIR /
COPY --from=build-env /grpc-caching-estimator /usr/local/bin/
ENTRYPOINT grpc-caching-estimator
