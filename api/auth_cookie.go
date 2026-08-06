package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (server *Server) setRefreshCookie(
	ctx *gin.Context,
	value string,
	expiresAt time.Time,
) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     server.config.RefreshCookieName,
		Value:    value,
		Path:     "/",
		Domain:   server.config.RefreshCookieDomain,
		MaxAge:   max(0, int(time.Until(expiresAt).Seconds())),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   server.config.RefreshCookieSecure,
		SameSite: refreshCookieSameSite(server.config.RefreshCookieSameSite),
	})
}

func (server *Server) clearRefreshCookie(ctx *gin.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     server.config.RefreshCookieName,
		Path:     "/",
		Domain:   server.config.RefreshCookieDomain,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   server.config.RefreshCookieSecure,
		SameSite: refreshCookieSameSite(server.config.RefreshCookieSameSite),
	})
}

func (server *Server) refreshCookie(ctx *gin.Context) (string, error) {
	cookie, err := ctx.Request.Cookie(server.config.RefreshCookieName)
	if err != nil || cookie.Value == "" {
		return "", ErrInvalidToken
	}
	return cookie.Value, nil
}

func (server *Server) originAllowed(ctx *gin.Context) bool {
	origin := ctx.GetHeader("Origin")
	if origin == "" {
		return true
	}

	for _, allowed := range strings.Split(server.config.AllowedOrigins, ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}

	parsed, err := url.Parse(origin)
	return err == nil && strings.EqualFold(parsed.Host, ctx.Request.Host)
}

func refreshCookieSameSite(value string) http.SameSite {
	switch value {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (server *Server) corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" {
			if !server.originAllowed(ctx) {
				ctx.AbortWithStatusJSON(
					http.StatusForbidden,
					errorResponse(ErrForbidden),
				)
				return
			}
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header(
				"Access-Control-Allow-Headers",
				"Authorization, Content-Type, Idempotency-Key",
			)
			ctx.Header("Access-Control-Expose-Headers", "Idempotency-Replayed")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			ctx.Header("Vary", "Origin")
		}
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
