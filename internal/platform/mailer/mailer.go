package mailer

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// InviteMessage is the tenant-scoped invite email payload.
type InviteMessage struct {
	ToEmail       string
	TenantName    string
	InviterName   string
	RoleLabel     string
	ActivationURL string
	ExpiresAt     time.Time
}

// Sender delivers invite emails.
type Sender interface {
	SendInvite(ctx context.Context, msg InviteMessage) error
}

// SMTPConfig configures SMTP-based email delivery.
type SMTPConfig struct {
	Addr        string
	Username    string
	Password    string
	FromAddress string
	FromName    string
}

type sendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// SMTPSender delivers invite emails over SMTP.
type SMTPSender struct {
	addr        string
	host        string
	auth        smtp.Auth
	fromAddress string
	fromName    string
	sendMail    sendMailFunc
}

// NewSMTPSender creates an SMTP-backed invite sender.
func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	addr := strings.TrimSpace(cfg.Addr)
	fromAddress := strings.TrimSpace(cfg.FromAddress)
	if addr == "" {
		return nil, fmt.Errorf("smtp addr is required")
	}
	if fromAddress == "" {
		return nil, fmt.Errorf("smtp from address is required")
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("parse smtp addr: %w", err)
	}

	var authMethod smtp.Auth
	if strings.TrimSpace(cfg.Username) != "" {
		authMethod = smtp.PlainAuth("", strings.TrimSpace(cfg.Username), cfg.Password, host)
	}

	return &SMTPSender{
		addr:        addr,
		host:        host,
		auth:        authMethod,
		fromAddress: fromAddress,
		fromName:    strings.TrimSpace(cfg.FromName),
		sendMail:    smtp.SendMail,
	}, nil
}

// SendInvite sends an invite email with the activation link.
func (s *SMTPSender) SendInvite(_ context.Context, msg InviteMessage) error {
	if s == nil {
		return fmt.Errorf("smtp sender is nil")
	}
	toEmail := strings.TrimSpace(msg.ToEmail)
	if toEmail == "" {
		return fmt.Errorf("invite recipient email is required")
	}
	if strings.TrimSpace(msg.ActivationURL) == "" {
		return fmt.Errorf("invite activation url is required")
	}

	body := buildInviteEmail(s.fromAddress, s.fromName, msg)
	if err := s.sendMail(s.addr, s.auth, s.fromAddress, []string{toEmail}, []byte(body)); err != nil {
		return fmt.Errorf("send invite email: %w", err)
	}
	return nil
}

func buildInviteEmail(fromAddress, fromName string, msg InviteMessage) string {
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}

	subjectScope := strings.TrimSpace(msg.TenantName)
	if subjectScope == "" {
		subjectScope = "P&AI Bot"
	}

	roleLabel := strings.TrimSpace(msg.RoleLabel)
	if roleLabel == "" {
		roleLabel = "workspace"
	}

	inviterName := strings.TrimSpace(msg.InviterName)
	if inviterName == "" {
		inviterName = "An administrator"
	}

	lines := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.TrimSpace(msg.ToEmail)),
		fmt.Sprintf("Subject: Your %s invite", subjectScope),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("%s invited you to join %s as a %s.", inviterName, subjectScope, roleLabel),
		"",
		"Open the activation link below to set your password and access the workspace:",
		msg.ActivationURL,
		"",
		fmt.Sprintf("This invite expires on %s.", msg.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC")),
		"",
		"If you were not expecting this invitation, you can ignore this email.",
	}

	return strings.Join(lines, "\r\n")
}
