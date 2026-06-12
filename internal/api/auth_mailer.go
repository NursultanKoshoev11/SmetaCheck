package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

func sendVerificationEmail(user User, token string) error {
	link := strings.TrimRight(strings.TrimSpace(os.Getenv("API_PUBLIC_BASE_URL")), "/") + "/v1/auth/email/verify?token=" + token
	subject := "Подтвердите email — SmetaCheck"
	body := fmt.Sprintf("Здравствуйте, %s!\r\n\r\nПодтвердите email для входа в SmetaCheck:\r\n%s\r\n\r\nСсылка действует 24 часа. Если вы не создавали аккаунт, проигнорируйте письмо.\r\n", user.FullName, link)
	if err := sendSMTPMessage(user.Email, subject, body); err != nil {
		log.Printf("verification email delivery failed recipient_domain=%s err=%v", emailDomain(user.Email), err)
		return err
	}
	return nil
}

func sendPasswordResetEmail(user User, token string) error {
	link := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/") + "/reset-password?token=" + token
	subject := "Сброс пароля — SmetaCheck"
	body := fmt.Sprintf("Здравствуйте, %s!\r\n\r\nДля установки нового пароля откройте ссылку:\r\n%s\r\n\r\nСсылка действует 1 час. Если вы не запрашивали сброс, проигнорируйте письмо.\r\n", user.FullName, link)
	if err := sendSMTPMessage(user.Email, subject, body); err != nil {
		log.Printf("password reset email delivery failed recipient_domain=%s err=%v", emailDomain(user.Email), err)
		return err
	}
	return nil
}

func sendSMTPMessage(recipient, subject, body string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := os.Getenv("SMTP_PASSWORD")
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	fromName := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME"))
	if host == "" || port == "" || from == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT and SMTP_FROM are required")
	}
	if fromName == "" {
		fromName = "SmetaCheck"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	if (username == "") != (password == "") {
		return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be configured together")
	}

	message, envelopeFrom, envelopeRecipient, err := buildSMTPMessage(from, fromName, recipient, subject, body)
	if err != nil {
		return err
	}

	address := net.JoinHostPort(host, port)
	timeout := envDuration("SMTP_TIMEOUT", 15*time.Second)
	if timeout < time.Second {
		timeout = time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	mode := smtpTLSMode(port)

	var client *smtp.Client
	switch mode {
	case "implicit":
		connection, dialErr := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if dialErr != nil {
			return fmt.Errorf("connect to SMTP server using implicit TLS: %w", dialErr)
		}
		if err := connection.SetDeadline(time.Now().Add(timeout)); err != nil {
			_ = connection.Close()
			return fmt.Errorf("set SMTP deadline: %w", err)
		}
		client, err = smtp.NewClient(connection, host)
		if err != nil {
			_ = connection.Close()
			return fmt.Errorf("initialize SMTP client: %w", err)
		}
	case "starttls", "none":
		connection, dialErr := dialer.Dial("tcp", address)
		if dialErr != nil {
			return fmt.Errorf("connect to SMTP server: %w", dialErr)
		}
		if err := connection.SetDeadline(time.Now().Add(timeout)); err != nil {
			_ = connection.Close()
			return fmt.Errorf("set SMTP deadline: %w", err)
		}
		client, err = smtp.NewClient(connection, host)
		if err != nil {
			_ = connection.Close()
			return fmt.Errorf("initialize SMTP client: %w", err)
		}
		if mode == "starttls" {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				_ = client.Close()
				return fmt.Errorf("SMTP server does not support STARTTLS")
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				_ = client.Close()
				return fmt.Errorf("start SMTP TLS: %w", err)
			}
		}
	default:
		return fmt.Errorf("unsupported SMTP_TLS_MODE %q", mode)
	}
	defer client.Close()

	if username != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return fmt.Errorf("SMTP server does not advertise AUTH")
		}
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return fmt.Errorf("authenticate to SMTP server: %w", err)
		}
	}
	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(envelopeRecipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message body: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP session: %w", err)
	}
	return nil
}

func buildSMTPMessage(from, fromName, recipient, subject, body string) ([]byte, string, string, error) {
	if strings.ContainsAny(subject, "\r\n") || strings.ContainsAny(fromName, "\r\n") {
		return nil, "", "", fmt.Errorf("invalid email header")
	}
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	recipientAddress, err := mail.ParseAddress(recipient)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid recipient email: %w", err)
	}

	var encodedBody bytes.Buffer
	quotedWriter := quotedprintable.NewWriter(&encodedBody)
	if _, err := quotedWriter.Write([]byte(body)); err != nil {
		return nil, "", "", fmt.Errorf("encode email body: %w", err)
	}
	if err := quotedWriter.Close(); err != nil {
		return nil, "", "", fmt.Errorf("finish email body encoding: %w", err)
	}

	fromHeader := (&mail.Address{Name: fromName, Address: fromAddress.Address}).String()
	message := []byte("From: " + fromHeader + "\r\n" +
		"To: " + recipientAddress.Address + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n" +
		"\r\n" + encodedBody.String())
	return message, fromAddress.Address, recipientAddress.Address, nil
}

func smtpTLSMode(port string) string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_TLS_MODE")))
	if mode != "" {
		return mode
	}
	if port == "465" {
		return "implicit"
	}
	return "starttls"
}

func emailDomain(address string) string {
	at := strings.LastIndex(strings.TrimSpace(address), "@")
	if at < 0 || at == len(address)-1 {
		return "unknown"
	}
	return strings.ToLower(address[at+1:])
}
