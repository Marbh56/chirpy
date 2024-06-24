package main

import (
	"fmt"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

var count int = 0

func (cfg *apiConfig) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	}
}
	
	
func getFilerserverCount(w http.RepsoneWritter, _ *http.Request){
	
}

func getHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	fmt.Print("starting server...")

	apiCfg := &apiConfig{}

	fs := http.FileServer(http.Dir("."))
	mux.Handle("/app/*", api.Cfg.middlewareMetrics(http.StripPrefix("/app/", fs)))

	mux.HandleFunc("/healthz", getHealth)

	http.ListenAndServe(":8080", mux)
}
