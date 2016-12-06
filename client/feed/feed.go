package feed

import (
	"time"

	"github.com/buptmiao/microservice-app/proto/feed"
	"github.com/buptmiao/microservice-app/util"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	jujuratelimit "github.com/juju/ratelimit"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type FeedClient struct {
	GetFeedsEndpoint   endpoint.Endpoint
	CreateFeedEndpoint endpoint.Endpoint
}

func (f FeedClient) GetFeeds(ctx context.Context, in *feed.GetFeedsRequest, opts ...grpc.CallOption) (*feed.GetFeedsResponse, error) {
	resp, err := f.GetFeedsEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*feed.GetFeedsResponse), nil
}

func (f FeedClient) CreateFeed(ctx context.Context, in *feed.FeedRecord, opts ...grpc.CallOption) (*feed.OkResponse, error) {
	resp, err := f.CreateFeedEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*feed.OkResponse), nil
}

func NewFeedClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) feed.FeedClient {

	limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))

	var getFeedsEndpoint endpoint.Endpoint
	{
		getFeedsEndpoint = grpctransport.NewClient(
			conn,
			"feed.Feed",
			"GetFeeds",
			util.DummyEncode,
			util.DummyDecode,
			feed.GetFeedsResponse{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		getFeedsEndpoint = opentracing.TraceClient(tracer, "GetFeeds")(getFeedsEndpoint)
		getFeedsEndpoint = limiter(getFeedsEndpoint)
		getFeedsEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetFeeds",
			Timeout: 5 * time.Second,
		}))(getFeedsEndpoint)
	}

	var createFeedEndpoint endpoint.Endpoint
	{
		createFeedEndpoint = grpctransport.NewClient(
			conn,
			"feed.Feed",
			"CreateFeed",
			util.DummyEncode,
			util.DummyDecode,
			feed.OkResponse{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		createFeedEndpoint = opentracing.TraceClient(tracer, "CreateFeed")(createFeedEndpoint)
		createFeedEndpoint = limiter(createFeedEndpoint)
		createFeedEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "CreateFeed",
			Timeout: 5 * time.Second,
		}))(createFeedEndpoint)
	}

	return FeedClient{
		GetFeedsEndpoint:   getFeedsEndpoint,
		CreateFeedEndpoint: createFeedEndpoint,
	}
}
