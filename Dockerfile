FROM golang:latest AS build-env
RUN echo $GOPATH
ADD . /go/src/github.com/llarsson/grpc-caching-estimator
WORKDIR /go/src/github.com/llarsson/grpc-caching-estimator
RUN go get ./... && go build -o /grpc-caching-estimator
# Multi-stage!
FROM alpine
WORKDIR /app
COPY --from=build-env /grpc-caching-estimator /app/
ENTRYPOINT ./grpc-caching-estimator
