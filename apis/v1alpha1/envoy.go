package v1alpha1

import "time"

type EnvoyConfig struct {
	Name string `yaml:"name"`
	Spec `yaml:"spec"`
}

type Spec struct {
	Listeners []Listener `yaml:"listeners"`
	Clusters  []Cluster  `yaml:"clusters"`
}

type Listener struct {
	Name         string        `yaml:"name"`
	Address      Address       `yaml:"address"`
	FilterChains []FilterChain `yaml:"filter_chains"`
}

type Address struct {
	SocketAddress SocketAddress `yaml:"socket_address"`
}

type SocketAddress struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port_value"`
}

type Cluster struct {
	Name           string         `yaml:"name"`
	ConnectTimeout time.Duration  `yaml:"connect_timeout"`
	MaglevLbPolicy MaglevLbPolicy `yaml:"maglev_lb_config"`
	HealthChecks   []HealthCheck  `yaml:"health_checks"`
	CommonLbConfig CommonLbConfig `yaml:"common_lb_config"`
}

type CommonLbConfig struct {
	HealthPanicThreshold float32 `yaml:"health_panic_threshold"`
}

type HealthCheck struct {
	Timeout            time.Duration   `yaml:"timeout"`
	Interval           time.Duration   `yaml:"interval"`
	UnhealthyThreshold uint32          `yaml:"unhealthy_threshold"`
	HealthyThreshold   uint32          `yaml:"healthy_threshold"`
	HttpHealthCheck    HttpHealthCheck `yaml:"http_health_check"`
}

type HttpHealthCheck struct {
	Path string `yaml:"path"`
}

type MaglevLbPolicy struct {
	TableSize int `yaml:"table_size"`
}

type FilterChain struct {
	Filters []Filter `yaml:"filters"`
}

type Filter struct {
	Name       string     `yaml:"name"`
	TypeConfig TypeConfig `yaml:"typed_config"`
}

type TypeConfig struct {
	Type       string `yaml:"@type"`
	StatPrefix string `yaml:"stat_prefix"`
	Cluster    string `yaml:"cluster"`
}
