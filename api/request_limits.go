package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxJSONBodySize = 64 << 10

var ErrRequestBodyTooLarge = errors.New("request body exceeds 64 KiB")

func (server *Server) requestBodyLimitMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Body != nil &&
			ctx.Request.URL.Path != "/webhooks/resend" {
			ctx.Request.Body = http.MaxBytesReader(
				ctx.Writer,
				ctx.Request.Body,
				maxJSONBodySize,
			)
		}
		ctx.Next()
	}
}

func bindJSON(ctx *gin.Context, destination any) error {
	return ctx.ShouldBindJSON(destination)
}

func writeBindingError(ctx *gin.Context, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		ctx.JSON(
			http.StatusRequestEntityTooLarge,
			codedErrorResponse(ctx, "request_body_too_large", ErrRequestBodyTooLarge),
		)
		return
	}
	ctx.JSON(http.StatusBadRequest, errorResponse(ctx, friendlyValidationError(err)))
}
