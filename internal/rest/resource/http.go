package resource

type BackendRequest struct {
	ClusterName string `json:"cluster_name"`
	Address     string `json:"ip"`
	Port        uint32 `json:"port"`
}

type ClusterRequest struct {
	Cluster  Cluster  `json:"cluster"`
	Listener Listener `json:"listener"`
}

type Cluster struct {
	Name                  string      `json:"name"`
	ConnectTimeout        uint32      `json:"connect_timeout"`
	HealthCheck           HealthCheck `json:"health_check"`
	HealthyPanicThreshold float32     `json:"healthy_panic_threshold"`
	MaglevTableSize       uint64      `json:"maglev_table_size"`
}

type HealthCheck struct {
	Path               string `json:"path"`
	Timeout            uint32 `json:"timeout"`
	Interval           uint32 `json:"interval"`
	UnhealthyThreshold uint32 `json:"unhealthy_threshold"`
	HealthyThreshold   uint32 `json:"healthy_threshold"`
}

type Listener struct {
	Name    string `json:"name"`
	Address string `json:"ip"`
	Port    uint32 `json:"port"`
}

type CommonResponse struct {
	Message string `json:"message"`
}
