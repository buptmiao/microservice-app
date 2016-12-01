package main

import (
	"flag"
	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"os"
	"os/signal"
	"strings"
	"golang.org/x/net/context"
	"github.com/buptmiao/microservice-demo-dev/feed"
	p_feed "github.com/buptmiao/microservice-demo-dev/proto/feed"
	"syscall"
	"fmt"
	"net"
	"google.golang.org/grpc"
)

func main() {
	var (
		addr = flag.String("addr", ":8082", "the microservices grpc address")
		zipkinAddr = flag.String("zipkin", "", "the zipkin address")
	)
	flag.Parse()

	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stdout)
	logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
	logger = log.NewContext(logger).With("caller", log.DefaultCaller)


	var tracer stdopentracing.Tracer
	if *zipkinAddr != "" {
		logger := log.NewContext(logger).With("tracer", "Zipkin")
		logger.Log("addr", *zipkinAddr)
		collector, err := zipkin.NewKafkaCollector(
			strings.Split(*zipkinAddr, ","),
			zipkin.KafkaLogger(logger),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		tracer, err = zipkin.NewTracer(
			zipkin.NewRecorder(collector, false, "localhost:80", "addsvc"),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
	}

	service := feed.NewFeedService()

	errchan := make(chan error)
	ctx := context.Background()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errchan <- fmt.Errorf("%s", <-c)
	}()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Log("err", err)
		return
	}

	srv := feed.MakeGRPCServer(ctx, service, tracer, logger)
	s := grpc.NewServer()
	p_feed.RegisterFeedServer(s, srv)

	go func() {
		//logger := log.NewContext(logger).With("transport", "gRPC")
		logger.Log("addr", *addr)
		errchan <- s.Serve(ln)
	}()
	logger.Log("graceful shutdown...", <-errchan)
	s.GracefulStop()
}