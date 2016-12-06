package profile

import (
	"time"

	"github.com/buptmiao/microservice-app/proto/profile"
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

type ProfileClient struct {
	GetProfileEndpoint endpoint.Endpoint
}

func (p ProfileClient) GetProfile(ctx context.Context, in *profile.GetProfileRequest, opts ...grpc.CallOption) (*profile.GetProfileResponse, error) {
	resp, err := p.GetProfileEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*profile.GetProfileResponse), nil
}

func NewProfileClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) profile.ProfileClient {
	limiter := ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(100, 100))

	var getProfileEndpoint endpoint.Endpoint
	{
		getProfileEndpoint = grpctransport.NewClient(
			conn,
			"profile.Profile",
			"GetProfile",
			util.DummyEncode,
			util.DummyDecode,
			profile.GetProfileResponse{},
			grpctransport.ClientBefore(opentracing.ToGRPCRequest(tracer, logger)),
		).Endpoint()
		getProfileEndpoint = opentracing.TraceClient(tracer, "GetProfile")(getProfileEndpoint)
		getProfileEndpoint = limiter(getProfileEndpoint)
		getProfileEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetProfile",
			Timeout: 5 * time.Second,
		}))(getProfileEndpoint)
	}

	return ProfileClient{
		GetProfileEndpoint: getProfileEndpoint,
	}
}
