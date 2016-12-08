package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/buptmiao/microservice-app/apigateway"
	"github.com/buptmiao/microservice-app/client/feed"
	"github.com/buptmiao/microservice-app/client/profile"
	"github.com/buptmiao/microservice-app/client/topic"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/etcd"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
)

func main() {
	var (
		httpAddr   = flag.String("http.addr", ":8080", "HTTP server address")
		etcdAddr   = flag.String("etcd.addr", "", "etcd registry address")
		zipkinAddr = flag.String("zipkin.addr", "", "tracer server address")
	)
	flag.Parse()
	ctx := context.Background()
	// Logging domain.
	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
	logger = log.NewContext(logger).With("caller", log.DefaultCaller)

	// Service discovery domain. In this example we use etcd.
	var sdClient etcd.Client
	var peers []string
	if len(*etcdAddr) > 0 {
		peers = strings.Split(*etcdAddr, ",")
	}
	sdClient, err := etcd.NewClient(ctx, peers, etcd.ClientOptions{})
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	// Transport domain.
	tracer := stdopentracing.GlobalTracer() // nop by default
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
			zipkin.NewRecorder(collector, false, "localhost:80", "http"),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
	}

	feed.InitWithSD(sdClient, tracer, logger)
	profile.InitWithSD(sdClient, tracer, logger)
	topic.InitWithSD(sdClient, tracer, logger)

	router := gin.New()
	apigateway.Register(router)

	server := &http.Server{Addr: *httpAddr, Handler: router}
	if err = gracehttp.Serve(server); err != nil {
		panic(err)
	}
}
