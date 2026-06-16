package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

// Attachment represents a file attachment for delivery.
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

// ReportDelivery handles report delivery via email, IM, etc.
type ReportDelivery struct {
	db  *gorm.DB
	cfg *config.Config
}

type ReportDeliveryService = ReportDelivery

// NewReportDelivery creates a new ReportDelivery.
func NewReportDelivery(db *gorm.DB, cfg *config.Config) *ReportDelivery {
	return &ReportDelivery{db: db, cfg: cfg}
}

func NewReportDeliveryService(cfg *config.Config, store storage.Driver) *ReportDeliveryService {
	return NewReportDelivery(nil, cfg)
}

// SendEmail sends an email with optional attachment via SMTP.
func (d *ReportDelivery) SendEmail(ctx context.Context, to []string, subject string, body string, attachment *Attachment) error {
	smtpCfg := d.cfg.SMTP
	if !smtpCfg.Enabled {
		return fmt.Errorf("SMTP is not enabled")
	}
	if smtpCfg.Host == "" {
		return fmt.Errorf("SMTP host is not configured")
	}
	if len(to) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build email message
	msg, err := d.buildMessage(smtpCfg.FromName, smtpCfg.FromEmail, to, subject, body, attachment)
	if err != nil {
		return fmt.Errorf("failed to build email message: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)

	switch strings.ToLower(smtpCfg.Encryption) {
	case "ssl":
		err = d.sendSSL(addr, auth, smtpCfg.FromEmail, to, msg)
	case "tls":
		err = d.sendSTARTTLS(addr, auth, smtpCfg.FromEmail, to, msg)
	default:
		err = smtp.SendMail(addr, auth, smtpCfg.FromEmail, to, msg)
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendToIM sends a message to an IM platform (placeholder).
func (d *ReportDelivery) SendToIM(ctx context.Context, platform string, recipient string, message string, fileURL string) error {
	log.Printf("[ReportDelivery] IM delivery placeholder: platform=%s, recipient=%s, message_len=%d, fileURL=%s",
		platform, recipient, len(message), fileURL)
	return nil
}

// buildMessage constructs a MIME email message.
func (d *ReportDelivery) buildMessage(fromName, fromEmail string, to []string, subject, body string, attachment *Attachment) ([]byte, error) {
	var sb strings.Builder

	// Headers
	sb.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString("MIME-Version: 1.0\r\n")

	boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())

	if attachment != nil {
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		sb.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(body)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
		sb.WriteString("Content-Transfer-Encoding: base64\r\n")
		sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
		sb.WriteString("\r\n")

		// Encode attachment content as base64 with line wrapping
		encoded := d.base64Encode(attachment.Content)
		sb.WriteString(encoded)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		sb.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(body)
	}

	return []byte(sb.String()), nil
}

// sendSTARTTLS sends email with STARTTLS.
func (d *ReportDelivery) sendSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: d.cfg.SMTP.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, d.cfg.SMTP.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT failed for %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	return w.Close()
}

// sendSSL sends email over direct SSL connection.
func (d *ReportDelivery) sendSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: d.cfg.SMTP.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("SSL dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, d.cfg.SMTP.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT failed for %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	return w.Close()
}

// base64Encode encodes data to base64 with line wrapping every 76 characters.
func (d *ReportDelivery) base64Encode(data []byte) string {
	const lineLen = 76
	var encoded strings.Builder

	raw := make([]byte, (len(data)+2)/3*4)
	rawLen := 0
	for i := 0; i < len(data); i += 3 {
		var val uint32
		remaining := len(data) - i
		if remaining >= 3 {
			val = (uint32(data[i]) << 16) | (uint32(data[i+1]) << 8) | uint32(data[i+2])
		} else if remaining == 2 {
			val = (uint32(data[i]) << 16) | (uint32(data[i+1]) << 8)
		} else {
			val = (uint32(data[i]) << 16)
		}

		const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		raw[rawLen] = alphabet[(val>>18)&0x3F]
		raw[rawLen+1] = alphabet[(val>>12)&0x3F]
		if remaining >= 2 {
			raw[rawLen+2] = alphabet[(val>>6)&0x3F]
		} else {
			raw[rawLen+2] = '='
		}
		if remaining >= 1 {
			raw[rawLen+3] = alphabet[val&0x3F]
		} else {
			raw[rawLen+3] = '='
		}
		rawLen += 4
	}

	// Insert line breaks
	lineChars := 0
	for i := 0; i < rawLen; i++ {
		encoded.WriteByte(raw[i])
		lineChars++
		if lineChars >= lineLen {
			encoded.WriteString("\r\n")
			lineChars = 0
		}
	}
	if lineChars > 0 {
		encoded.WriteString("\r\n")
	}

	return encoded.String()
}
