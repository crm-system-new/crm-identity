package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crm-system-new/crm-identity/internal/infrastructure/handler"
	inframsg "github.com/crm-system-new/crm-identity/internal/infrastructure/messaging"
	"github.com/crm-system-new/crm-identity/internal/infrastructure/postgres"
	"github.com/crm-system-new/crm-identity/internal/service"
	"github.com/crm-system-new/crm-shared/pkg/auth"
	"github.com/crm-system-new/crm-shared/pkg/config"
	sharedotel "github.com/crm-system-new/crm-shared/pkg/otel"
	sharedpg "github.com/crm-system-new/crm-shared/pkg/postgres"
)

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := config.Load("identity")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize OpenTelemetry
	shutdown, err := sharedotel.InitTracer(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Printf("WARN: failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	// Connect to PostgreSQL
	pool, err := sharedpg.NewPool(ctx, sharedpg.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		SSLMode:  cfg.DBSSLMode,
	})
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	// Connect to NATS
	publisher, err := inframsg.NewIdentityPublisher(cfg.NatsURL)
	if err != nil {
		log.Fatalf("connect to nats: %v", err)
	}
	defer publisher.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 15*time.Minute, 7*24*time.Hour)

	// Wire dependencies
	userRepo := postgres.NewUserRepository(pool)
	authService := service.NewAuthService(userRepo, publisher, jwtManager)
	userService := service.NewUserService(userRepo)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)

	router := handler.NewRouter(authHandler, userHandler, jwtManager)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.ServicePort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Identity service listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down identity service...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
