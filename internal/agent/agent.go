package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"lb/internal/rest/resource"
	httpserver "lb/internal/rest/server"
	"lb/internal/xds/processor"
	"lb/internal/xds/server"
	"net/http"
	"os"
	"sync"
)

type Config struct {
	// envoy config
	EnvoyConfig string
	// GrpcPort is the port for client xds connections.
	GrpcPort int
	// GrpcMaxConcurrentStreams
	GrpcMaxConcurrentStreams int
	// RestPort is the port for client rest api calls.
	RestPort int
	// xds server id.
	NodeName string
	AwxUrl   string
}

type Agent struct {
	Config Config

	restServer *http.Server
	processor  *processor.Processor

	shutdown     bool
	shutdowns    chan struct{}
	shutdownLock sync.Mutex
	grpcServer   *grpc.Server
	router       *resource.Router
}

func New(config Config) (*Agent, error) {

	a := &Agent{
		Config:    config,
		shutdowns: make(chan struct{}),
	}

	setup := []func() error{
		a.setUpExecuteEnvoy,
		a.setupXdsServer,
		a.setupRestServer,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	go func() {
		err := a.serve()
		if err != nil {
			log.Fatalf("failed to start server: %v", err)
			os.Exit(1)
		}
	}()
	return a, nil
}

func (a *Agent) setupXdsServer() error {
	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	proc := processor.NewProcessor(cache, a.Config.NodeName, log.WithField("context", "processor"))
	a.processor = proc
	return nil
}

func (a *Agent) setupRestServer() error {
	router := resource.NewRouter()
	server := httpserver.NewHttpServer(a.Config.RestPort, router.AppendEndpoints())
	a.restServer = server
	a.router = router
	a.router.InjectProcessor(a.processor)
	return nil
}

func (a *Agent) serve() error {
	go func() {
		// Run the xDS server
		ctx := context.Background()
		srv := serverv3.NewServer(ctx, a.processor.Cache, nil)
		a.grpcServer = server.RunServer(ctx, srv, uint(a.Config.GrpcPort), a.Config.GrpcMaxConcurrentStreams)
	}()

	a.processor.ProcessFile(a.Config.EnvoyConfig)

	go func() {
		log.Printf("RestAPI server listening on :%d\n", a.Config.RestPort)
		err := a.restServer.ListenAndServe()
		if err != nil {
			log.Fatalf("failed to rest serve: %v", err)
			os.Exit(1)
		}
	}()

	return nil
}

func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()
	if a.shutdown {
		return nil
	}
	a.shutdown = true
	close(a.shutdowns)

	shutdown := []func() error{
		func() error {
			a.grpcServer.GracefulStop()
			return nil
		},
		a.restServer.Close,
	}
	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) setUpExecuteEnvoy() error {

	err := a.spawnEnvoy("/api/v2/job_templates/9/callback/")
	if err != nil {
		log.Fatalf("failed to spawn 1st envoy: %v", err)
		os.Exit(1)
	}
	err = a.spawnEnvoy("/api/v2/job_templates/10/callback/")
	if err != nil {
		log.Fatalf("failed to spawn 2nd envoy: %v", err)
		os.Exit(1)
	}

	return nil
}

func (a *Agent) spawnEnvoy(path string) error {
	pbytes, _ := json.Marshal(resource.EnvoyRequest{
		Key: "awxshell",
	})
	buff := bytes.NewBuffer(pbytes)
	req, err := http.NewRequest("POST", a.Config.AwxUrl+path, buff)
	if err != nil {
		log.Fatalf("failed to spawn 1st envoy: %v", err)
		os.Exit(1)
	}

	client := &http.Client{}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to spawn 2nd envoy: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	return err
}
