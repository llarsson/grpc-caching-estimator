package main

import (
	"fmt"
	"github.com/mwitkow/grpc-proxy/proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"strconv"
)

var (
	director proxy.StreamDirector
)

func EstimateMaxAge(ctx context.Context, fullMethodName string) (int, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	log.Println(fmt.Sprintf("%s :: %s", fullMethodName, md))

	switch os.Getenv("PROXY_MAX_AGE") {
	case "dynamic":
		{
			maxAge, err := estimatedValidity(ctx, fullMethodName)
			if err != nil {
				return -1, err
			}
			grpc.SetHeader(ctx, metadata.Pairs("cache-control", fmt.Sprintf("must-revalidate, max-age=%d", maxAge)))
			return maxAge, nil
		}
	case "passthrough":
		{
			return -1, nil
		}
	default:
		maxAge, err := strconv.Atoi(os.Getenv("PROXY_MAX_AGE"))
		if err != nil {
			log.Printf("Failed to parse PROXY_MAX_AGE (%s) into integer", os.Getenv("PROXY_MAX_AGE"))
			return -1, err
		}
		grpc.SetHeader(ctx, metadata.Pairs("cache-control", fmt.Sprintf("must-revalidate, max-age=%d", maxAge)))
		return maxAge, nil
	}
}

func estimatedValidity(ctx context.Context, fullMethodName string) (int, error) {
	return -1, status.Errorf(codes.Unimplemented, "Dynamic validity not implemented yet")
}

func main() {
	upstream := fmt.Sprintf("%s:%s", os.Getenv("PROXY_UPSTREAM_HOST"), os.Getenv("PROXY_UPSTREAM_PORT"))
	conn, err := grpc.Dial(upstream, grpc.WithCodec(proxy.Codec()), grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Failed to connect to upstream service at %s", upstream)
	}

	director = func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		_, err = EstimateMaxAge(ctx, fullMethodName)
		return ctx, conn, err
	}

	server := grpc.NewServer(grpc.CustomCodec(proxy.Codec()), grpc.UnknownServiceHandler(proxy.TransparentHandler(director)))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("PROXY_LISTEN_PORT")))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening to %s\n", os.Getenv("PROXY_LISTEN_PORT"))

	server.Serve(lis)
}
