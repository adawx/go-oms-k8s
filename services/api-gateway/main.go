package main

import (
	"log"
	"net/http"

	"go-oms/shared/env"
	"go-oms/shared/httpserver"
	"go-oms/shared/metrics"
)

// serviceName labels every metric this service exports.
const serviceName = "api-gateway"

var (
	httpAddr  = env.GetEnv("HTTP_ADDR", ":8081")
	adminAddr = env.GetEnv("ADMIN_HTTP_ADDR", ":9090")
)

func main() {
	log.Print("STARTING API GATEWAY")

	m := metrics.New(serviceName)

	mux := http.NewServeMux()
	mux.Handle("/", m.Middleware("/", http.HandlerFunc(rootHandler)))

	srv := httpserver.New(httpAddr, adminAddr, mux, m, nil)
	httpserver.ExitOnError(srv.Run())
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from API Gateway"))
}
