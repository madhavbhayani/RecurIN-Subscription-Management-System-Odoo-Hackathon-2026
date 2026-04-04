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
	roleService := services.NewRoleService(dbPool)
	quoteNotifier := services.NewSubscriptionQuoteNotifier(services.SubscriptionQuoteNotifierConfig{
		SMTPHost:        cfg.SMTPHost,
		SMTPPort:        cfg.SMTPPort,
		SMTPUsername:    cfg.SMTPUsername,
		SMTPPassword:    cfg.SMTPPassword,
		SMTPFromEmail:   cfg.SMTPFromEmail,
		SMTPFromName:    cfg.SMTPFromName,
		FrontendBaseURL: cfg.FrontendBaseURL,
	})
	subscriptionService := services.NewSubscriptionService(dbPool, quoteNotifier)

	router := http.NewServeMux()
	registerRoutes(router, dbPool, tokenManager, workerPool, userService, attributeService, taxService, productService, recurringPlanService, quotationService, paymentTermService, discountService, subscriptionService, roleService)

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
	dbPool *pgxpool.Pool,
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
	roleService *services.RoleService,
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
	roleHandler := handlers.NewRoleHandler(roleService)

	adminRoleRoute := func(handler http.Handler) http.Handler {
		return auth.AuthMiddleware(tokenManager)(
			rbac.RequireRoles("admin", "internal-user")(
				handler,
			),
		)
	}

	adminPermissionRoute := func(resourceKey string, action rbac.PermissionAction, handler http.Handler) http.Handler {
		return auth.AuthMiddleware(tokenManager)(
			rbac.RequireRoles("admin", "internal-user")(
				rbac.RequirePermission(dbPool, resourceKey, action)(
					handler,
				),
			),
		)
	}

	router.HandleFunc("GET /api/v1/health", healthHandler.HandleHealth)
	router.HandleFunc("POST /api/v1/auth/signup", authHandler.HandleSignup)
	router.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)
	router.HandleFunc("GET /api/v1/products", productHandler.HandleListProducts)
	router.HandleFunc("GET /api/v1/recurring-plans", recurringPlanHandler.HandleListRecurringPlans)

	authenticatedRoute := auth.AuthMiddleware(tokenManager)(
		rbac.RequireRoles("admin", "internal-user", "user", "portal-user")(
			http.HandlerFunc(authHandler.HandleWhoAmI),
		),
	)
	router.Handle("GET /api/v1/auth/me", authenticatedRoute)

	router.Handle("GET /api/v1/admin/ping", adminRoleRoute(http.HandlerFunc(authHandler.HandleAdminPing)))

	router.Handle(
		"GET /api/v1/admin/users/customers",
		adminPermissionRoute(rbac.ResourceUsers, rbac.PermissionActionRead, http.HandlerFunc(userHandler.HandleListCustomerUsers)),
	)
	router.Handle(
		"GET /api/v1/admin/users",
		adminPermissionRoute(rbac.ResourceUsers, rbac.PermissionActionRead, http.HandlerFunc(userHandler.HandleListUsers)),
	)
	router.Handle(
		"GET /api/v1/admin/users/{userID}",
		adminPermissionRoute(rbac.ResourceUsers, rbac.PermissionActionRead, http.HandlerFunc(userHandler.HandleGetUserByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/users/{userID}",
		adminPermissionRoute(rbac.ResourceUsers, rbac.PermissionActionUpdate, http.HandlerFunc(userHandler.HandleUpdateUser)),
	)
	router.Handle(
		"DELETE /api/v1/admin/users/{userID}",
		adminPermissionRoute(rbac.ResourceUsers, rbac.PermissionActionDelete, http.HandlerFunc(userHandler.HandleDeleteUser)),
	)

	router.Handle(
		"POST /api/v1/admin/attributes",
		adminPermissionRoute(rbac.ResourceConfigurationsAttribute, rbac.PermissionActionCreate, http.HandlerFunc(attributeHandler.HandleCreateAttribute)),
	)
	router.Handle(
		"GET /api/v1/admin/attributes",
		adminPermissionRoute(rbac.ResourceConfigurationsAttribute, rbac.PermissionActionRead, http.HandlerFunc(attributeHandler.HandleListAttributes)),
	)
	router.Handle(
		"GET /api/v1/admin/attributes/{attributeID}",
		adminPermissionRoute(rbac.ResourceConfigurationsAttribute, rbac.PermissionActionRead, http.HandlerFunc(attributeHandler.HandleGetAttributeByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/attributes/{attributeID}",
		adminPermissionRoute(rbac.ResourceConfigurationsAttribute, rbac.PermissionActionUpdate, http.HandlerFunc(attributeHandler.HandleUpdateAttribute)),
	)
	router.Handle(
		"DELETE /api/v1/admin/attributes/{attributeID}",
		adminPermissionRoute(rbac.ResourceConfigurationsAttribute, rbac.PermissionActionDelete, http.HandlerFunc(attributeHandler.HandleDeleteAttribute)),
	)

	router.Handle(
		"POST /api/v1/admin/taxes",
		adminPermissionRoute(rbac.ResourceConfigurationsTaxes, rbac.PermissionActionCreate, http.HandlerFunc(taxHandler.HandleCreateTax)),
	)
	router.Handle(
		"GET /api/v1/admin/taxes",
		adminPermissionRoute(rbac.ResourceConfigurationsTaxes, rbac.PermissionActionRead, http.HandlerFunc(taxHandler.HandleListTaxes)),
	)
	router.Handle(
		"GET /api/v1/admin/taxes/{taxID}",
		adminPermissionRoute(rbac.ResourceConfigurationsTaxes, rbac.PermissionActionRead, http.HandlerFunc(taxHandler.HandleGetTaxByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/taxes/{taxID}",
		adminPermissionRoute(rbac.ResourceConfigurationsTaxes, rbac.PermissionActionUpdate, http.HandlerFunc(taxHandler.HandleUpdateTax)),
	)
	router.Handle(
		"DELETE /api/v1/admin/taxes/{taxID}",
		adminPermissionRoute(rbac.ResourceConfigurationsTaxes, rbac.PermissionActionDelete, http.HandlerFunc(taxHandler.HandleDeleteTax)),
	)

	router.Handle(
		"POST /api/v1/admin/products",
		adminPermissionRoute(rbac.ResourceProducts, rbac.PermissionActionCreate, http.HandlerFunc(productHandler.HandleCreateProduct)),
	)
	router.Handle(
		"GET /api/v1/admin/products",
		adminPermissionRoute(rbac.ResourceProducts, rbac.PermissionActionRead, http.HandlerFunc(productHandler.HandleListProducts)),
	)
	router.Handle(
		"GET /api/v1/admin/products/{productID}",
		adminPermissionRoute(rbac.ResourceProducts, rbac.PermissionActionRead, http.HandlerFunc(productHandler.HandleGetProductByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/products/{productID}",
		adminPermissionRoute(rbac.ResourceProducts, rbac.PermissionActionUpdate, http.HandlerFunc(productHandler.HandleUpdateProduct)),
	)
	router.Handle(
		"DELETE /api/v1/admin/products/{productID}",
		adminPermissionRoute(rbac.ResourceProducts, rbac.PermissionActionDelete, http.HandlerFunc(productHandler.HandleDeleteProduct)),
	)

	router.Handle(
		"POST /api/v1/admin/recurring-plans",
		adminPermissionRoute(rbac.ResourceConfigurationsRecurring, rbac.PermissionActionCreate, http.HandlerFunc(recurringPlanHandler.HandleCreateRecurringPlan)),
	)
	router.Handle(
		"GET /api/v1/admin/recurring-plans",
		adminPermissionRoute(rbac.ResourceConfigurationsRecurring, rbac.PermissionActionRead, http.HandlerFunc(recurringPlanHandler.HandleListRecurringPlans)),
	)
	router.Handle(
		"GET /api/v1/admin/recurring-plans/{recurringPlanID}",
		adminPermissionRoute(rbac.ResourceConfigurationsRecurring, rbac.PermissionActionRead, http.HandlerFunc(recurringPlanHandler.HandleGetRecurringPlanByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/recurring-plans/{recurringPlanID}",
		adminPermissionRoute(rbac.ResourceConfigurationsRecurring, rbac.PermissionActionUpdate, http.HandlerFunc(recurringPlanHandler.HandleUpdateRecurringPlan)),
	)
	router.Handle(
		"DELETE /api/v1/admin/recurring-plans/{recurringPlanID}",
		adminPermissionRoute(rbac.ResourceConfigurationsRecurring, rbac.PermissionActionDelete, http.HandlerFunc(recurringPlanHandler.HandleDeleteRecurringPlan)),
	)

	router.Handle(
		"POST /api/v1/admin/quotations",
		adminPermissionRoute(rbac.ResourceConfigurationsQuotation, rbac.PermissionActionCreate, http.HandlerFunc(quotationHandler.HandleCreateQuotation)),
	)
	router.Handle(
		"GET /api/v1/admin/quotations",
		adminPermissionRoute(rbac.ResourceConfigurationsQuotation, rbac.PermissionActionRead, http.HandlerFunc(quotationHandler.HandleListQuotations)),
	)
	router.Handle(
		"GET /api/v1/admin/quotations/{quotationID}",
		adminPermissionRoute(rbac.ResourceConfigurationsQuotation, rbac.PermissionActionRead, http.HandlerFunc(quotationHandler.HandleGetQuotationByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/quotations/{quotationID}",
		adminPermissionRoute(rbac.ResourceConfigurationsQuotation, rbac.PermissionActionUpdate, http.HandlerFunc(quotationHandler.HandleUpdateQuotation)),
	)
	router.Handle(
		"DELETE /api/v1/admin/quotations/{quotationID}",
		adminPermissionRoute(rbac.ResourceConfigurationsQuotation, rbac.PermissionActionDelete, http.HandlerFunc(quotationHandler.HandleDeleteQuotation)),
	)

	router.Handle(
		"POST /api/v1/admin/payment-terms",
		adminPermissionRoute(rbac.ResourceConfigurationsPaymentTerm, rbac.PermissionActionCreate, http.HandlerFunc(paymentTermHandler.HandleCreatePaymentTerm)),
	)
	router.Handle(
		"GET /api/v1/admin/payment-terms",
		adminPermissionRoute(rbac.ResourceConfigurationsPaymentTerm, rbac.PermissionActionRead, http.HandlerFunc(paymentTermHandler.HandleListPaymentTerms)),
	)
	router.Handle(
		"GET /api/v1/admin/payment-terms/{paymentTermID}",
		adminPermissionRoute(rbac.ResourceConfigurationsPaymentTerm, rbac.PermissionActionRead, http.HandlerFunc(paymentTermHandler.HandleGetPaymentTermByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/payment-terms/{paymentTermID}",
		adminPermissionRoute(rbac.ResourceConfigurationsPaymentTerm, rbac.PermissionActionUpdate, http.HandlerFunc(paymentTermHandler.HandleUpdatePaymentTerm)),
	)
	router.Handle(
		"DELETE /api/v1/admin/payment-terms/{paymentTermID}",
		adminPermissionRoute(rbac.ResourceConfigurationsPaymentTerm, rbac.PermissionActionDelete, http.HandlerFunc(paymentTermHandler.HandleDeletePaymentTerm)),
	)

	router.Handle(
		"POST /api/v1/admin/discounts",
		adminPermissionRoute(rbac.ResourceConfigurationsDiscount, rbac.PermissionActionCreate, http.HandlerFunc(discountHandler.HandleCreateDiscount)),
	)
	router.Handle(
		"GET /api/v1/admin/discounts",
		adminPermissionRoute(rbac.ResourceConfigurationsDiscount, rbac.PermissionActionRead, http.HandlerFunc(discountHandler.HandleListDiscounts)),
	)
	router.Handle(
		"GET /api/v1/admin/discounts/{discountID}",
		adminPermissionRoute(rbac.ResourceConfigurationsDiscount, rbac.PermissionActionRead, http.HandlerFunc(discountHandler.HandleGetDiscountByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/discounts/{discountID}",
		adminPermissionRoute(rbac.ResourceConfigurationsDiscount, rbac.PermissionActionUpdate, http.HandlerFunc(discountHandler.HandleUpdateDiscount)),
	)
	router.Handle(
		"DELETE /api/v1/admin/discounts/{discountID}",
		adminPermissionRoute(rbac.ResourceConfigurationsDiscount, rbac.PermissionActionDelete, http.HandlerFunc(discountHandler.HandleDeleteDiscount)),
	)

	router.Handle(
		"POST /api/v1/admin/subscriptions",
		adminPermissionRoute(rbac.ResourceSubscriptions, rbac.PermissionActionCreate, http.HandlerFunc(subscriptionHandler.HandleCreateSubscription)),
	)
	router.Handle(
		"GET /api/v1/admin/subscriptions",
		adminPermissionRoute(rbac.ResourceSubscriptions, rbac.PermissionActionRead, http.HandlerFunc(subscriptionHandler.HandleListSubscriptions)),
	)
	router.Handle(
		"GET /api/v1/admin/subscriptions/{subscriptionID}",
		adminPermissionRoute(rbac.ResourceSubscriptions, rbac.PermissionActionRead, http.HandlerFunc(subscriptionHandler.HandleGetSubscriptionByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/subscriptions/{subscriptionID}",
		adminPermissionRoute(rbac.ResourceSubscriptions, rbac.PermissionActionUpdate, http.HandlerFunc(subscriptionHandler.HandleUpdateSubscription)),
	)
	router.Handle(
		"DELETE /api/v1/admin/subscriptions/{subscriptionID}",
		adminPermissionRoute(rbac.ResourceSubscriptions, rbac.PermissionActionDelete, http.HandlerFunc(subscriptionHandler.HandleDeleteSubscription)),
	)

	router.Handle(
		"POST /api/v1/admin/roles",
		adminPermissionRoute(rbac.ResourceRoles, rbac.PermissionActionCreate, http.HandlerFunc(roleHandler.HandleCreateRole)),
	)
	router.Handle(
		"GET /api/v1/admin/roles",
		adminPermissionRoute(rbac.ResourceRoles, rbac.PermissionActionRead, http.HandlerFunc(roleHandler.HandleListRoles)),
	)
	router.Handle(
		"GET /api/v1/admin/roles/{roleID}",
		adminPermissionRoute(rbac.ResourceRoles, rbac.PermissionActionRead, http.HandlerFunc(roleHandler.HandleGetRoleByID)),
	)
	router.Handle(
		"PUT /api/v1/admin/roles/{roleID}",
		adminPermissionRoute(rbac.ResourceRoles, rbac.PermissionActionUpdate, http.HandlerFunc(roleHandler.HandleUpdateRole)),
	)
	router.Handle(
		"DELETE /api/v1/admin/roles/{roleID}",
		adminPermissionRoute(rbac.ResourceRoles, rbac.PermissionActionDelete, http.HandlerFunc(roleHandler.HandleDeleteRole)),
	)
}
