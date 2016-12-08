package topic

import (
	"io"
	"time"

	"github.com/buptmiao/microservice-app/proto/topic"
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

var topicCli topic.TopicClient

func Init(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) {
	topicCli = NewTopicClient(conn, tracer, logger)
}

func InitWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) {
	topicCli = NewTopicClientWithSD(sdClient, tracer, logger)
}

func GetClient() topic.TopicClient {
	if topicCli == nil {
		panic("topic client is not be initialized!")
	}
	return topicCli
}

type TopicClient struct {
	GetTopicEndpoint endpoint.Endpoint
}

func (p *TopicClient) GetTopic(ctx context.Context, in *topic.GetTopicRequest, opts ...grpc.CallOption) (*topic.GetTopicResponse, error) {
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

	return &TopicClient{
		GetTopicEndpoint: getTopicEndpoint,
	}
}

func MakeGetTopicEndpoint(f topic.TopicClient) endpoint.Endpoint {
	return f.(*TopicClient).GetTopicEndpoint
}

func NewTopicClientWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) topic.TopicClient {
	res := &TopicClient{}

	factory := TopicFactory(MakeGetTopicEndpoint, tracer, logger)
	subscriber, _ := etcd.NewSubscriber(sdClient, "/services/topic", factory, logger)
	balancer := lb.NewRoundRobin(subscriber)
	retry := lb.Retry(3, time.Second, balancer)
	res.GetTopicEndpoint = retry

	return res
}

// Todo: use connect pool, and reference counting to one connection.
func TopicFactory(makeEndpoint func(f topic.TopicClient) endpoint.Endpoint, tracer stdopentracing.Tracer, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := NewTopicClient(conn, tracer, logger)
		endpoint := makeEndpoint(service)

		return endpoint, conn, nil
	}
}
