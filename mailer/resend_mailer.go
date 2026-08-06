package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/mail"
	"net/url"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/resend/resend-go/v3"
)

const (
	resendEmailSubject        = "Welcome to Monierave"
	resendVerificationSubject = "Verify your Monierave email"
	resendFinancialSubject    = "Monierave account activity"
)

type resendEmailSender interface {
	SendWithOptions(
		ctx context.Context,
		params *resend.SendEmailRequest,
		options *resend.SendEmailOptions,
	) (*resend.SendEmailResponse, error)
}

type ResendMailer struct {
	sender resendEmailSender
	from   string
}

type resendPayload struct {
	VerificationURL string `json:"verification_url"`
}

type resendTemplateData struct {
	Username        string
	VerificationURL string
}

type financialPayload struct {
	EventType     string    `json:"event_type"`
	Reference     string    `json:"reference"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Direction     string    `json:"direction"`
	OccurredAt    time.Time `json:"occurred_at"`
	AccountStatus string    `json:"account_status"`
	Reason        string    `json:"reason"`
}

type financialTemplateData struct {
	Username      string
	Title         string
	Summary       string
	Reference     string
	Amount        int64
	Currency      string
	Direction     string
	OccurredAt    string
	AccountStatus string
	Reason        string
}

var resendHTMLTemplate = template.Must(template.New("verification-email").Parse(`<!doctype html>
<html lang="en">
<body style="margin:0;background:#f4f1e8;color:#17211b;font-family:Georgia,serif;">
  <div style="max-width:600px;margin:0 auto;padding:40px 24px;">
    <div style="background:#fff;border:1px solid #d9d3c5;border-radius:16px;padding:36px;">
      <p style="margin:0 0 24px;color:#276749;font:700 13px/1.4 Arial,sans-serif;letter-spacing:0.12em;text-transform:uppercase;">Monierave</p>
      <h1 style="margin:0 0 16px;font-size:30px;line-height:1.2;">Welcome, {{.Username}}</h1>
      <p style="margin:0 0 24px;font-size:17px;line-height:1.7;">We received a request to create a Monierave account using this email address.</p>
      {{if .VerificationURL}}
      <table role="presentation" border="0" cellpadding="0" cellspacing="0" style="margin:0 0 28px;">
        <tr>
          <td bgcolor="#276749" style="border-radius:8px;text-align:center;">
            <a href="{{.VerificationURL}}" target="_blank" rel="noopener noreferrer" style="display:inline-block;border:1px solid #276749;border-radius:8px;background:#276749;color:#ffffff;font:700 15px/1 Arial,sans-serif;padding:15px 24px;text-decoration:none;">Verify email address</a>
          </td>
        </tr>
      </table>
      <p style="margin:0 0 8px;color:#5d655f;font:14px/1.6 Arial,sans-serif;">If the button does not work, copy and paste this link into your browser:</p>
      <p style="margin:0 0 24px;font:14px/1.6 Arial,sans-serif;word-break:break-all;overflow-wrap:anywhere;">
        <a href="{{.VerificationURL}}" target="_blank" rel="noopener noreferrer" style="color:#276749;text-decoration:underline;word-break:break-all;">{{.VerificationURL}}</a>
      </p>
      <p style="margin:0 0 24px;color:#5d655f;font:14px/1.6 Arial,sans-serif;">This verification link expires in 24 hours.</p>
      {{end}}
      <p style="margin:0;color:#5d655f;font-size:14px;line-height:1.6;">If you did not create this account, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`))

var resendTextTemplate = texttemplate.Must(texttemplate.New("verification-email").Parse(`Welcome, {{.Username}}

We received a request to create a Monierave account using this email address.
{{if .VerificationURL}}
Verify your email address: {{.VerificationURL}}

If the link is not clickable, copy and paste it into your browser.
This verification link expires in 24 hours.
{{end}}
If you did not create this account, you can safely ignore this email.
`))

var financialHTMLTemplate = template.Must(template.New("financial-email").Parse(`<!doctype html>
<html lang="en">
<body style="margin:0;background:#f4f1e8;color:#17211b;font-family:Georgia,serif;">
  <div style="max-width:600px;margin:0 auto;padding:40px 24px;">
    <div style="background:#fff;border:1px solid #d9d3c5;border-radius:16px;padding:36px;">
      <p style="margin:0 0 24px;color:#276749;font:700 13px/1.4 Arial,sans-serif;letter-spacing:0.12em;text-transform:uppercase;">Monierave</p>
      <h1 style="margin:0 0 16px;font-size:28px;line-height:1.2;">{{.Title}}</h1>
      <p style="margin:0 0 24px;font-size:16px;line-height:1.7;">Hello {{.Username}}, {{.Summary}}</p>
      <table role="presentation" style="width:100%;border-collapse:collapse;font:14px/1.6 Arial,sans-serif;">
        {{if .Reference}}<tr><td style="padding:8px 0;color:#5d655f;">Reference</td><td style="padding:8px 0;text-align:right;font-weight:700;">{{.Reference}}</td></tr>{{end}}
        {{if .Currency}}<tr><td style="padding:8px 0;color:#5d655f;">Amount</td><td style="padding:8px 0;text-align:right;font-weight:700;">{{.Amount}} {{.Currency}} (minor units)</td></tr>{{end}}
        {{if .Direction}}<tr><td style="padding:8px 0;color:#5d655f;">Direction</td><td style="padding:8px 0;text-align:right;">{{.Direction}}</td></tr>{{end}}
        {{if .AccountStatus}}<tr><td style="padding:8px 0;color:#5d655f;">Account status</td><td style="padding:8px 0;text-align:right;">{{.AccountStatus}}</td></tr>{{end}}
        <tr><td style="padding:8px 0;color:#5d655f;">Time</td><td style="padding:8px 0;text-align:right;">{{.OccurredAt}}</td></tr>
      </table>
      {{if .Reason}}<p style="margin:24px 0 0;color:#5d655f;font:14px/1.6 Arial,sans-serif;">Reason: {{.Reason}}</p>{{end}}
      <p style="margin:24px 0 0;color:#5d655f;font:14px/1.6 Arial,sans-serif;">If you do not recognize this activity, contact support through the official Monierave application.</p>
    </div>
  </div>
</body>
</html>`))

var financialTextTemplate = texttemplate.Must(texttemplate.New("financial-email").Parse(`{{.Title}}

Hello {{.Username}}, {{.Summary}}
{{if .Reference}}Reference: {{.Reference}}
{{end}}{{if .Currency}}Amount: {{.Amount}} {{.Currency}} (minor units)
{{end}}{{if .Direction}}Direction: {{.Direction}}
{{end}}{{if .AccountStatus}}Account status: {{.AccountStatus}}
{{end}}Time: {{.OccurredAt}}
{{if .Reason}}Reason: {{.Reason}}
{{end}}
If you do not recognize this activity, contact support through the official Monierave application.
`))

func NewResendMailer(apiKey string, from string) (*ResendMailer, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("Resend API key is required")
	}

	from = strings.TrimSpace(from)
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("invalid EMAIL_FROM: %w", err)
	}

	return &ResendMailer{
		sender: resend.NewClient(apiKey).Emails,
		from:   from,
	}, nil
}

func (mailer *ResendMailer) SendVerificationEmail(
	ctx context.Context,
	message VerificationEmail,
) (string, error) {
	if _, err := mail.ParseAddress(message.Recipient); err != nil {
		return "", NewPermanentError(fmt.Errorf("invalid recipient: %w", err))
	}

	var payload resendPayload
	if len(message.Payload) > 0 {
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return "", NewPermanentError(fmt.Errorf("invalid verification email payload: %w", err))
		}
	}
	if payload.VerificationURL != "" {
		verificationURL, err := url.Parse(payload.VerificationURL)
		if err != nil ||
			verificationURL.Host == "" ||
			(verificationURL.Scheme != "http" && verificationURL.Scheme != "https") {
			return "", NewPermanentError(errors.New(
				"verification_url must be an absolute HTTP or HTTPS URL",
			))
		}
	}

	htmlBody, textBody, err := renderResendEmail(resendTemplateData{
		Username:        message.Username,
		VerificationURL: payload.VerificationURL,
	})
	if err != nil {
		return "", NewPermanentError(fmt.Errorf("render verification email: %w", err))
	}

	subject := resendEmailSubject
	if payload.VerificationURL != "" {
		subject = resendVerificationSubject
	}

	response, err := mailer.sender.SendWithOptions(
		ctx,
		&resend.SendEmailRequest{
			From:    mailer.from,
			To:      []string{message.Recipient},
			Subject: subject,
			Html:    htmlBody,
			Text:    textBody,
			Tags: []resend.Tag{
				{Name: "category", Value: "verify_email"},
				{Name: "job_id", Value: message.JobID},
			},
		},
		&resend.SendEmailOptions{
			IdempotencyKey: "verify-email/" + message.JobID,
		},
	)
	if err != nil {
		var invalidRequest *resend.MissingRequiredFieldsError
		if errors.As(err, &invalidRequest) {
			return "", NewPermanentError(fmt.Errorf("Resend rejected email request: %w", err))
		}
		return "", fmt.Errorf("send email with Resend: %w", err)
	}
	if response == nil || response.Id == "" {
		return "", errors.New("Resend accepted email without returning a message ID")
	}

	return response.Id, nil
}

func (mailer *ResendMailer) SendFinancialNotification(
	ctx context.Context,
	message FinancialNotificationEmail,
) (string, error) {
	if _, err := mail.ParseAddress(message.Recipient); err != nil {
		return "", NewPermanentError(fmt.Errorf("invalid recipient: %w", err))
	}
	var payload financialPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return "", NewPermanentError(fmt.Errorf(
			"invalid financial notification payload: %w",
			err,
		))
	}
	title, summary, err := financialEventCopy(payload.EventType)
	if err != nil {
		return "", NewPermanentError(err)
	}
	if payload.OccurredAt.IsZero() {
		return "", NewPermanentError(errors.New(
			"financial notification occurred_at is required",
		))
	}
	if strings.HasPrefix(payload.EventType, "transaction.") &&
		payload.Reference == "" {
		return "", NewPermanentError(errors.New(
			"financial transaction reference is required",
		))
	}
	htmlBody, textBody, err := renderFinancialEmail(financialTemplateData{
		Username:      message.Username,
		Title:         title,
		Summary:       summary,
		Reference:     payload.Reference,
		Amount:        payload.Amount,
		Currency:      payload.Currency,
		Direction:     payload.Direction,
		OccurredAt:    payload.OccurredAt.UTC().Format(time.RFC3339),
		AccountStatus: payload.AccountStatus,
		Reason:        payload.Reason,
	})
	if err != nil {
		return "", NewPermanentError(fmt.Errorf(
			"render financial notification: %w",
			err,
		))
	}
	response, err := mailer.sender.SendWithOptions(
		ctx,
		&resend.SendEmailRequest{
			From:    mailer.from,
			To:      []string{message.Recipient},
			Subject: resendFinancialSubject + ": " + title,
			Html:    htmlBody,
			Text:    textBody,
			Tags: []resend.Tag{
				{
					Name:  "category",
					Value: strings.ReplaceAll(payload.EventType, ".", "_"),
				},
				{Name: "job_id", Value: message.JobID},
			},
		},
		&resend.SendEmailOptions{
			IdempotencyKey: "financial-notification/" + message.JobID,
		},
	)
	if err != nil {
		var invalidRequest *resend.MissingRequiredFieldsError
		if errors.As(err, &invalidRequest) {
			return "", NewPermanentError(fmt.Errorf(
				"Resend rejected email request: %w",
				err,
			))
		}
		return "", fmt.Errorf("send financial notification with Resend: %w", err)
	}
	if response == nil || response.Id == "" {
		return "", errors.New("Resend accepted email without returning a message ID")
	}
	return response.Id, nil
}

func financialEventCopy(eventType string) (string, string, error) {
	switch eventType {
	case "transaction.posted":
		return "Transaction posted", "a transaction was posted on your account.", nil
	case "transaction.failed":
		return "Transaction failed", "a transaction could not be completed.", nil
	case "transaction.reversed":
		return "Transaction reversed", "a transaction was reversed.", nil
	case "account.frozen":
		return "Account frozen", "your account has been frozen.", nil
	case "account.unfrozen":
		return "Account unfrozen", "your account has been unfrozen.", nil
	case "account.closed":
		return "Account closed", "your account has been closed.", nil
	default:
		return "", "", fmt.Errorf("unsupported financial event type %q", eventType)
	}
}

func renderResendEmail(data resendTemplateData) (string, string, error) {
	var htmlBody bytes.Buffer
	if err := resendHTMLTemplate.Execute(&htmlBody, data); err != nil {
		return "", "", err
	}

	var textBody bytes.Buffer
	if err := resendTextTemplate.Execute(&textBody, data); err != nil {
		return "", "", err
	}

	return htmlBody.String(), textBody.String(), nil
}

func renderFinancialEmail(
	data financialTemplateData,
) (string, string, error) {
	var htmlBody bytes.Buffer
	if err := financialHTMLTemplate.Execute(&htmlBody, data); err != nil {
		return "", "", err
	}
	var textBody bytes.Buffer
	if err := financialTextTemplate.Execute(&textBody, data); err != nil {
		return "", "", err
	}
	return htmlBody.String(), textBody.String(), nil
}
