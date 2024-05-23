package resource

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"lb/apis/v1alpha1"
	"lb/internal/xds/processor"
	"net/http"
	"strconv"
	"time"
)

type RouteConfig struct {
	Path     string
	Callback func(writer http.ResponseWriter, request *http.Request)
	Method   string
}

type Router struct {
	processor *processor.Processor
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) AppendEndpoints() []RouteConfig {
	return []RouteConfig{
		{
			Path:     "/cluster",
			Callback: r.addCluster,
			Method:   "POST",
		},
		{
			Path:     "/cluster",
			Callback: r.modifyCluster,
			Method:   "PUT",
		},
		{
			Path:     "/cluster",
			Callback: r.removeCluster,
			Method:   "DELETE",
		},
		{
			Path:     "/backend",
			Callback: r.addBackend,
			Method:   "POST",
		},
		{
			Path:     "/backend",
			Callback: r.removeBackend,
			Method:   "DELETE",
		},
	}
}

func (r *Router) InjectProcessor(processor *processor.Processor) {
	r.processor = processor
}

func (r *Router) addCluster(writer http.ResponseWriter, request *http.Request) {
	var req ClusterRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	err = r.validate(writer, err, req)
	if err != nil {
		log.Info("Failed to validate request structures")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	listener := req.Listener
	cluster := req.Cluster

	exists := r.processor.ExistsListener(listener.Name)
	if !exists {
		r.processor.AppendListener(cluster.Name, listener.Name, listener.Address, listener.Port)
	}

	exists = r.processor.ExistsClusterName(cluster.Name)
	if !exists {
		clusterHealthCheck := cluster.HealthCheck

		if cluster.ConnectTimeout == 0 {
			cluster.ConnectTimeout = 5
		}

		err := r.processor.AppendCluster(cluster.Name,
			listener.Name,
			time.Duration(cluster.ConnectTimeout)*time.Second,
			cluster.MaglevTableSize,
			cluster.HealthyPanicThreshold,
			v1alpha1.HealthCheck{
				Timeout:            time.Duration(clusterHealthCheck.Timeout) * time.Second,
				Interval:           time.Duration(clusterHealthCheck.Interval) * time.Second,
				UnhealthyThreshold: clusterHealthCheck.UnhealthyThreshold,
				HealthyThreshold:   clusterHealthCheck.HealthyThreshold,
				HttpHealthCheck: v1alpha1.HttpHealthCheck{
					Path: clusterHealthCheck.Path,
				},
			})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	}

	r.processor.SyncXds()
	log.Info("synchronize successfully")

	res := CommonResponse{
		Message: "cluster : " + req.Cluster.Name + " is modified.",
	}
	err = json.NewEncoder(writer).Encode(res)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) modifyCluster(writer http.ResponseWriter, request *http.Request) {
	var req ClusterModificationRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	err = r.validate(writer, err, req)
	if err != nil {
		log.Info("Failed to validate request structures")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	cluster := req.Cluster

	exists := r.processor.ExistsClusterName(cluster.Name)
	if !exists {
		http.Error(writer, "cluster name doesn't exists", http.StatusBadRequest)
		return
	}

	clusterHealthCheck := cluster.HealthCheck

	if cluster.ConnectTimeout == 0 {
		cluster.ConnectTimeout = 5
	}

	listenerName := r.processor.FindListenerNameByCluster(cluster.Name)

	err = r.processor.ModifyCluster(cluster.Name,
		listenerName,
		time.Duration(cluster.ConnectTimeout)*time.Second,
		cluster.MaglevTableSize,
		cluster.HealthyPanicThreshold,
		v1alpha1.HealthCheck{
			Timeout:            time.Duration(clusterHealthCheck.Timeout) * time.Second,
			Interval:           time.Duration(clusterHealthCheck.Interval) * time.Second,
			UnhealthyThreshold: clusterHealthCheck.UnhealthyThreshold,
			HealthyThreshold:   clusterHealthCheck.HealthyThreshold,
			HttpHealthCheck: v1alpha1.HttpHealthCheck{
				Path: clusterHealthCheck.Path,
			},
		})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	r.processor.SyncXds()
	log.Info("synchronize successfully")

	res := CommonResponse{
		Message: "cluster : " + req.Cluster.Name + " is created.",
	}
	err = json.NewEncoder(writer).Encode(res)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) removeCluster(writer http.ResponseWriter, request *http.Request) {
	clusterName := request.URL.Query().Get("name")
	if clusterName == "" {
		http.Error(writer, "cluster name is required", http.StatusBadRequest)
		return
	}
	exists := r.processor.ExistsClusterName(clusterName)
	if !exists {
		http.Error(writer, "cluster name doesn't exists", http.StatusBadRequest)
		return
	}
	r.processor.RemoveCluster(clusterName)
	r.processor.RemoveListener(clusterName)
	r.processor.SyncXds()
	log.Info("remove cluster successfully")

	res := CommonResponse{
		Message: "cluster : " + clusterName + " is deleted.",
	}
	err := json.NewEncoder(writer).Encode(res)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) addBackend(writer http.ResponseWriter, request *http.Request) {
	var req BackendRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	err = r.validate(writer, err, req)
	if err != nil {
		log.Info("Failed to validate request structures")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	exists := r.processor.ExistsClusterName(req.ClusterName)
	if !exists {
		http.Error(writer, "cluster name doesn't exists", http.StatusBadRequest)
		return
	}

	exists = r.processor.ExistsEndpoint(req.ClusterName, req.Address, req.Port)
	if exists {
		http.Error(writer, "Endpoint already exists", http.StatusBadRequest)
		return
	}

	r.processor.AddEndpoint(req.ClusterName, req.Address, req.Port)
	r.processor.SyncXds()
	res := CommonResponse{
		Message: "Backend : " + req.Address + ":" + strconv.Itoa(int(req.Port)) + " is added.",
	}

	err = json.NewEncoder(writer).Encode(res)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) validate(writer http.ResponseWriter, err error, req any) error {
	validate := validator.New()
	err = validate.Struct(req)

	if err != nil {
		return err
	}
	return nil
}

func (r *Router) removeBackend(writer http.ResponseWriter, request *http.Request) {
	var req BackendRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	err = r.validate(writer, err, req)
	if err != nil {
		log.Info("Failed to validate request structures")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	exists := r.processor.ExistsClusterName(req.ClusterName)
	if !exists {
		http.Error(writer, "cluster name doesn't exists", http.StatusBadRequest)
		return
	}

	exists = r.processor.ExistsEndpoint(req.ClusterName, req.Address, req.Port)
	if !exists {
		http.Error(writer, "Endpoint doesn't exists", http.StatusBadRequest)
		return
	}

	r.processor.RemoveEndpoint(req.ClusterName, req.Address, req.Port)
	r.processor.SyncXds()

	res := CommonResponse{
		Message: "Backend : " + req.Address + ":" + strconv.Itoa(int(req.Port)) + " is removed.",
	}

	err = json.NewEncoder(writer).Encode(res)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
