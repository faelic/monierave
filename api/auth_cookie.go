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
	server.setSessionCookie(ctx, server.config.RefreshCookieName, value, expiresAt)
}

func (server *Server) setDeviceCookie(
	ctx *gin.Context,
	value string,
	expiresAt time.Time,
) {
	server.setSessionCookie(ctx, server.config.DeviceCookieName, value, expiresAt)
}

func (server *Server) setSessionCookie(
	ctx *gin.Context,
	name string,
	value string,
	expiresAt time.Time,
) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
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
	server.clearSessionCookie(ctx, server.config.RefreshCookieName)
}

func (server *Server) clearDeviceCookie(ctx *gin.Context) {
	server.clearSessionCookie(ctx, server.config.DeviceCookieName)
}

func (server *Server) clearSessionCookies(ctx *gin.Context) {
	server.clearRefreshCookie(ctx)
	server.clearDeviceCookie(ctx)
}

func (server *Server) clearSessionCookie(ctx *gin.Context, name string) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
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
	return sessionCookie(ctx, server.config.RefreshCookieName)
}

func (server *Server) deviceCookie(ctx *gin.Context) (string, error) {
	return sessionCookie(ctx, server.config.DeviceCookieName)
}

func sessionCookie(ctx *gin.Context, name string) (string, error) {
	cookie, err := ctx.Request.Cookie(name)
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
	if err != nil || parsed.Host == "" || ctx.Request.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Host, ctx.Request.Host)
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
			ctx.Header("Vary", "Origin")
		}
		if origin != "" && server.originAllowed(ctx) {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header(
				"Access-Control-Allow-Headers",
				"Authorization, Content-Type, Idempotency-Key",
			)
			ctx.Header(
				"Access-Control-Expose-Headers",
				"Idempotency-Replayed, X-Request-ID, X-Correlation-ID",
			)
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if ctx.Request.Method == http.MethodOptions {
			if origin == "" || !server.originAllowed(ctx) {
				ctx.AbortWithStatusJSON(
					http.StatusForbidden,
					errorResponse(ctx, ErrForbidden),
				)
				return
			}
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

// requireTrustedBrowserOrigin rejects cross-origin browser mutations while
// preserving requests from non-browser API clients, which do not send Origin.
func (server *Server) requireTrustedBrowserOrigin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !server.originAllowed(ctx) {
			ctx.AbortWithStatusJSON(
				http.StatusForbidden,
				errorResponse(ctx, ErrForbidden),
			)
			return
		}
		ctx.Next()
	}
}
