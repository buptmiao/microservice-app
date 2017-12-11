package profile

import (
	"io"
	"time"

	"github.com/buptmiao/microservice-app/proto/profile"
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

var profileCli profile.ProfileClient
var profileInstancer *etcd.Instancer

func Init(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) {
	profileCli = NewProfileClient(conn, tracer, logger)
}

func InitWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) {
	profileCli = NewProfileClientWithSD(sdClient, tracer, logger)
	profileInstancer, _ = etcd.NewInstancer(sdClient, "prifileSD", logger)
}

func GetClient() profile.ProfileClient {
	if profileCli == nil {
		panic("profile client is not be initialized!")
	}
	return profileCli
}

type ProfileClient struct {
	GetProfileEndpoint endpoint.Endpoint
}

func (p *ProfileClient) GetProfile(ctx context.Context, in *profile.GetProfileRequest, opts ...grpc.CallOption) (*profile.GetProfileResponse, error) {
	resp, err := p.GetProfileEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*profile.GetProfileResponse), nil
}

func NewProfileClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) profile.ProfileClient {
	limiter := ratelimit.NewDelayingLimiter(rate.NewLimiter(rate.Every(time.Second), 1000))

	var getProfileEndpoint endpoint.Endpoint
	{
		getProfileEndpoint = grpctransport.NewClient(
			conn,
			"profile.Profile",
			"GetProfile",
			util.DummyEncode,
			util.DummyDecode,
			profile.GetProfileResponse{},
			grpctransport.ClientBefore(opentracing.ContextToGRPC(tracer, logger)),
		).Endpoint()
		getProfileEndpoint = opentracing.TraceClient(tracer, "GetProfile")(getProfileEndpoint)
		getProfileEndpoint = limiter(getProfileEndpoint)
		getProfileEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetProfile",
			Timeout: 5 * time.Second,
		}))(getProfileEndpoint)
	}

	return &ProfileClient{
		GetProfileEndpoint: getProfileEndpoint,
	}
}

func MakeGetProfileEndpoint(f profile.ProfileClient) endpoint.Endpoint {
	return f.(*ProfileClient).GetProfileEndpoint
}

func NewProfileClientWithSD(sdClient etcd.Client, tracer stdopentracing.Tracer, logger log.Logger) profile.ProfileClient {
	res := &ProfileClient{}

	factory := ProfileFactory(MakeGetProfileEndpoint, tracer, logger)
	endpointer := sd.NewEndpointer(profileInstancer, factory, logger)
	balancer := lb.NewRoundRobin(endpointer)
	retry := lb.Retry(3, time.Second, balancer)
	res.GetProfileEndpoint = retry

	return res
}

// Todo: use connect pool, and reference counting to one connection.
func ProfileFactory(makeEndpoint func(f profile.ProfileClient) endpoint.Endpoint, tracer stdopentracing.Tracer, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := NewProfileClient(conn, tracer, logger)
		endpoint := makeEndpoint(service)

		return endpoint, conn, nil
	}
}
