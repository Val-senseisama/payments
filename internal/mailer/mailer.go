package mailer

import (
	"github.com/resend/resend-go/v2"
)

// Mailer is a thin wrapper around the Resend client.
// Instantiate once at startup and inject into handlers that need to send email.
type Mailer struct {
	client *resend.Client
	from   string
}

// New creates a Mailer with the given Resend API key and sender address.
// from should be in the format "Name <address@domain.com>".
func New(apiKey, from string) *Mailer {
	return &Mailer{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendCompanyInvite sends a company invitation email to the given address.
func (m *Mailer) SendCompanyInvite(toEmail, companyName, inviterName string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{toEmail},
		Subject: "You've been invited to join " + companyName,
		Html:    buildInviteHTML(companyName, inviterName),
	}
	_, err := m.client.Emails.Send(params)
	return err
}

// SendLoginOTP sends a one-time passcode to the user's email for passwordless login.
func (m *Mailer) SendLoginOTP(toEmail, otp string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{toEmail},
		Subject: "Your login code",
		Html:    buildLoginOTPHTML(otp),
	}
	_, err := m.client.Emails.Send(params)
	return err
}
