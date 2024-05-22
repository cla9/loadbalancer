package server

import (
	"context"
	"fmt"
	cds "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	ads "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	eds "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	lds "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

func RunServer(ctx context.Context, srv3 serverv3.Server, port uint, grpcMaxConcurrentStreams int) *grpc.Server {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(uint32(grpcMaxConcurrentStreams)))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	registerServer(grpcServer, srv3)

	log.Printf("management server listening on %d\n", port)
	if err = grpcServer.Serve(lis); err != nil {
		log.Println(err)
	}

	return grpcServer
}

func registerServer(grpcServer *grpc.Server, server serverv3.Server) {
	// 서비스 등록
	ads.RegisterAggregatedDiscoveryServiceServer(grpcServer, server) // Aggregated Discovery Service
	eds.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	lds.RegisterListenerDiscoveryServiceServer(grpcServer, server) // Listener Discovery Service (LDS)
	cds.RegisterClusterDiscoveryServiceServer(grpcServer, server)  // Cluster Discovery Service (CDS)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}
