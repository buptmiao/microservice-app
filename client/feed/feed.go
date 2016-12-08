package feed

import (
	"io"
	"time"

	"github.com/buptmiao/microservice-app/proto/feed"
	"github.com/buptmiao/microservice-app/util"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/etcd"
	"github.com/go-kit/kit/sd/lb"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	jujuratelimit "github.com/juju/ratelimit"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var feedCli feed.FeedClient

func Init(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) {
	feedCli = NewFeedClient(conn, tracer, logger)
}

func InitWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) {
	feedCli = NewFeedClientWithSD(sdClient, tracer, logger)
}

func GetClient() feed.FeedClient {
	if feedCli == nil {
		panic("feed client is not be initialized!")
	}
	return feedCli
}

type FeedClient struct {
	GetFeedsEndpoint   endpoint.Endpoint
	CreateFeedEndpoint endpoint.Endpoint
}

func (f *FeedClient) GetFeeds(ctx context.Context, in *feed.GetFeedsRequest, opts ...grpc.CallOption) (*feed.GetFeedsResponse, error) {
	resp, err := f.GetFeedsEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*feed.GetFeedsResponse), nil
}

func (f *FeedClient) CreateFeed(ctx context.Context, in *feed.FeedRecord, opts ...grpc.CallOption) (*feed.OkResponse, error) {
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

	return &FeedClient{
		GetFeedsEndpoint:   getFeedsEndpoint,
		CreateFeedEndpoint: createFeedEndpoint,
	}
}

func MakeGetFeedsEndpoint(f feed.FeedClient) endpoint.Endpoint {
	return f.(*FeedClient).GetFeedsEndpoint
}

func MakeCreateFeedEndpoint(f feed.FeedClient) endpoint.Endpoint {
	return f.(*FeedClient).CreateFeedEndpoint
}

func NewFeedClientWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) feed.FeedClient {
	res := &FeedClient{}

	factory := FeedFactory(MakeGetFeedsEndpoint, tracer, logger)
	subscriber, _ := etcd.NewSubscriber(sdClient, "/services/feed", factory, logger)
	balancer := lb.NewRoundRobin(subscriber)
	retry := lb.Retry(3, time.Second, balancer)
	res.GetFeedsEndpoint = retry

	factory = FeedFactory(MakeCreateFeedEndpoint, tracer, logger)
	subscriber, _ = etcd.NewSubscriber(sdClient, "/services/feed", factory, logger)
	balancer = lb.NewRoundRobin(subscriber)
	retry = lb.Retry(3, time.Second, balancer)
	res.CreateFeedEndpoint = retry

	return res
}

// Todo: use connect pool, and reference counting to one connection.
func FeedFactory(makeEndpoint func(f feed.FeedClient) endpoint.Endpoint, tracer stdopentracing.Tracer, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := NewFeedClient(conn, tracer, logger)
		endpoint := makeEndpoint(service)

		return endpoint, conn, nil
	}
}
