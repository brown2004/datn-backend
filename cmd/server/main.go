package main

import (
	"log"
	"net/http"

	deliveryhttp "datn-backend/internal/delivery/http"

	"datn-backend/internal/config"
)

func main() {
	log.Println("backend starting...")

	cfg := config.Load()
	router := deliveryhttp.NewRouter()

	addr := ":" + cfg.AppPort
	log.Printf("health endpoint available at http://localhost%s/health", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
