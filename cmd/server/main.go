package main

// goose:
// goose -dir internal/database/migrations postgres "postgres://postgres:postgres@localhost:5433/datn?sslmode=disable" up
import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"datn-backend/internal/config"
	"datn-backend/internal/database"
	deliveryhttp "datn-backend/internal/delivery/http"
	httprouter "datn-backend/internal/delivery/http/router"
	backendmqtt "datn-backend/internal/mqtt"
	"datn-backend/internal/notification"
	postgresrepo "datn-backend/internal/repo/postgres"
	"datn-backend/internal/token"
	"datn-backend/internal/usecase"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

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
	pairingSessionRepo := postgresrepo.NewPairingSessionRepository(db)
	alertRepo := postgresrepo.NewAlertRepository(db)
	tokenService := token.NewService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	notificationSender := newNotificationSender(cfg)
	authUseCase := usecase.NewAuthUseCase(userRepo, authOTPRepo, refreshTokenRepo, registrationRepo, usecase.AuthOptions{
		TokenService:    tokenService,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
	})
	pairingUseCase := usecase.NewPairingUseCase(pcAgentRepo, mobileDeviceRepo, pairingSessionRepo)
	alertUseCase := usecase.NewAlertUseCase(alertRepo, pcAgentRepo, mobileDeviceRepo, notificationSender)
	authHandler := deliveryhttp.NewAuthHandler(authUseCase)
	protectionCommandPublisher := newProtectionCommandPublisher(cfg)
	pcAgentHandler := deliveryhttp.NewPCAgentHandler(pairingUseCase, protectionCommandPublisher)
	mobileDeviceHandler := deliveryhttp.NewMobileDeviceHandler(pairingUseCase)
	alertHandler := deliveryhttp.NewAlertHandler(alertUseCase)
	router := httprouter.NewRouter(authHandler, pcAgentHandler, mobileDeviceHandler, alertHandler, tokenService)

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()
	startMQTTSubscriber(runtimeCtx, cfg, alertUseCase)

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
		runtimeCancel()
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

func newNotificationSender(cfg config.Config) notification.Sender {
	if strings.TrimSpace(cfg.FCMServiceAccountFile) == "" {
		log.Println("FIREBASE_SERVICE_ACCOUNT_FILE/GOOGLE_APPLICATION_CREDENTIALS is empty; using mock notification sender")
		return notification.NewMockSender()
	}

	return notification.NewFCMSender(cfg.FCMServiceAccountFile, cfg.FCMProjectID)
}

func newProtectionCommandPublisher(cfg config.Config) *backendmqtt.ProtectionCommandPublisher {
	if strings.TrimSpace(cfg.MQTTBroker) == "" {
		log.Println("MQTT_BROKER is empty; mqtt protection command publisher disabled")
		return nil
	}

	return &backendmqtt.ProtectionCommandPublisher{
		Broker:   cfg.MQTTBroker,
		ClientID: cfg.MQTTClientID,
		Username: cfg.MQTTUsername,
		Password: cfg.MQTTPassword,
	}
}

func startMQTTSubscriber(ctx context.Context, cfg config.Config, alertUseCase *usecase.AlertUseCase) {
	if strings.TrimSpace(cfg.MQTTBroker) == "" {
		log.Println("MQTT_BROKER is empty; mqtt alert subscriber disabled")
		return
	}

	subscriber := &backendmqtt.Subscriber{
		Broker:      cfg.MQTTBroker,
		ClientID:    cfg.MQTTClientID,
		Username:    cfg.MQTTUsername,
		Password:    cfg.MQTTPassword,
		TopicFilter: cfg.MQTTAlertTopic,
	}

	go func() {
		err := subscriber.Run(ctx, func(ctx context.Context, message backendmqtt.Message) error {
			return handleMQTTAlert(ctx, alertUseCase, message)
		})
		if err != nil {
			log.Printf("mqtt subscriber stopped: %v", err)
		}
	}()
}

type mqttAlertPayload struct {
	PCAgentID string    `json:"pc_agent_id"`
	DeviceID  string    `json:"device_id"`
	AlertType string    `json:"alert_type"`
	EventType string    `json:"event_type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func handleMQTTAlert(ctx context.Context, alertUseCase *usecase.AlertUseCase, message backendmqtt.Message) error {
	var payload mqttAlertPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return err
	}

	pcAgentID := strings.TrimSpace(payload.PCAgentID)
	if pcAgentID == "" {
		pcAgentID = topicTail(message.Topic)
	}

	alertType := strings.TrimSpace(payload.AlertType)
	if alertType == "" {
		alertType = payload.EventType
	}

	alert, err := alertUseCase.CreateAlertFromAgent(ctx, usecase.CreateAlertFromAgentInput{
		PCAgentID:   pcAgentID,
		AlertType:   alertType,
		Message:     payload.Message,
		TriggeredAt: payload.Timestamp,
	})
	if err != nil {
		return err
	}

	log.Printf("alert accepted: id=%s pc_agent_id=%s type=%s", alert.ID, alert.AgentID, alert.Type)
	return nil
}

func topicTail(topic string) string {
	topic = strings.Trim(strings.TrimSpace(topic), "/")
	if topic == "" {
		return ""
	}

	parts := strings.Split(topic, "/")
	return parts[len(parts)-1]
}
