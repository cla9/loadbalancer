package xdscache

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"lb/apis/v1alpha1"
	resources2 "lb/internal/xds/resources"
	"time"
)

type XDSCache struct {
	Listeners map[string]resources2.Listener
	Clusters  map[string]resources2.Cluster
}

func (xds *XDSCache) ClusterContents() []types.Resource {
	var r []types.Resource

	for _, c := range xds.Clusters {

		r = append(r, resources2.MakeCluster(c.Name, c.ConnectTimeout, c.HealthCheck, c.MaglevTableSize, c.HealthPanicThreshold, c.Endpoints))
	}

	return r
}

func (xds *XDSCache) ListenerContents() []types.Resource {
	var r []types.Resource

	for _, l := range xds.Listeners {
		r = append(r, resources2.MakeHTTPListener(l.Name, l.Address, l.Port, l.FilterChains))
	}

	return r
}

func (xds *XDSCache) EndpointsContents() []types.Resource {
	var r []types.Resource

	for _, c := range xds.Clusters {
		r = append(r, resources2.MakeEndpoint(c.Name, c.Endpoints))
	}

	return r
}

func (xds *XDSCache) AddListener(name string, address string, port uint32, filterChains []v1alpha1.FilterChain) {
	xds.Listeners[name] = resources2.Listener{
		Name:         name,
		Address:      address,
		Port:         port,
		FilterChains: filterChains,
	}
}

func (xds *XDSCache) AddCluster(clusterName string, listenerName string, connectTimeout time.Duration, maglevTableSize uint64, healthCheck v1alpha1.HealthCheck, healthPanicThreshold float32) error {

	xds.Clusters[clusterName] = resources2.Cluster{
		Name:                 clusterName,
		ListenerName:         listenerName,
		ConnectTimeout:       connectTimeout,
		MaglevTableSize:      maglevTableSize,
		HealthCheck:          healthCheck,
		HealthPanicThreshold: healthPanicThreshold,
	}
	return nil
}

func (xds *XDSCache) AddEndpoint(clusterName, upstreamHost string, upstreamPort uint32) {
	cluster := xds.Clusters[clusterName]

	cluster.Endpoints = append(cluster.Endpoints, resources2.Endpoint{
		UpstreamHost: upstreamHost,
		UpstreamPort: upstreamPort,
	})

	xds.Clusters[clusterName] = cluster
}

func (xds *XDSCache) RemoveCluster(clusterName string) {
	delete(xds.Clusters, clusterName)
}

func (xds *XDSCache) RemoveEndpoint(clusterName string, address string, port uint32) {
	cluster := xds.Clusters[clusterName]
	endpoints := cluster.Endpoints
	newEndpoints := make([]resources2.Endpoint, len(endpoints)-1)

	k := 0
	for i := 0; i < len(endpoints); i++ {
		if endpoints[i].UpstreamHost == address && endpoints[i].UpstreamPort == port {
			continue
		}
		newEndpoints[k] = endpoints[i]
		k++
	}

	cluster.Endpoints = newEndpoints

	xds.Clusters[clusterName] = cluster
}

func (xds *XDSCache) RemoveListener(clusterName string) {
	cluster := xds.Clusters[clusterName]
	_, ok := xds.Listeners[cluster.ListenerName]
	if ok {
		delete(xds.Listeners, cluster.ListenerName)
	}
}
