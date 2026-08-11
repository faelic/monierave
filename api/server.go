package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/observability"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config                 util.Config
	store                  db.Store
	tokenMaker             token.Maker
	emailVerificationMaker token.EmailVerificationMaker
	router                 *gin.Engine
	databaseReady          ReadinessCheck
	redisReady             ReadinessCheck
	rateLimiter            RateLimiter
	metrics                *observability.Registry
	passwordBreachChecker  PasswordBreachChecker
}

func NewServer(
	config util.Config,
	store db.Store,
	options ...ServerOption,
) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate token maker: %w", err)
	}
	emailVerificationMaker, err := token.NewEmailVerificationMaker(config.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate email verification token maker: %w", err)
	}

	server := &Server{
		config:                 config,
		store:                  store,
		tokenMaker:             tokenMaker,
		emailVerificationMaker: emailVerificationMaker,
		metrics:                observability.Default,
		passwordBreachChecker:  noopPasswordBreachChecker{},
	}
	for _, option := range options {
		option(server)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}
	binding.EnableDecoderDisallowUnknownFields = true

	server.setupRouter()

	return server, nil
}

func WithPasswordBreachChecker(checker PasswordBreachChecker) ServerOption {
	return func(server *Server) {
		if checker != nil {
			server.passwordBreachChecker = checker
		}
	}
}

func (server *Server) setupRouter() {
	router := gin.New()

	router.Use(
		server.requestBodyLimitMiddleware(),
		server.requestContextMiddleware(),
		server.requestLoggerMiddleware(),
		server.recoveryMiddleware(),
		server.corsMiddleware(),
	)
	_ = router.SetTrustedProxies(nil)

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Monierave API is running"})
	})
	router.GET("/livez", server.live)
	router.GET("/readyz", server.requireOperationsToken(), server.ready)
	router.GET("/metrics", server.requireOperationsToken(), gin.WrapH(server.metricsHandler()))
	router.POST(
		"/users",
		server.requireTrustedBrowserOrigin(),
		server.rateLimitMiddleware("signup", 5, time.Hour, clientIPRateLimitKey),
		server.CreateUser,
	)
	router.POST(
		"/users/login",
		server.requireTrustedBrowserOrigin(),
		server.rateLimitMiddleware("login", 5, time.Minute, clientIPRateLimitKey),
		server.loginUser,
	)
	router.GET("/users/verify-email", server.verifyUserEmail)
	router.POST("/users/verify-email", server.confirmUserEmail)
	router.POST(
		"/tokens/renew_access",
		server.rateLimitMiddleware("refresh", 10, time.Minute, clientIPRateLimitKey),
		server.renewAccessToken,
	)
	router.POST("/sessions/logout", server.logoutCurrentSession)
	router.POST("/webhooks/resend", server.handleResendWebhook)

	authRoutes := router.Group("/").Use(authMiddleware(
		server.tokenMaker,
		server.store,
		server.config.DeviceCookieName,
	))
	authRoutes.GET("/users/me", server.getCurrentUser)
	authRoutes.PATCH("/users/me", server.updateUser)
	authRoutes.GET("/users/me/email-status", server.getUserEmailStatus)
	authRoutes.POST(
		"/users/me/resend-verification",
		server.rateLimitMiddleware(
			"resend_verification",
			3,
			time.Hour,
			authenticatedRateLimitKey,
		),
		server.resendUserEmailVerification,
	)
	authRoutes.POST("/sessions/logout-all", server.logoutAllSessions)

	financialMiddleware := []gin.HandlerFunc{
		authMiddleware(
			server.tokenMaker,
			server.store,
			server.config.DeviceCookieName,
		),
	}
	if server.config.EnforceEmailVerification {
		financialMiddleware = append(
			financialMiddleware,
			verifiedAccountMiddleware(server.store),
		)
	}
	financialRoutes := router.Group("/").Use(financialMiddleware...)
	financialRoutes.POST("/accounts", server.createAccount)
	financialRoutes.POST(
		"/accounts/resolve",
		server.rateLimitMiddleware(
			"recipient_resolve",
			20,
			time.Minute,
			authenticatedRateLimitKey,
		),
		server.resolveRecipient,
	)
	financialRoutes.GET("/accounts/:public_id", server.getAccount)
	financialRoutes.GET("/accounts", server.listAccount)
	financialRoutes.GET("/accounts/:public_id/transactions", server.listAccountTransactions)
	financialRoutes.GET("/accounts/:public_id/statement", server.getAccountStatement)
	financialRoutes.POST("/accounts/:public_id/close", server.closeAccount)
	financialRoutes.POST("/beneficiaries", server.createBeneficiary)
	financialRoutes.GET("/beneficiaries", server.listBeneficiaries)
	financialRoutes.PATCH("/beneficiaries/:id", server.updateBeneficiary)
	financialRoutes.DELETE("/beneficiaries/:id", server.deleteBeneficiary)
	financialRoutes.GET("/transactions/:reference", server.getTransaction)
	financialRoutes.POST(
		"/transfers",
		server.rateLimitMiddleware(
			"transfer",
			30,
			time.Minute,
			authenticatedRateLimitKey,
		),
		server.createTransfer,
	)

	server.router = router
}

// Start runs the http server on a particular address
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func (server *Server) Handler() http.Handler {
	return server.router
}

func errorResponse(ctx *gin.Context, err error) gin.H {
	return codedErrorResponse(ctx, stableErrorCode(err), err)
}

func codedErrorResponse(ctx *gin.Context, code string, err error) gin.H {
	return gin.H{
		"code":       code,
		"message":    err.Error(),
		"error":      err.Error(),
		"request_id": requestID(ctx),
	}
}
