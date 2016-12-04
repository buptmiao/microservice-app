package topic

import (
	"fmt"
	"github.com/buptmiao/microservice-app/proto/topic"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"time"
)

var (
	duration metrics.Histogram = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "topic",
		Name:      "request_duration_ns",
		Help:      "Request duration in nanoseconds.",
	}, []string{"method", "success"})
)

func EndpointInstrumentingMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}

func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				logger.Log("error", err, "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)
		}
	}
}

func MakeGetTopicEndpoint(s topic.TopicServer, tracer stdopentracing.Tracer, logger log.Logger) endpoint.Endpoint {
	ep := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*topic.GetTopicRequest)
		return s.GetTopic(ctx, req)
	}
	epduration := duration.With("method", "GetTopic")
	eplog := log.NewContext(logger).With("method", "GetTopic")
	ep = opentracing.TraceServer(tracer, "GetTopic")(ep)
	ep = EndpointInstrumentingMiddleware(epduration)(ep)
	ep = EndpointLoggingMiddleware(eplog)(ep)
	return ep
}

// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(ctx context.Context, s topic.TopicServer, tracer stdopentracing.Tracer, logger log.Logger) topic.TopicServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}

	return &grpcServer{
		gettopic: grpctransport.NewServer(
			ctx,
			MakeGetTopicEndpoint(s, tracer, logger),
			nil,
			nil,
			append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "GetTopic", logger)))...,
		),
	}
}

type grpcServer struct {
	gettopic grpctransport.Handler
}

func (s *grpcServer) GetTopic(ctx context.Context, req *topic.GetTopicRequest) (*topic.GetTopicResponse, error) {
	_, rep, err := s.gettopic.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*topic.GetTopicResponse), nil
}
