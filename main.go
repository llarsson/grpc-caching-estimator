package main

import (
	"errors"
	"fmt"
	"log"
	"github.com/mwitkow/grpc-proxy/proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"net"
	"os"
)

var (
	director proxy.StreamDirector
)

func CacheEstimatingStreamDirector(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
	_, ok := metadata.FromIncomingContext(ctx)
	if ok {
		conn, err := grpc.DialContext(ctx,
			os.Getenv("PROXY_UPSTREAM_HOST")+":"+os.Getenv("PROXY_UPSTREAM_PORT"),
			grpc.WithCodec(proxy.Codec()),
			grpc.WithInsecure())

		EstimateMaxAge(ctx, fullMethodName)

		return ctx, conn, err
	}

	return nil, nil, grpc.Errorf(codes.Unimplemented, "Unknown method")
}

func EstimateMaxAge(ctx context.Context, fullMethodName string) {
	md, _ := metadata.FromIncomingContext(ctx)

	log.Println(fmt.Sprintf("%s :: %s", fullMethodName, md))

	switch os.Getenv("PROXY_MAX_AGE") {
	case "dynamic":
		{
			maxAge, err := estimatedValidity(ctx, fullMethodName)
			if err != nil {
				panic(err)
			}
			grpc.SetHeader(ctx, metadata.Pairs("cache-control", "must-revalidate, max-age="+maxAge))
		}
	case "passthrough":
		{
		}
	default:
		grpc.SetHeader(ctx, metadata.Pairs("cache-control", "must-revalidate, max-age="+os.Getenv("PROXY_MAX_AGE")))
	}
}

func estimatedValidity(ctx context.Context, fullMethodName string) (string, error) {
	return "", errors.New("Not implemented yet")
}

func main() {
	director = CacheEstimatingStreamDirector

	server := grpc.NewServer(grpc.CustomCodec(proxy.Codec()), grpc.UnknownServiceHandler(proxy.TransparentHandler(director)))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("PROXY_LISTEN_PORT")))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening to %s\n", os.Getenv("PROXY_LISTEN_PORT"))

	server.Serve(lis)
}
