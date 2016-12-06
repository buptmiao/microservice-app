package topic

import (
	"time"

	"github.com/buptmiao/microservice-app/proto/topic"
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

type TopicClient struct {
	GetTopicEndpoint endpoint.Endpoint
}

func (p TopicClient) GetTopic(ctx context.Context, in *topic.GetTopicRequest, opts ...grpc.CallOption) (*topic.GetTopicResponse, error) {
	resp, err := p.GetTopicEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*topic.GetTopicResponse), nil
}

func NewTopicClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) topic.TopicClient {
	limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))

	var getTopicEndpoint endpoint.Endpoint
	{
		getTopicEndpoint = grpctransport.NewClient(
			conn,
			"topic.Topic",
			"GetTopic",
			util.DummyEncode,
			util.DummyDecode,
			topic.GetTopicResponse{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		getTopicEndpoint = opentracing.TraceClient(tracer, "GetTopic")(getTopicEndpoint)
		getTopicEndpoint = limiter(getTopicEndpoint)
		getTopicEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetTopic",
			Timeout: 5 * time.Second,
		}))(getTopicEndpoint)
	}

	return TopicClient{
		GetTopicEndpoint: getTopicEndpoint,
	}
}
