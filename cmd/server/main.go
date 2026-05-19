package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"datn-backend/internal/config"
	"datn-backend/internal/database"
	deliveryhttp "datn-backend/internal/delivery/http"
	httprouter "datn-backend/internal/delivery/http/router"
	postgresrepo "datn-backend/internal/repo/postgres"
	"datn-backend/internal/token"
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
	refreshTokenRepo := postgresrepo.NewRefreshTokenRepository(db)
	registrationRepo := postgresrepo.NewRegistrationRepository(db)
	pcAgentRepo := postgresrepo.NewPCAgentRepository(db)
	mobileDeviceRepo := postgresrepo.NewMobileDeviceRepository(db)
	tokenService := token.NewService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	authUseCase := usecase.NewAuthUseCase(userRepo, authOTPRepo, refreshTokenRepo, registrationRepo, usecase.AuthOptions{
		ExposeDevOTP:    cfg.AppEnv != "production",
		TokenService:    tokenService,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
	})
	deviceLinkUseCase := usecase.NewDeviceLinkUseCase(pcAgentRepo, mobileDeviceRepo)
	authHandler := deliveryhttp.NewAuthHandler(authUseCase)
	featureHandler := deliveryhttp.NewFeatureHandler(deviceLinkUseCase)
	router := httprouter.NewRouter(authHandler, featureHandler, tokenService)

	addr := ":" + cfg.AppPort
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("health endpoint available at http://localhost%s/health", addr)
		errCh <- server.ListenAndServe()
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		return
	case <-shutdownCh:
		log.Println("backend shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
	}

	if err := <-errCh; err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
