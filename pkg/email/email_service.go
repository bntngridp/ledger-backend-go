package email

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type EmailService interface {
	SendOTPEmail(toEmail, otpCode, purpose string) error
	SendOTPEmailAsync(toEmail, otpCode, purpose string)
}

type emailService struct {
	host       string
	port       int
	username   string
	password   string
	senderName string
	logger     *slog.Logger
}

func NewEmailService(host string, port int, username, password, senderName string, logger *slog.Logger) EmailService {
	return &emailService{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		senderName: senderName,
		logger:     logger,
	}
}

// GenerateNumericOTP generates a random 6-digit OTP code
func GenerateNumericOTP() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456" // fallback
	}
	return fmt.Sprintf("%06d", nBig.Int64())
}

func (s *emailService) SendOTPEmail(toEmail, otpCode, purpose string) error {
	if s.username == "" || s.password == "" || s.host == "" {
		s.logger.Warn("SMTP credentials missing, logging OTP instead of sending email",
			"to", toEmail, "code", otpCode, "purpose", purpose)
		return nil
	}

	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	subject := fmt.Sprintf("[%s] Kode Verifikasi OTP Anda", s.senderName)
	if purpose != "" {
		subject = fmt.Sprintf("[%s] Kode OTP Verifikasi %s", s.senderName, purpose)
	}

	htmlBody := s.buildHTMLBody(toEmail, otpCode, purpose)

	header := make(map[string]string)
	header["From"] = fmt.Sprintf("%s <%s>", s.senderName, s.username)
	header["To"] = toEmail
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"UTF-8\""

	var message strings.Builder
	for k, v := range header {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n" + htmlBody)

	err := smtp.SendMail(addr, auth, s.username, []string{toEmail}, []byte(message.String()))
	if err != nil {
		s.logger.Error("Failed to send OTP email via SMTP", "to", toEmail, "error", err)
		return err
	}

	s.logger.Info("Successfully sent OTP email", "to", toEmail, "purpose", purpose)
	return nil
}

func (s *emailService) SendOTPEmailAsync(toEmail, otpCode, purpose string) {
	go func() {
		_ = s.SendOTPEmail(toEmail, otpCode, purpose)
	}()
}

func (s *emailService) buildHTMLBody(toEmail, otpCode, purpose string) string {
	now := time.Now().Format("02 Jan 2006, 15:04 MST")
	if purpose == "" {
		purpose = "Autentikasi Akun"
	}

	// Split 6-digit OTP code into individual HTML digit boxes
	var digitBoxes strings.Builder
	for _, ch := range otpCode {
		digitBoxes.WriteString(fmt.Sprintf(`<span style="display: inline-block; width: 38px; height: 46px; line-height: 46px; background-color: #1e293b; border: 1.5px solid #6366f1; border-radius: 10px; font-family: 'SF Pro Display', -apple-system, 'Segoe UI', Roboto, sans-serif; font-size: 24px; font-weight: 800; color: #ffffff; text-align: center; margin: 0 4px; box-shadow: 0 2px 8px rgba(99, 102, 241, 0.25);">%c</span>`, ch))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Kode OTP Ledger</title>
</head>
<body style="margin: 0; padding: 0; background-color: #0b0f19; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; -webkit-font-smoothing: antialiased;">
  <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #0b0f19; padding: 40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width: 480px; background-color: #0f172a; border-radius: 20px; border: 1px solid #1e293b; overflow: hidden; box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);">
          
          <!-- Header Banner -->
          <tr>
            <td style="background: linear-gradient(135deg, #6366f1 0%%, #4f46e5 100%%); padding: 28px 32px; text-align: center;">
              <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0">
                <tr>
                  <td align="center">
                    <span style="font-size: 26px; font-weight: 900; color: #ffffff; letter-spacing: -0.5px; text-transform: uppercase;">⚡ LEDGER</span>
                  </td>
                </tr>
                <tr>
                  <td align="center" style="padding-top: 4px;">
                    <span style="font-size: 11px; font-weight: 700; color: #c7d2fe; letter-spacing: 2px; text-transform: uppercase;">Hybrid Multi-Asset Wallet</span>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Content Body -->
          <tr>
            <td style="padding: 36px 32px; text-align: center;">
              <h1 style="margin: 0 0 10px 0; font-size: 22px; font-weight: 800; color: #f8fafc; letter-spacing: -0.3px;">Kode Verifikasi OTP</h1>
              <p style="margin: 0 0 28px 0; font-size: 14px; color: #94a3b8; line-height: 1.6;">
                Gunakan 6-digit kode di bawah ini untuk mengonfirmasi permintaan <strong>%s</strong> di aplikasi Ledger Anda.
              </p>

              <!-- 6-Digit OTP Box Grid -->
              <div style="margin: 24px 0 28px 0; text-align: center;">
                %s
              </div>

              <!-- Expiry & Security Notice Card -->
              <table role="presentation" width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #1e293b; border-radius: 12px; border-left: 4px solid #6366f1; padding: 16px; text-align: left; margin-bottom: 24px;">
                <tr>
                  <td>
                    <p style="margin: 0; font-size: 13px; font-weight: 700; color: #f8fafc; line-height: 1.4;">
                      🔒 Keamanan Akun Anda
                    </p>
                    <p style="margin: 4px 0 0 0; font-size: 12px; color: #94a3b8; line-height: 1.5;">
                      • Kode berlaku selama <strong>5 menit</strong>.<br>
                      • Rahasiakan kode ini! Tim Ledger tidak pernah meminta OTP Anda.
                    </p>
                  </td>
                </tr>
              </table>

              <p style="margin: 0; font-size: 12px; color: #64748b; line-height: 1.5;">
                Jika Anda tidak merasa melakukan permintaan ini, abaikan pesan ini atau segera amankan akun Anda.
              </p>
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="background-color: #0b0f19; padding: 20px 32px; border-top: 1px solid #1e293b; text-align: center;">
              <p style="margin: 0; font-size: 11px; color: #475569; line-height: 1.5;">
                © 2026 Ledger Inc. Hak Cipta Dilindungi.<br>
                Pesan otomatis dikirim pada %s
              </p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, purpose, digitBoxes.String(), now)
}

func ParseSMTPPort(portStr string) int {
	p, err := strconv.Atoi(portStr)
	if err != nil || p <= 0 {
		return 587
	}
	return p
}
