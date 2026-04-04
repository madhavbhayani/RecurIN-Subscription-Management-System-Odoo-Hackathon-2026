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
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
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
	userService := services.NewUserService(dbPool)
	attributeService := services.NewAttributeService(dbPool)
	taxService := services.NewTaxService(dbPool)
	productService := services.NewProductService(dbPool)
	recurringPlanService := services.NewRecurringPlanService(dbPool)
	quotationService := services.NewQuotationService(dbPool)
	paymentTermService := services.NewPaymentTermService(dbPool)
	discountService := services.NewDiscountService(dbPool)
	subscriptionService := services.NewSubscriptionService(dbPool)

	router := http.NewServeMux()
	registerRoutes(router, tokenManager, workerPool, userService, attributeService, taxService, productService, recurringPlanService, quotationService, paymentTermService, discountService, subscriptionService)

	httpServer := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           withCORS(router),
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

func registerRoutes(
	router *http.ServeMux,
	tokenManager *auth.TokenManager,
	workerPool *queue.WorkerPool,
	userService *services.UserService,
	attributeService *services.AttributeService,
	taxService *services.TaxService,
	productService *services.ProductService,
	recurringPlanService *services.RecurringPlanService,
	quotationService *services.QuotationService,
	paymentTermService *services.PaymentTermService,
	discountService *services.DiscountService,
	subscriptionService *services.SubscriptionService,
) {
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(tokenManager, workerPool, userService)
	userHandler := handlers.NewUserHandler(userService)
	attributeHandler := handlers.NewAttributeHandler(attributeService)
	taxHandler := handlers.NewTaxHandler(taxService)
	productHandler := handlers.NewProductHandler(productService)
	recurringPlanHandler := handlers.NewRecurringPlanHandler(recurringPlanService)
	quotationHandler := handlers.NewQuotationHandler(quotationService)
	paymentTermHandler := handlers.NewPaymentTermHandler(paymentTermService)
	discountHandler := handlers.NewDiscountHandler(discountService)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService)

	router.HandleFunc("GET /api/v1/health", healthHandler.HandleHealth)
	router.HandleFunc("POST /api/v1/auth/signup", authHandler.HandleSignup)
	router.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)

	authenticatedRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user", "user", "portal-user")(
			http.HandlerFunc(authHandler.HandleWhoAmI),
		),
	)
	router.Handle("GET /api/v1/auth/me", authenticatedRoute)

	adminRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(authHandler.HandleAdminPing),
		),
	)
	router.Handle("GET /api/v1/admin/ping", adminRoute)

	adminListCustomerUsersRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(userHandler.HandleListCustomerUsers),
		),
	)
	router.Handle("GET /api/v1/admin/users/customers", adminListCustomerUsersRoute)

	adminCreateAttributeRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(attributeHandler.HandleCreateAttribute),
		),
	)
	router.Handle("POST /api/v1/admin/attributes", adminCreateAttributeRoute)

	adminListAttributesRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(attributeHandler.HandleListAttributes),
		),
	)
	router.Handle("GET /api/v1/admin/attributes", adminListAttributesRoute)

	adminGetAttributeByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(attributeHandler.HandleGetAttributeByID),
		),
	)
	router.Handle("GET /api/v1/admin/attributes/{attributeID}", adminGetAttributeByIDRoute)

	adminUpdateAttributeRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(attributeHandler.HandleUpdateAttribute),
		),
	)
	router.Handle("PUT /api/v1/admin/attributes/{attributeID}", adminUpdateAttributeRoute)

	adminDeleteAttributeRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(attributeHandler.HandleDeleteAttribute),
		),
	)
	router.Handle("DELETE /api/v1/admin/attributes/{attributeID}", adminDeleteAttributeRoute)

	adminCreateTaxRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(taxHandler.HandleCreateTax),
		),
	)
	router.Handle("POST /api/v1/admin/taxes", adminCreateTaxRoute)

	adminListTaxesRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(taxHandler.HandleListTaxes),
		),
	)
	router.Handle("GET /api/v1/admin/taxes", adminListTaxesRoute)

	adminGetTaxByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(taxHandler.HandleGetTaxByID),
		),
	)
	router.Handle("GET /api/v1/admin/taxes/{taxID}", adminGetTaxByIDRoute)

	adminUpdateTaxRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(taxHandler.HandleUpdateTax),
		),
	)
	router.Handle("PUT /api/v1/admin/taxes/{taxID}", adminUpdateTaxRoute)

	adminDeleteTaxRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(taxHandler.HandleDeleteTax),
		),
	)
	router.Handle("DELETE /api/v1/admin/taxes/{taxID}", adminDeleteTaxRoute)

	adminCreateProductRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(productHandler.HandleCreateProduct),
		),
	)
	router.Handle("POST /api/v1/admin/products", adminCreateProductRoute)

	adminListProductsRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(productHandler.HandleListProducts),
		),
	)
	router.Handle("GET /api/v1/admin/products", adminListProductsRoute)

	adminGetProductByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(productHandler.HandleGetProductByID),
		),
	)
	router.Handle("GET /api/v1/admin/products/{productID}", adminGetProductByIDRoute)

	adminUpdateProductRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(productHandler.HandleUpdateProduct),
		),
	)
	router.Handle("PUT /api/v1/admin/products/{productID}", adminUpdateProductRoute)

	adminDeleteProductRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(productHandler.HandleDeleteProduct),
		),
	)
	router.Handle("DELETE /api/v1/admin/products/{productID}", adminDeleteProductRoute)

	adminCreateRecurringPlanRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(recurringPlanHandler.HandleCreateRecurringPlan),
		),
	)
	router.Handle("POST /api/v1/admin/recurring-plans", adminCreateRecurringPlanRoute)

	adminListRecurringPlansRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(recurringPlanHandler.HandleListRecurringPlans),
		),
	)
	router.Handle("GET /api/v1/admin/recurring-plans", adminListRecurringPlansRoute)

	adminGetRecurringPlanByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(recurringPlanHandler.HandleGetRecurringPlanByID),
		),
	)
	router.Handle("GET /api/v1/admin/recurring-plans/{recurringPlanID}", adminGetRecurringPlanByIDRoute)

	adminUpdateRecurringPlanRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(recurringPlanHandler.HandleUpdateRecurringPlan),
		),
	)
	router.Handle("PUT /api/v1/admin/recurring-plans/{recurringPlanID}", adminUpdateRecurringPlanRoute)

	adminDeleteRecurringPlanRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(recurringPlanHandler.HandleDeleteRecurringPlan),
		),
	)
	router.Handle("DELETE /api/v1/admin/recurring-plans/{recurringPlanID}", adminDeleteRecurringPlanRoute)

	adminCreateQuotationRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(quotationHandler.HandleCreateQuotation),
		),
	)
	router.Handle("POST /api/v1/admin/quotations", adminCreateQuotationRoute)

	adminListQuotationsRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(quotationHandler.HandleListQuotations),
		),
	)
	router.Handle("GET /api/v1/admin/quotations", adminListQuotationsRoute)

	adminGetQuotationByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(quotationHandler.HandleGetQuotationByID),
		),
	)
	router.Handle("GET /api/v1/admin/quotations/{quotationID}", adminGetQuotationByIDRoute)

	adminUpdateQuotationRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(quotationHandler.HandleUpdateQuotation),
		),
	)
	router.Handle("PUT /api/v1/admin/quotations/{quotationID}", adminUpdateQuotationRoute)

	adminDeleteQuotationRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(quotationHandler.HandleDeleteQuotation),
		),
	)
	router.Handle("DELETE /api/v1/admin/quotations/{quotationID}", adminDeleteQuotationRoute)

	adminCreatePaymentTermRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(paymentTermHandler.HandleCreatePaymentTerm),
		),
	)
	router.Handle("POST /api/v1/admin/payment-terms", adminCreatePaymentTermRoute)

	adminListPaymentTermsRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(paymentTermHandler.HandleListPaymentTerms),
		),
	)
	router.Handle("GET /api/v1/admin/payment-terms", adminListPaymentTermsRoute)

	adminGetPaymentTermByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(paymentTermHandler.HandleGetPaymentTermByID),
		),
	)
	router.Handle("GET /api/v1/admin/payment-terms/{paymentTermID}", adminGetPaymentTermByIDRoute)

	adminUpdatePaymentTermRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(paymentTermHandler.HandleUpdatePaymentTerm),
		),
	)
	router.Handle("PUT /api/v1/admin/payment-terms/{paymentTermID}", adminUpdatePaymentTermRoute)

	adminDeletePaymentTermRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(paymentTermHandler.HandleDeletePaymentTerm),
		),
	)
	router.Handle("DELETE /api/v1/admin/payment-terms/{paymentTermID}", adminDeletePaymentTermRoute)

	adminCreateDiscountRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(discountHandler.HandleCreateDiscount),
		),
	)
	router.Handle("POST /api/v1/admin/discounts", adminCreateDiscountRoute)

	adminListDiscountsRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(discountHandler.HandleListDiscounts),
		),
	)
	router.Handle("GET /api/v1/admin/discounts", adminListDiscountsRoute)

	adminGetDiscountByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(discountHandler.HandleGetDiscountByID),
		),
	)
	router.Handle("GET /api/v1/admin/discounts/{discountID}", adminGetDiscountByIDRoute)

	adminUpdateDiscountRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(discountHandler.HandleUpdateDiscount),
		),
	)
	router.Handle("PUT /api/v1/admin/discounts/{discountID}", adminUpdateDiscountRoute)

	adminDeleteDiscountRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(discountHandler.HandleDeleteDiscount),
		),
	)
	router.Handle("DELETE /api/v1/admin/discounts/{discountID}", adminDeleteDiscountRoute)

	adminCreateSubscriptionRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(subscriptionHandler.HandleCreateSubscription),
		),
	)
	router.Handle("POST /api/v1/admin/subscriptions", adminCreateSubscriptionRoute)

	adminListSubscriptionsRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(subscriptionHandler.HandleListSubscriptions),
		),
	)
	router.Handle("GET /api/v1/admin/subscriptions", adminListSubscriptionsRoute)

	adminGetSubscriptionByIDRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(subscriptionHandler.HandleGetSubscriptionByID),
		),
	)
	router.Handle("GET /api/v1/admin/subscriptions/{subscriptionID}", adminGetSubscriptionByIDRoute)

	adminUpdateSubscriptionRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(subscriptionHandler.HandleUpdateSubscription),
		),
	)
	router.Handle("PUT /api/v1/admin/subscriptions/{subscriptionID}", adminUpdateSubscriptionRoute)

	adminDeleteSubscriptionRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user")(
			http.HandlerFunc(subscriptionHandler.HandleDeleteSubscription),
		),
	)
	router.Handle("DELETE /api/v1/admin/subscriptions/{subscriptionID}", adminDeleteSubscriptionRoute)
}
