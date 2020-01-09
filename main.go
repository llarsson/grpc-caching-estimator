package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/llarsson/grpc-caching-estimator/hipstershop"
)

const (
	productCatalogServiceAddrKey = "PRODUCT_CATALOG_SERVICE_ADDR"
	currencyServiceAddrKey       = "CURRENCY_SERVICE_ADDR"
	cartServiceAddrKey           = "CART_SERVICE_ADDR"
	recommendationServiceAddrKey = "RECOMMENDATION_SERVICE_ADDR"
	shippingServiceAddrKey       = "SHIPPING_SERVICE_ADDR"
	checkoutServiceAddrKey       = "CHECKOUT_SERVICE_ADDR"
	adServiceAddrKey             = "AD_SERVICE_ADDR"
	paymentSerivceAddrKey        = "PAYMENT_SERVICE_ADDR"
	emailServiceAddrKey          = "EMAIL_SERVICE_ADDR"
)

// ValidityEstimator estimates validity for request/response pairs
type ValidityEstimator interface {
	EstimateMaxAge(fullMethod string, req interface{}, resp interface{}) (int, error)
	CreateUnaryInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)
}

// SimplisticValidityEstimator is rather simplistic in its operation
type SimplisticValidityEstimator struct {
}

// EstimateMaxAge Estimates the max age of the specified request/response pair for the given method
func (estimator *SimplisticValidityEstimator) EstimateMaxAge(fullMethod string, req interface{}, resp interface{}) (int, error) {
	value, present := os.LookupEnv("PROXY_MAX_AGE")

	if !present {
		// It is not an error to not have the proxy max age key present in environment. We just act as if we were in passthrough mode.
		return -1, nil
	}

	switch value {
	case "dynamic":
		{
			return -1, status.Errorf(codes.Unimplemented, "Dynamic validity not implemented yet")
		}
	case "passthrough":
		{
			return -1, nil
		}
	default:
		maxAge, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Failed to parse PROXY_MAX_AGE (%s) into integer", value)
			return -1, err
		}
		return maxAge, nil
	}
}

// CreateUnaryInterceptor Creates the gRPC Unary interceptor
func (estimator *SimplisticValidityEstimator) CreateUnaryInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	validityEstimationInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			log.Printf("Upstream call failed with error %v", err)
			return resp, err
		}

		maxAge, err := estimator.EstimateMaxAge(info.FullMethod, req, resp)
		if err == nil && maxAge > 0 {
			grpc.SetHeader(ctx, metadata.Pairs("cache-control", fmt.Sprintf("must-revalidate, max-age=%d", maxAge)))
		}

		return resp, err
	}

	return validityEstimationInterceptor
}

func main() {
	port, err := strconv.Atoi(os.Getenv("PROXY_LISTEN_PORT"))
	if err != nil {
		log.Fatalf("PROXY_LISTEN_PORT cannot be parsed as integer")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Fatalf("Failed to register ocgrpc server views: %v", err)
	}

	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatalf("Failed to register ocgrpc client views: %v", err)
	}

	estimator := new(SimplisticValidityEstimator)

	grpcServer := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}), grpc.UnaryInterceptor(estimator.CreateUnaryInterceptor()))

	serviceAddrKeys := []string{productCatalogServiceAddrKey, currencyServiceAddrKey,
		cartServiceAddrKey, recommendationServiceAddrKey, shippingServiceAddrKey,
		checkoutServiceAddrKey, adServiceAddrKey, paymentSerivceAddrKey, emailServiceAddrKey}

	for _, serviceAddrKey := range serviceAddrKeys {
		upstreamAddr, ok := os.LookupEnv(serviceAddrKey)
		if !ok {
			continue
		}

		conn, err := grpc.Dial(upstreamAddr, grpc.WithInsecure(), grpc.WithStatsHandler(new(ocgrpc.ClientHandler)))
		if err != nil {
			log.Fatalf("Cannot connect to upstream %s : %v", serviceAddrKey, err)
		}
		defer conn.Close()

		switch serviceAddrKey {
		case productCatalogServiceAddrKey:
			{
				proxy := pb.ProductCatalogServiceProxy{Client: pb.NewProductCatalogServiceClient(conn)}
				pb.RegisterProductCatalogServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Product Catalog Service calls to %s", upstreamAddr)
			}
		case currencyServiceAddrKey:
			{
				proxy := pb.CurrencyServiceProxy{Client: pb.NewCurrencyServiceClient(conn)}
				pb.RegisterCurrencyServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Currency Service calls to %s", upstreamAddr)
			}
		case cartServiceAddrKey:
			{
				proxy := pb.CartServiceProxy{Client: pb.NewCartServiceClient(conn)}
				pb.RegisterCartServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Cart Service calls to %s", upstreamAddr)
			}
		case recommendationServiceAddrKey:
			{
				proxy := pb.RecommendationServiceProxy{Client: pb.NewRecommendationServiceClient(conn)}
				pb.RegisterRecommendationServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Recommendation Service calls to %s", upstreamAddr)
			}
		case shippingServiceAddrKey:
			{
				proxy := pb.ShippingServiceProxy{Client: pb.NewShippingServiceClient(conn)}
				pb.RegisterShippingServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Shipping Service calls to %s", upstreamAddr)
			}
		case checkoutServiceAddrKey:
			{
				proxy := pb.CheckoutServiceProxy{Client: pb.NewCheckoutServiceClient(conn)}
				pb.RegisterCheckoutServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Checkout Service calls to %s", upstreamAddr)
			}
		case adServiceAddrKey:
			{
				proxy := pb.AdServiceProxy{Client: pb.NewAdServiceClient(conn)}
				pb.RegisterAdServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Ad Service calls to %s", upstreamAddr)
			}
		case paymentSerivceAddrKey:
			{
				proxy := pb.PaymentServiceProxy{Client: pb.NewPaymentServiceClient(conn)}
				pb.RegisterPaymentServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Payment Service calls to %s", upstreamAddr)
			}
		case emailServiceAddrKey:
			{
				proxy := pb.EmailServiceProxy{Client: pb.NewEmailServiceClient(conn)}
				pb.RegisterEmailServiceServer(grpcServer, &proxy)
				log.Printf("Proxying Email Service calls to %s", upstreamAddr)
			}
		default:
			{
				log.Fatalf("This should never happen")
			}
		}

	}

	grpcServer.Serve(lis)
}
