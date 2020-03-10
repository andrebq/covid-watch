package webapp

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/showTags", showTagsPage).Methods("GET")
	r.HandleFunc("/", landingPage).Methods("GET")
	return r
}
