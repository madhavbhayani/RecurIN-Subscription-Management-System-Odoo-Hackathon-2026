package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/config"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/internal/migrations"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/queue"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/rbac"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/handlers"
)

// Application holds the top-level backend dependencies.
type Application struct {
	Config       config.Config
	DB           *pgxpool.Pool
	Queue        *queue.WorkerPool
	TokenManager *auth.TokenManager
	HTTPServer   *http.Server
}

// NewApplication wires app dependencies and routes.
func NewApplication(ctx context.Context) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	dbPool, err := config.NewPostgresPool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := migrations.ApplyUpMigrations(ctx, dbPool, cfg.MigrationsDir); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to apply database migrations: %w", err)
	}

	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTExpiryMinutes)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to initialize token manager: %w", err)
	}

	workerPool := queue.NewWorkerPool(cfg.QueueWorkerCount, cfg.QueueBufferSize)
	workerPool.Start()

	router := http.NewServeMux()
	registerRoutes(router, tokenManager)

	httpServer := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Application{
		Config:       cfg,
		DB:           dbPool,
		Queue:        workerPool,
		TokenManager: tokenManager,
		HTTPServer:   httpServer,
	}, nil
}

// Start runs the HTTP server.
func (a *Application) Start() error {
	log.Printf("starting backend server on %s", a.HTTPServer.Addr)
	return a.HTTPServer.ListenAndServe()
}

// Shutdown closes server and shared resources.
func (a *Application) Shutdown(ctx context.Context) error {
	a.Queue.Stop()
	a.DB.Close()
	return a.HTTPServer.Shutdown(ctx)
}

func registerRoutes(router *http.ServeMux, tokenManager *auth.TokenManager) {
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(tokenManager)

	router.HandleFunc("GET /api/v1/health", healthHandler.HandleHealth)
	router.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)

	authenticatedRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user", "portal-user")(
			http.HandlerFunc(authHandler.HandleWhoAmI),
		),
	)
	router.Handle("GET /api/v1/auth/me", authenticatedRoute)

	adminRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin")(
			http.HandlerFunc(authHandler.HandleAdminPing),
		),
	)
	router.Handle("GET /api/v1/admin/ping", adminRoute)
}
