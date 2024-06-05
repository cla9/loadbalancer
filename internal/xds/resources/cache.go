package resources

import (
	"lb/apis/v1alpha1"
	"time"
)

type Listener struct {
	Name          string
	Address       string
	Port          uint32
	AccessLogPath string
	FilterChains  []v1alpha1.FilterChain
}

type Cluster struct {
	Name                 string
	ListenerName         string
	Endpoints            []Endpoint
	ConnectTimeout       time.Duration
	HealthCheck          v1alpha1.HealthCheck
	HealthPanicThreshold float32
	MaglevTableSize      uint64
}

type Endpoint struct {
	UpstreamHost string
	UpstreamPort uint32
}
