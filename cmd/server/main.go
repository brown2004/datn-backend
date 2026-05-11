package main

import (
	"context"
	"log"
	"net/http"
	"time"

	deliveryhttp "datn-backend/internal/delivery/http"

	"datn-backend/internal/config"
	"datn-backend/internal/database"
	postgresrepo "datn-backend/internal/repo/postgres"
	"datn-backend/internal/usecase"
)

func main() {
	log.Println("backend starting...")

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := postgresrepo.NewUserRepository(db)
	authOTPRepo := postgresrepo.NewAuthOTPRepository(db)
	authUseCase := usecase.NewAuthUseCase(userRepo, authOTPRepo, usecase.AuthOptions{
		ExposeDevOTP: cfg.AppEnv != "production",
	})
	authHandler := deliveryhttp.NewAuthHandler(authUseCase)
	router := deliveryhttp.NewRouter(authHandler)

	addr := ":" + cfg.AppPort
	log.Printf("health endpoint available at http://localhost%s/health", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
