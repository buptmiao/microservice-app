package main

import (
	"flag"
	"fmt"
	p_topic "github.com/buptmiao/microservice-app/proto/topic"
	"github.com/buptmiao/microservice-app/topic"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd/etcd"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		addr       = flag.String("addr", ":8084", "the microservices grpc address")
		debugAddr  = flag.String("debug.addr", ":6064", "the debug and metrics address")
		etcdAddr   = flag.String("etcd.addr", "", "etcd registry address")
		zipkinAddr = flag.String("zipkin", "", "the zipkin address")
	)
	flag.Parse()
	key := "/services/topic/" + *addr
	value := *addr
	ctx := context.Background()

	//logger
	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stdout)
	logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
	logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	logger = log.NewContext(logger).With("service", "topic")

	// Service registrar domain. In this example we use etcd.
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

	// Build the registrar.
	registrar := etcd.NewRegistrar(sdClient, etcd.Service{
		Key:   key,
		Value: value,
	}, log.NewNopLogger())

	// Register our instance.
	registrar.Register()

	defer registrar.Deregister()

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
			zipkin.NewRecorder(collector, false, "localhost:80", "topic"),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
	}

	service := topic.NewTopicService()

	errchan := make(chan error)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		errchan <- fmt.Errorf("%s", <-c)
	}()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Log("err", err)
		return
	}

	srv := topic.MakeGRPCServer(ctx, service, tracer, logger)
	s := grpc.NewServer()
	p_topic.RegisterTopicServer(s, srv)

	go func() {
		//logger := log.NewContext(logger).With("transport", "gRPC")
		logger.Log("addr", *addr)
		errchan <- s.Serve(ln)
	}()

	// Debug listener.
	go func() {
		logger := log.NewContext(logger).With("transport", "debug")

		m := http.NewServeMux()
		m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		m.Handle("/metrics", stdprometheus.Handler())

		logger.Log("addr", *debugAddr)
		errchan <- http.ListenAndServe(*debugAddr, m)
	}()

	logger.Log("graceful shutdown...", <-errchan)
	s.GracefulStop()
}
