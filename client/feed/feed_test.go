package feed_test

import (
	client "github.com/buptmiao/microservice-app/client/feed"
	"github.com/buptmiao/microservice-app/feed"
	p_feed "github.com/buptmiao/microservice-app/proto/feed"
	"github.com/go-kit/kit/log"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

func runFeedServer(addr string) *grpc.Server {
	service := feed.NewFeedService()
	ctx := context.Background()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	srv := feed.MakeGRPCServer(ctx, service, opentracing.NoopTracer{}, log.NewNopLogger())
	s := grpc.NewServer()
	p_feed.RegisterFeedServer(s, srv)

	go func() {
		s.Serve(ln)
	}()
	time.Sleep(time.Second)
	return s
}

func TestNewFeedClient(t *testing.T) {
	s := runFeedServer(":8001")
	defer s.GracefulStop()
	conn, err := grpc.Dial(":8001", grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	service := client.NewFeedClient(conn, opentracing.NoopTracer{}, log.NewNopLogger())
	req := &p_feed.FeedRecord{
		Id:      1,
		UserId:  123,
		Content: "hello world",
	}
	_, err = service.CreateFeed(context.Background(), req)
	if err != nil {
		panic(err)
	}
	req2 := &p_feed.GetFeedsRequest{
		UserId: 123,
		Size:   5,
	}
	resp, err := service.GetFeeds(context.Background(), req2)
	if err != nil {
		panic(err)
	}
	if len(resp.GetFeeds()) <= 0 {
		panic(resp)
	}
}
