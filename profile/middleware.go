package profile

import (
	"fmt"
	"github.com/buptmiao/microservice-app/proto/profile"
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
		Namespace: "profile",
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

func MakeGetPrifileEndpoint(s profile.ProfileServer, tracer stdopentracing.Tracer, logger log.Logger) endpoint.Endpoint {
	ep := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*profile.GetProfileRequest)
		return s.GetProfile(ctx, req)
	}
	epduration := duration.With("method", "GetProfile")
	eplog := log.NewContext(logger).With("method", "GetProfile")
	ep = opentracing.TraceServer(tracer, "GetProfile")(ep)
	ep = EndpointInstrumentingMiddleware(epduration)(ep)
	ep = EndpointLoggingMiddleware(eplog)(ep)
	return ep
}

// MakeGRPCServer makes a set of endpoints available as a gRPC AddServer.
func MakeGRPCServer(ctx context.Context, s profile.ProfileServer, tracer stdopentracing.Tracer, logger log.Logger) profile.ProfileServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(logger),
	}

	return &grpcServer{
		getprofile: grpctransport.NewServer(
			ctx,
			MakeGetPrifileEndpoint(s, tracer, logger),
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			func(_ context.Context, request interface{}) (interface{}, error) { return request, nil },
			append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "GetProfile", logger)))...,
		),
	}
}

type grpcServer struct {
	getprofile grpctransport.Handler
}

func (s *grpcServer) GetProfile(ctx context.Context, req *profile.GetProfileRequest) (*profile.GetProfileResponse, error) {
	_, rep, err := s.getprofile.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*profile.GetProfileResponse), nil
}
