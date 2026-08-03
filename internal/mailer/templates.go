package mailer

import "fmt"

func buildInviteHTML(companyName, inviterName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>You've been invited</title>
  <style>
    body { margin: 0; padding: 0; background: #f4f4f7; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    .wrapper { max-width: 560px; margin: 40px auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
    .header { background: #1a1a2e; padding: 32px 40px; text-align: center; }
    .header h1 { margin: 0; color: #ffffff; font-size: 22px; font-weight: 600; letter-spacing: -0.3px; }
    .body { padding: 40px; }
    .body p { margin: 0 0 16px; color: #374151; font-size: 15px; line-height: 1.6; }
    .cta { display: block; margin: 32px auto 0; padding: 14px 32px; background: #4f46e5; color: #ffffff; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 15px; text-align: center; }
    .footer { padding: 24px 40px; text-align: center; border-top: 1px solid #e5e7eb; }
    .footer p { margin: 0; color: #9ca3af; font-size: 13px; }
  </style>
</head>
<body>
  <div class="wrapper">
    <div class="header">
      <h1>Payments Platform</h1>
    </div>
    <div class="body">
      <p>Hi there,</p>
      <p><strong>%s</strong> has invited you to join <strong>%s</strong> on the Payments Platform.</p>
      <p>Click the button below to accept your invitation and get started.</p>
      <a href="#" class="cta">Accept Invitation</a>
    </div>
    <div class="footer">
      <p>If you weren't expecting this invitation, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`, inviterName, companyName)
}

func buildLoginOTPHTML(otp string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Your login code</title>
  <style>
    body { margin: 0; padding: 0; background: #f4f4f7; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    .wrapper { max-width: 480px; margin: 40px auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
    .header { background: #1a1a2e; padding: 32px 40px; text-align: center; }
    .header h1 { margin: 0; color: #ffffff; font-size: 22px; font-weight: 600; letter-spacing: -0.3px; }
    .body { padding: 40px; text-align: center; }
    .body p { margin: 0 0 16px; color: #374151; font-size: 15px; line-height: 1.6; text-align: left; }
    .otp { display: inline-block; margin: 24px 0; padding: 16px 40px; background: #f3f4f6; border-radius: 10px; font-size: 36px; font-weight: 700; letter-spacing: 12px; color: #111827; font-variant-numeric: tabular-nums; }
    .expiry { color: #6b7280; font-size: 13px; margin-top: 8px; }
    .footer { padding: 24px 40px; text-align: center; border-top: 1px solid #e5e7eb; }
    .footer p { margin: 0; color: #9ca3af; font-size: 13px; }
  </style>
</head>
<body>
  <div class="wrapper">
    <div class="header">
      <h1>Payments Platform</h1>
    </div>
    <div class="body">
      <p>Here is your one-time login code. It expires in <strong>10 minutes</strong>.</p>
      <div class="otp">%s</div>
      <p class="expiry">Do not share this code with anyone.</p>
    </div>
    <div class="footer">
      <p>If you didn't request this code, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`, otp)
}
