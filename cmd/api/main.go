package main

import (
	"log"
	"net/http"
	"time"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/config"
	"bookwork-api/internal/database"
	"bookwork-api/internal/handlers"
	customMiddleware "bookwork-api/internal/middleware"
	"bookwork-api/internal/migrations"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.New(database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		PgBouncerAddr:   cfg.Database.PgBouncerAddr,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	migrator := migrations.NewMigrator(db.DB)
	if err := migrator.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize auth service
	authService := auth.NewService(cfg.JWT.SecretKey, cfg.JWT.Issuer)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, authService)
	clubHandler := handlers.NewClubHandler(db)
	eventHandler := handlers.NewEventHandler(db)
	eventItemHandler := handlers.NewEventItemHandler(db)
	availabilityHandler := handlers.NewAvailabilityHandler(db)
	healthHandler := handlers.NewHealthHandler(db.DB)

	// Setup router
	r := chi.NewRouter()

	// Security middleware with configuration
	r.Use(customMiddleware.SecurityHeadersWithConfig(
		customMiddleware.SecurityConfig{
			EnableHSTS:      cfg.Security.EnableHSTS,
			HSTSMaxAge:      cfg.Security.HSTSMaxAge,
			EnableHTTPSOnly: cfg.Security.EnableHTTPSOnly,
		},
	))

	// Rate limiting (100 requests per minute)
	rateLimiter := customMiddleware.NewRateLimiter(100, time.Minute)
	r.Use(rateLimiter.Middleware)

	// Standard middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Heartbeat("/healthz"))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Health and monitoring routes (no auth required)
		r.Mount("/", healthHandler.RegisterRoutes())

		// Public authentication routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(authService.AuthMiddleware)
				r.Post("/validate", authHandler.Validate)
				r.Post("/logout", authHandler.Logout)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authService.AuthMiddleware)

			// Club member management
			r.Route("/club/{clubId}/members", func(r chi.Router) {
				r.Get("/", clubHandler.GetMembers)
				r.Post("/", clubHandler.AddMember)
				r.Put("/{memberId}", clubHandler.UpdateMember)
				r.Delete("/{memberId}", clubHandler.RemoveMember)
			})

			// Club events
			r.Route("/club/{clubId}/events", func(r chi.Router) {
				r.Get("/", eventHandler.GetEvents)
				r.Post("/", eventHandler.CreateEvent)
			})

			// Event management
			r.Route("/events/{eventId}", func(r chi.Router) {
				r.Put("/", eventHandler.UpdateEvent)
				r.Delete("/", eventHandler.DeleteEvent)

				// Event items
				r.Route("/items", func(r chi.Router) {
					r.Get("/", eventItemHandler.GetItems)
					r.Post("/", eventItemHandler.CreateItem)
					r.Put("/{itemId}", eventItemHandler.UpdateItem)
					r.Delete("/{itemId}", eventItemHandler.DeleteItem)
				})

				// Event availability
				r.Route("/availability", func(r chi.Router) {
					r.Get("/", availabilityHandler.GetAvailability)
					r.Post("/", availabilityHandler.UpdateAvailability)
				})
			})
		})
	})

	// Health check endpoint
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// Start server
	addr := ":" + cfg.Server.Port
	log.Printf("Starting server on %s", addr)
	log.Printf("Health check available at http://localhost%s/healthz", addr)
	log.Printf("API base URL: http://localhost%s/api", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
