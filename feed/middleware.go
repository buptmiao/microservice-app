package feed

import (
	"context"
	"fmt"
	"github.com/buptmiao/microservice-app/proto/feed"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	oldcontext "golang.org/x/net/context"
	"time"
)

var (
	duration metrics.Histogram = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "feed",
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

func MakeGetFeedsEndpoint(s feed.FeedServer, tracer stdopentracing.Tracer, logger log.Logger) endpoint.Endpoint {
	ep := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*feed.GetFeedsRequest)
		return s.GetFeeds(ctx, req)
	}
	epduration := duration.With("method", "GetFeeds")
	eplog := log.With(logger, "method", "GetFeeds")
	ep = opentracing.TraceServer(tracer, "GetFeeds")(ep)
	ep = EndpointInstrumentingMiddleware(epduration)(ep)
	ep = EndpointLoggingMiddleware(eplog)(ep)
	return ep
}

func MakeCreateFeedEndpoint(s feed.FeedServer, tracer stdopentracing.Tracer, logger log.Logger) endpoint.Endpoint {
	ep := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*feed.FeedRecord)
		return s.CreateFeed(ctx, req)
	}
	epduration := duration.With("method", "CreateFeed")
	eplog := log.With(logger, "method", "CreateFeed")
	ep = opentracing.TraceServer(tracer, "CreateFeed")(ep)
	ep = EndpointInstrumentingMiddleware(epduration)(ep)
	ep = EndpointLoggingMiddleware(eplog)(ep)
	return ep
}

// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(s feed.FeedServer, tracer stdopentracing.Tracer, logger log.Logger) feed.FeedServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}

	return &grpcServer{
		getfeeds: grpctransport.NewServer(
			MakeGetFeedsEndpoint(s, tracer, logger),
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(tracer, "GetFeeds", logger)))...,
		),
		createfeed: grpctransport.NewServer(
			MakeCreateFeedEndpoint(s, tracer, logger),
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			append(options, grpctransport.ServerBefore(opentracing.GRPCToContext(tracer, "CreateFeed", logger)))...,
		),
	}
}

type grpcServer struct {
	getfeeds   grpctransport.Handler
	createfeed grpctransport.Handler
}

func (s *grpcServer) GetFeeds(ctx oldcontext.Context, req *feed.GetFeedsRequest) (*feed.GetFeedsResponse, error) {
	_, rep, err := s.getfeeds.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*feed.GetFeedsResponse), nil
}

func (s *grpcServer) CreateFeed(ctx oldcontext.Context, req *feed.FeedRecord) (*feed.OkResponse, error) {
	_, rep, err := s.createfeed.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*feed.OkResponse), nil
}
