package api

import (
	"html/template"

	"github.com/gin-gonic/gin"
)

type verificationPageData struct {
	Title       string
	Message     string
	Token       string
	ShowConfirm bool
}

var verificationPageTemplate = template.Must(template.New("verification").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}} | Monierave</title>
</head>
<body style="margin:0;background:#f4f1e8;color:#17211b;font-family:Georgia,serif;">
  <main style="max-width:560px;margin:10vh auto;padding:24px;">
    <section style="background:#fff;border:1px solid #d9d3c5;border-radius:16px;padding:36px;">
      <p style="margin:0 0 20px;color:#276749;font:700 13px Arial,sans-serif;letter-spacing:.12em;text-transform:uppercase;">Monierave</p>
      <h1 style="margin:0 0 16px;font-size:30px;">{{.Title}}</h1>
      <p style="margin:0 0 28px;font:16px/1.7 Arial,sans-serif;color:#4d574f;">{{.Message}}</p>
      {{if .ShowConfirm}}
      <form method="post" action="/users/verify-email">
        <input type="hidden" name="token" value="{{.Token}}">
        <button type="submit" style="border:0;border-radius:8px;background:#276749;color:#fff;font:700 15px Arial,sans-serif;padding:15px 24px;cursor:pointer;">Confirm email address</button>
      </form>
      {{end}}
    </section>
  </main>
</body>
</html>`))

func setVerificationPageHeaders(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("X-Frame-Options", "DENY")
	ctx.Header(
		"Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'",
	)
}

func renderVerificationPage(
	ctx *gin.Context,
	status int,
	title string,
	message string,
	token string,
	showConfirm bool,
) {
	setVerificationPageHeaders(ctx)
	ctx.Status(status)
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	if err := verificationPageTemplate.Execute(ctx.Writer, verificationPageData{
		Title:       title,
		Message:     message,
		Token:       token,
		ShowConfirm: showConfirm,
	}); err != nil {
		_ = ctx.Error(err)
	}
}

func wantsJSON(ctx *gin.Context) bool {
	return ctx.ContentType() == "application/json" ||
		ctx.GetHeader("Accept") == "application/json"
}

func verificationErrorPage(ctx *gin.Context, status int, err error) {
	renderVerificationPage(ctx, status, "Verification unavailable", err.Error(), "", false)
}
