package api

import (
	"fmt"
	"net/http"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
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
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
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
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.New()

	router.Use(gin.Logger(), gin.Recovery(), server.corsMiddleware())
	_ = router.SetTrustedProxies(nil)

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Monierave API is running"})
	})
	router.POST("/users", server.CreateUser)
	router.POST("/users/login", server.loginUser)
	router.GET("/users/verify-email", server.verifyUserEmail)
	router.POST("/tokens/renew_access", server.renewAccessToken)
	router.POST("/sessions/logout", server.logoutCurrentSession)
	router.POST("/webhooks/resend", server.handleResendWebhook)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker, server.store))
	authRoutes.PATCH("/users/me", server.updateUser)
	authRoutes.GET("/users/me/email-status", server.getUserEmailStatus)
	authRoutes.POST("/users/me/resend-verification", server.resendUserEmailVerification)
	authRoutes.POST("/sessions/logout-all", server.logoutAllSessions)

	financialMiddleware := []gin.HandlerFunc{
		authMiddleware(server.tokenMaker, server.store),
	}
	if server.config.EnforceEmailVerification {
		financialMiddleware = append(
			financialMiddleware,
			verifiedAccountMiddleware(server.store),
		)
	}
	financialRoutes := router.Group("/").Use(financialMiddleware...)
	financialRoutes.POST("/accounts", server.createAccount)
	financialRoutes.GET("/accounts/:public_id", server.getAccount)
	financialRoutes.GET("/accounts", server.listAccount)
	financialRoutes.POST("/accounts/:public_id/close", server.closeAccount)
	financialRoutes.POST("/transfers", server.createTransfer)

	server.router = router
}

// Start runs the http server on a particular address
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func (server *Server) Handler() http.Handler {
	return server.router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
