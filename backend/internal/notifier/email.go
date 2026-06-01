package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/jose/ratiodash/internal/domain"
)

const (
	emailTLSModeStartTLS = "starttls"
	emailTLSModeTLS      = "tls"
)

type emailConfig struct {
	host     string
	port     int
	from     string
	to       string // comma/semicolon-separated addresses
	username string
	password string
	tlsMode  string
}

type emailNotifier struct {
	host     string
	port     int
	from     string
	to       []string
	username string
	password string
	tlsMode  string
}

func newEmailNotifier(cfg emailConfig) (domain.Notifier, error) {
	from, err := parseEmailAddress(cfg.from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}
	to, err := parseRecipientList(cfg.to)
	if err != nil {
		return nil, fmt.Errorf("invalid to address list: %w", err)
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.tlsMode))
	if mode == "" {
		mode = emailTLSModeStartTLS
	}
	if mode != emailTLSModeStartTLS && mode != emailTLSModeTLS {
		return nil, fmt.Errorf("invalid tls_mode %q (expected %q or %q)", cfg.tlsMode, emailTLSModeStartTLS, emailTLSModeTLS)
	}
	if (cfg.username == "") != (cfg.password == "") {
		return nil, fmt.Errorf("username and password must be provided together")
	}

	return &emailNotifier{
		host:     cfg.host,
		port:     cfg.port,
		from:     from,
		to:       to,
		username: cfg.username,
		password: cfg.password,
		tlsMode:  mode,
	}, nil
}

func (n *emailNotifier) Notify(ctx context.Context, notif domain.Notification) error {
	c, err := n.connectSMTP(ctx)
	if err != nil {
		return fmt.Errorf("email notify: connect smtp: %w", err)
	}
	defer c.Close()

	if n.username != "" {
		auth := smtp.PlainAuth("", n.username, n.password, n.host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("email notify: auth: %w", err)
		}
	}

	if err := c.Mail(n.from); err != nil {
		return fmt.Errorf("email notify: mail from: %w", err)
	}
	for _, rcpt := range n.to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("email notify: rcpt to %q: %w", rcpt, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("email notify: data: %w", err)
	}
	if _, err := w.Write([]byte(n.buildMessage(notif))); err != nil {
		_ = w.Close()
		return fmt.Errorf("email notify: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email notify: finalize body: %w", err)
	}

	if err := c.Quit(); err != nil {
		return fmt.Errorf("email notify: quit: %w", err)
	}
	return nil
}

func (n *emailNotifier) connectSMTP(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(n.host, strconv.Itoa(n.port))
	if n.tlsMode == emailTLSModeTLS {
		d := tls.Dialer{
			NetDialer: &net.Dialer{},
			Config: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: n.host,
			},
		}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		client, err := smtp.NewClient(conn, n.host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, n.host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	ok, _ := client.Extension("STARTTLS")
	if !ok {
		_ = client.Close()
		return nil, fmt.Errorf("server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: n.host}); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func (n *emailNotifier) buildMessage(notif domain.Notification) string {
	subject := notif.Title
	if subject == "" {
		subject = "RatioDash Notification"
	}

	prefix := levelPrefix(notif.Level)
	if prefix != "" {
		subject = prefix + " " + subject
	}

	body := notif.Body
	if body == "" {
		body = "(empty body)"
	}

	return strings.Join([]string{
		fmt.Sprintf("From: %s", n.from),
		fmt.Sprintf("To: %s", strings.Join(n.to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
		"",
	}, "\r\n")
}

func levelPrefix(level domain.NotificationLevel) string {
	switch level {
	case domain.LevelError:
		return "[ERROR]"
	case domain.LevelWarning:
		return "[WARNING]"
	default:
		return "[INFO]"
	}
}

func parseEmailAddress(raw string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

func parseRecipientList(raw string) ([]string, error) {
	normalized := strings.ReplaceAll(raw, ";", ",")
	parts := strings.Split(normalized, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		addr, err := parseEmailAddress(trimmed)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}
	return out, nil
}
