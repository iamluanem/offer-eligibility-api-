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
	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/handler"
	"offer-eligibility-api/internal/middleware"
	"offer-eligibility-api/internal/service"
	tlsconfig "offer-eligibility-api/internal/tls"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const (
	defaultPort        = "8080"
	defaultDBPath      = "./offer_eligibility.db"
	defaultRateLimit   = 100  // requests per window
	defaultRateWindow  = 60   // seconds
)

func main() {
	port := flag.String("port", defaultPort, "Server port")
	dbPath := flag.String("db", defaultDBPath, "Database file path")
	rateLimit := flag.Int("rate-limit", defaultRateLimit, "Rate limit (requests per window)")
	rateWindow := flag.Int("rate-window", defaultRateWindow, "Rate limit window in seconds")
	enableTLS := flag.Bool("tls", false, "Enable HTTPS/TLS")
	certFile := flag.String("cert", "", "TLS certificate file path (required if -tls is set)")
	keyFile := flag.String("key", "", "TLS private key file path (required if -tls is set)")
	flag.Parse()

	// Initialize database
	db, err := database.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize service
	svc := service.NewService(db)

	// Initialize handlers
	h := handler.NewHandler(svc)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(*rateLimit, time.Duration(*rateWindow)*time.Second)
	defer rateLimiter.Stop()

	// Setup router
	r := chi.NewRouter()

	// Middleware (order matters)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	
	// Rate limiting middleware
	r.Use(middleware.RateLimitMiddleware(rateLimiter))
	
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
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
	if *enableTLS {
		tlsCfg := tlsconfig.Config{
			CertFile: *certFile,
			KeyFile:  *keyFile,
		}

		var err error
		tlsConfig, err = tlsconfig.LoadTLSConfig(tlsCfg)
		if err != nil {
			log.Fatalf("Failed to load TLS configuration: %v", err)
		}

		if *certFile == "" || *keyFile == "" {
			log.Println("WARNING: No certificate files provided, using self-signed certificate for development")
		}
	}

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	protocol := "HTTP"
	if *enableTLS {
		protocol = "HTTPS"
	}
	log.Printf("Starting %s server on %s", protocol, addr)
	log.Printf("Database: %s", *dbPath)
	log.Printf("Rate limit: %d requests per %d seconds", *rateLimit, *rateWindow)

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

	if *enableTLS {
		// For TLS, we need to use ListenAndServeTLS, but since we're using TLSConfig,
		// we'll use ListenAndServe with the TLS config already set
		// However, ListenAndServeTLS is simpler for this case
		if *certFile != "" && *keyFile != "" {
			if err := server.ListenAndServeTLS(*certFile, *keyFile); err != nil && err != http.ErrServerClosed {
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

	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
