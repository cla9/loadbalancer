package resource

type BackendRequest struct {
	ClusterName string `json:"cluster_name" validate:"required"`
	Address     string `json:"ip" validate:"required"`
	Port        uint32 `json:"port" validate:"required"`
}

type ClusterRequest struct {
	Cluster  Cluster  `json:"cluster" validate:"required"`
	Listener Listener `json:"listener" validate:"required"`
}

type ClusterModificationRequest struct {
	Cluster Cluster `json:"cluster" validate:"required"`
}

type Cluster struct {
	Name                  string      `json:"name" validate:"required"`
	ConnectTimeout        uint32      `json:"connect_timeout"`
	HealthCheck           HealthCheck `json:"health_check" validate:"required"`
	HealthyPanicThreshold float32     `json:"healthy_panic_threshold"`
	MaglevTableSize       uint64      `json:"maglev_table_size"`
}

type HealthCheck struct {
	Path               string `json:"path" validate:"required"`
	Timeout            uint32 `json:"timeout" validate:"required"`
	Interval           uint32 `json:"interval" validate:"required"`
	UnhealthyThreshold uint32 `json:"unhealthy_threshold" validate:"required"`
	HealthyThreshold   uint32 `json:"healthy_threshold" validate:"required"`
}

type Listener struct {
	Name          string `json:"name" validate:"required"`
	Address       string `json:"ip" validate:"required"`
	Port          uint32 `json:"port" validate:"required"`
	AccessLogPath string `json:"access_log_path" validate:"required"`
}

type CommonResponse struct {
	Message string `json:"message"`
}
