package api

import (
	"crypto/tls"
	"fmt"
	"net"
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
	return sendSMTPMessage(user.Email, subject, body)
}

func sendPasswordResetEmail(user User, token string) error {
	link := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/") + "/reset-password?token=" + token
	subject := "Сброс пароля — SmetaCheck"
	body := fmt.Sprintf("Здравствуйте, %s!\r\n\r\nДля установки нового пароля откройте ссылку:\r\n%s\r\n\r\nСсылка действует 1 час. Если вы не запрашивали сброс, проигнорируйте письмо.\r\n", user.FullName, link)
	return sendSMTPMessage(user.Email, subject, body)
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
	if fromName == "" { fromName = "SmetaCheck" }
	if _, err := strconv.Atoi(port); err != nil { return fmt.Errorf("invalid SMTP_PORT") }

	message := []byte("From: " + fromName + " <" + from + ">\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n" +
		"\r\n" + body)

	address := net.JoinHostPort(host, port)
	var client *smtp.Client
	if port == "465" {
		connection, err := tls.Dial("tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err != nil { return err }
		client, err = smtp.NewClient(connection, host)
		if err != nil { connection.Close(); return err }
	} else {
		connection, err := net.DialTimeout("tcp", address, 10*time.Second)
		if err != nil { return err }
		client, err = smtp.NewClient(connection, host)
		if err != nil { connection.Close(); return err }
		if ok, _ := client.Extension("STARTTLS"); !ok {
			client.Close()
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil { client.Close(); return err }
	}
	defer client.Close()
	if username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil { return err }
	}
	if err := client.Mail(from); err != nil { return err }
	if err := client.Rcpt(recipient); err != nil { return err }
	writer, err := client.Data()
	if err != nil { return err }
	if _, err := writer.Write(message); err != nil { writer.Close(); return err }
	if err := writer.Close(); err != nil { return err }
	return client.Quit()
}
