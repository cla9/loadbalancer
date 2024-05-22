package server

import (
	"github.com/gorilla/mux"
	"lb/internal/rest/resource"
	"net/http"
	"strconv"
)

func NewHttpServer(port int, routes []resource.RouteConfig) *http.Server {
	r := mux.NewRouter()

	for _, config := range routes {
		r.HandleFunc(config.Path, config.Callback).Methods(config.Method)
	}

	return &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: r,
	}
}
