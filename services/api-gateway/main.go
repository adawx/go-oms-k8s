package main

import (
	"log"
	"net/http"

	"go-oms/shared/env"
)

var (
	httpAddr = env.GetEnv("HTTP_ADDR", ":8081")
)

func main() {
	log.Print("STARTING API GATEWAY")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from API Gateway"))
	})

	http.ListenAndServe(httpAddr, nil)
}
