package processor

import (
	"context"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"lb/apis/v1alpha1"
	"lb/internal/xds/resources"
	"lb/internal/xds/xdscache"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type Processor struct {
	Cache  cache.SnapshotCache
	nodeID string

	snapshotVersion int64
	logrus.FieldLogger
	xdsCache xdscache.XDSCache
}

func NewProcessor(cache cache.SnapshotCache, nodeID string, log logrus.FieldLogger) *Processor {
	return &Processor{
		Cache:           cache,
		nodeID:          nodeID,
		snapshotVersion: rand.Int63n(1000),
		FieldLogger:     log,
		xdsCache: xdscache.XDSCache{
			Listeners: make(map[string]resources.Listener),
			Clusters:  make(map[string]resources.Cluster),
		},
	}
}

func (p *Processor) newSnapshotVersion() string {

	if p.snapshotVersion == math.MaxInt64 {
		p.snapshotVersion = 0
	}

	p.snapshotVersion++
	return strconv.FormatInt(p.snapshotVersion, 10)
}

func (p *Processor) ProcessFile(path string) {

	if path == "" {
		p.Info("envoy config file doesn't exist. skip the file sync process")
		return
	}

	envoyConfig, err := parseYaml(path)
	if err != nil {
		p.Errorf("error parsing yaml file: %+v", err)
		os.Exit(1)
		return
	}

	listenerMap := make(map[string]string)

	for _, l := range envoyConfig.Listeners {
		socketAddress := l.Address.SocketAddress
		p.xdsCache.AddListener(l.Name, socketAddress.Address, uint32(socketAddress.Port), l.FilterChains)
		listenerMap[l.FilterChains[0].Filters[0].TypeConfig.Cluster] = l.Name
	}

	for _, c := range envoyConfig.Clusters {
		err := p.xdsCache.AddCluster(c.Name, listenerMap[c.Name], c.ConnectTimeout, c.HealthChecks[0], c.CommonLbConfig.HealthPanicThreshold)
		if err != nil {
			p.Errorf("error parsing cluster configuration: %+v", err)
			os.Exit(1)
			return
		}
	}

	p.SyncXds()
}

func (p *Processor) ExistsListener(listenerName string) bool {
	_, ok := p.xdsCache.Listeners[listenerName]
	return ok
}

func (p *Processor) AppendListener(clusterName string, listenerName string, address string, port uint32) {
	p.xdsCache.AddListener(listenerName, address, port, []v1alpha1.FilterChain{
		{
			Filters: []v1alpha1.Filter{
				{
					Name: "envoy.filters.network.tcp_proxy",
					TypeConfig: v1alpha1.TypeConfig{
						Type:       "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy",
						StatPrefix: "tcp_proxy",
						Cluster:    clusterName,
					},
				},
			},
		},
	})
}

func (p *Processor) ExistsClusterName(clusterName string) bool {
	_, ok := p.xdsCache.Clusters[clusterName]
	return ok
}

func (p *Processor) SyncXds() {
	resources := map[resource.Type][]types.Resource{
		resource.EndpointType: p.xdsCache.EndpointsContents(),
		resource.ClusterType:  p.xdsCache.ClusterContents(),
		resource.ListenerType: p.xdsCache.ListenerContents(),
	}

	snapshot, err := cache.NewSnapshot(
		p.newSnapshotVersion(),
		resources,
	)
	if err != nil {
		p.Errorf("error generating new snapshot: %v", err)
		return
	}

	if err := snapshot.Consistent(); err != nil {
		p.Errorf("snapshot inconsistency: %+v\n\n\n%+v", snapshot, err)
		return
	}
	p.Debugf("will serve snapshot %+v", snapshot)

	if err := p.Cache.SetSnapshot(context.Background(), p.nodeID, snapshot); err != nil {
		p.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}
}

func (p *Processor) AppendCluster(clusterName string, listenerName string, connectionTimeout time.Duration, healthPanicThreshold float32, healthCheck v1alpha1.HealthCheck) error {
	err := p.xdsCache.AddCluster(clusterName, listenerName, connectionTimeout, healthCheck, healthPanicThreshold)
	return err
}

func (p *Processor) RemoveCluster(clusterName string) {
	p.xdsCache.RemoveCluster(clusterName)
}

func (p *Processor) AddEndpoint(clusterName string, address string, port uint32) {
	p.xdsCache.AddEndpoint(clusterName, address, port)

}

func (p *Processor) RemoveEndpoint(clusterName string, address string, port uint32) {
	p.xdsCache.RemoveEndpoint(clusterName, address, port)
}

func (p *Processor) RemoveListener(clusterName string) {
	p.xdsCache.RemoveListener(clusterName)
}

func (p *Processor) ExistsEndpoint(clusterName string, address string, port interface{}) bool {
	endpoints := p.xdsCache.Clusters[clusterName].Endpoints
	for _, e := range endpoints {
		if e.UpstreamHost == address && e.UpstreamPort == port {
			return true
		}
	}
	return false
}
