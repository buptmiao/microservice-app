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
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

var topicCli topic.TopicClient
var topicInstancer *etcd.Instancer

func Init(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) {
	topicCli = NewTopicClient(conn, tracer, logger)
}

func InitWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) {
	topicCli = NewTopicClientWithSD(sdClient, tracer, logger)
	topicInstancer, _ = etcd.NewInstancer(sdClient, "topicSD", logger)

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
	limiter := ratelimit.NewDelayingLimiter(rate.NewLimiter(rate.Every(time.Second), 1000))

	var getTopicEndpoint endpoint.Endpoint
	{
		getTopicEndpoint = grpctransport.NewClient(
			conn,
			"topic.Topic",
			"GetTopic",
			util.DummyEncode,
			util.DummyDecode,
			topic.GetTopicResponse{},
			grpctransport.ClientBefore(opentracing.ContextToGRPC(tracer, logger)),
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
	endpointer := sd.NewEndpointer(topicInstancer, factory, logger)
	balancer := lb.NewRoundRobin(endpointer)
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
