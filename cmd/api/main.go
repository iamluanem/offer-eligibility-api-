package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"offer-eligibility-api/internal/database"
	"offer-eligibility-api/internal/handler"
	"offer-eligibility-api/internal/middleware"
	"offer-eligibility-api/internal/service"
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

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Database: %s", *dbPath)
	log.Printf("Rate limit: %d requests per %d seconds", *rateLimit, *rateWindow)

	server := &http.Server{
		Addr:    addr,
		Handler: r,
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

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
