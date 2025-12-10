package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"crypto/tls"
	"strings"
	"context"
	"offer-eligibility-api/internal/config"
	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/features"
	"offer-eligibility-api/internal/handler"
	"offer-eligibility-api/internal/middleware"
	"offer-eligibility-api/internal/service"
	tlsconfig "offer-eligibility-api/internal/tls"
	tracing "offer-eligibility-api/internal/tracing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const (
	defaultPort       = "8080"
	defaultDBPath     = "./offer_eligibility.db"
	defaultRateLimit  = 100 // requests per window
	defaultRateWindow = 60  // seconds
)

func main() {
	configFile := flag.String("config", "", "Path to configuration file (JSON)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize event manager (if enabled)
	var eventManager *events.Manager
	if cfg.Features.EventHooksEnabled {
		eventManager = events.NewManager(true)
		defer eventManager.Shutdown()
		log.Println("Event-driven hooks: enabled")
	}

	// Initialize event manager (if enabled)
	var eventManager *events.Manager
	if cfg.Features.EventHooksEnabled {
		eventManager = events.NewManager(true)
		defer eventManager.Shutdown()
		log.Println("Event-driven hooks: enabled")
	}

	// Initialize service
	svc := service.NewService(db)
	if eventManager != nil {
		svc.SetEventManager(eventManager)
	}
	if eventManager != nil {
		svc.SetEventManager(eventManager)
	}

	// Initialize feature flags
	featureManager := features.NewManager()
	featureManager.Register(features.FeatureCacheEnabled, cfg.Features.CacheEnabled, "Enable caching layer")
	featureManager.Register(features.FeatureEventHooksEnabled, cfg.Features.EventHooksEnabled, "Enable event-driven hooks")
	featureManager.Register(features.FeatureAdvancedEligibility, cfg.Features.AdvancedEligibility, "Enable advanced eligibility calculations")
	featureManager.Register(features.FeatureBatchProcessing, cfg.Features.BatchProcessing, "Enable batch processing optimizations")
	defer featureManager.Shutdown()

	// Initialize tracing (if enabled)
	if cfg.Tracing.Enabled {
		_, err := tracing.InitTracing(tracing.Config{
			Enabled:     cfg.Tracing.Enabled,
			Endpoint:    cfg.Tracing.Endpoint,
			ServiceName: cfg.Tracing.ServiceName,
			Environment: cfg.Tracing.Environment,
		})
		if err != nil {
			log.Printf("WARNING: Failed to initialize tracing: %v", err)
		} else {
			log.Printf("Tracing enabled: %s -> %s", cfg.Tracing.ServiceName, cfg.Tracing.Endpoint)
			defer func() {
				if err := tracing.Shutdown(context.Background()); err != nil {
					log.Printf("Error shutting down tracing: %v", err)
				}
			}()
		}
	}

	// Initialize handlers with configuration
	h := handler.NewHandlerWithOptions(svc, handler.NewHandlerOptions{
		MaxBodySize: cfg.Security.MaxRequestBodySize,
	})

	// Initialize rate limiter (if enabled)
	var rateLimiter *middleware.RateLimiter
	if cfg.RateLimit.Enabled {
		rateLimiter = middleware.NewRateLimiter(cfg.RateLimit.Rate, time.Duration(cfg.RateLimit.Window)*time.Second)
		defer rateLimiter.Stop()
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware (order matters)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	
	// Tracing middleware (if enabled)
	if cfg.Tracing.Enabled {
		r.Use(middleware.TracingMiddleware())
	}
	
	// Rate limiting middleware (if enabled)
	if cfg.RateLimit.Enabled && rateLimiter != nil {
		r.Use(middleware.RateLimitMiddleware(rateLimiter))
	}
	
	// CORS configuration
	allowedOrigins := strings.Split(cfg.Security.AllowedOrigins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/offers", func(r chi.Router) {
		r.Post("/", h.CreateOffer)
	})

	r.Route("/transactions", func(r chi.Router) {
		r.Post("/", h.CreateTransactions)
	})

	r.Route("/users", func(r chi.Router) {
		r.Get("/{user_id}/eligible-offers", h.GetEligibleOffers)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Configure TLS if enabled
	var tlsConfig *tls.Config
	if cfg.Server.EnableTLS {
		tlsCfg := tlsconfig.Config{
			CertFile: cfg.Server.CertFile,
			KeyFile:  cfg.Server.KeyFile,
		}

		var err error
		tlsConfig, err = tlsconfig.LoadTLSConfig(tlsCfg)
		if err != nil {
			log.Fatalf("Failed to load TLS configuration: %v", err)
		}

		if cfg.Server.CertFile == "" || cfg.Server.KeyFile == "" {
			log.Println("WARNING: No certificate files provided, using self-signed certificate for development")
		}
	}

	// Build server address
	addr := cfg.Server.Port
	if cfg.Server.Host != "" {
		addr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	} else {
		addr = fmt.Sprintf(":%s", cfg.Server.Port)
	}

	protocol := "HTTP"
	if cfg.Server.EnableTLS {
		protocol = "HTTPS"
	}
	log.Printf("Starting %s server on %s", protocol, addr)
	log.Printf("Database: %s", cfg.Database.Path)
	if cfg.RateLimit.Enabled {
		log.Printf("Rate limit: %d requests per %d seconds", cfg.RateLimit.Rate, cfg.RateLimit.Window)
	} else {
		log.Println("Rate limiting: disabled")
	}

	server := &http.Server{
		Addr:      addr,
		Handler:   r,
		TLSConfig: tlsConfig,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	if cfg.Server.EnableTLS {
		// For TLS, we need to use ListenAndServeTLS, but since we're using TLSConfig,
		// we'll use ListenAndServe with the TLS config already set
		// However, ListenAndServeTLS is simpler for this case
		if cfg.Server.CertFile != "" && cfg.Server.KeyFile != "" {
			if err := server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed: %v", err)
			}
		} else {
			// Self-signed cert - need to use custom listener
			listener, listenErr := tls.Listen("tcp", addr, tlsConfig)
			if listenErr != nil {
				log.Fatalf("Failed to create TLS listener: %v", listenErr)
			}
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed: %v", err)
			}
		}
	} else {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
