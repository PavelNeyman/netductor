package notify

import (
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SMTP settings from env (also loadable via netductor.conf → env):
//   NETDUCTOR_SMTP_HOST=mail.example.com:587
//   NETDUCTOR_SMTP_USER=...
//   NETDUCTOR_SMTP_PASS=...
//   NETDUCTOR_SMTP_FROM=netductor@example.com
//   NETDUCTOR_SMTP_TO=you@example.com   (comma-separated)
func smtpConfigured() bool {
	return strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_HOST")) != "" &&
		strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_TO")) != ""
}

// Email sends a plain-text alert. No extra daemon — direct SMTP from netductor.
func Email(subject, body string) error {
	hostPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_HOST"))
	toList := strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_TO"))
	from := strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_FROM"))
	user := strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_USER"))
	pass := strings.TrimSpace(os.Getenv("NETDUCTOR_SMTP_PASS"))
	if hostPort == "" || toList == "" {
		return fmt.Errorf("smtp not configured")
	}
	if from == "" {
		from = user
	}
	if from == "" {
		from = "netductor@localhost"
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
		hostPort = net.JoinHostPort(host, "587")
	}
	recipients := []string{}
	for _, a := range strings.Split(toList, ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			recipients = append(recipients, a)
		}
	}
	msg := strings.Builder{}
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n\r\n")
	msg.WriteString(body)

	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(hostPort, auth, from, recipients, []byte(msg.String()))
}

// stripHTML rough for email body
func stripTags(s string) string {
	out := strings.Builder{}
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			out.WriteRune(r)
		}
	}
	return out.String()
}
