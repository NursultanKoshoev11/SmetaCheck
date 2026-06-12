package api

import (
	"strings"
	"testing"
)

func TestBuildSMTPMessageEncodesUnicodeHeadersAndBody(t *testing.T) {
	message, from, recipient, err := buildSMTPMessage(
		"no-reply@smetacheck.kg",
		"SmetaCheck KG",
		"user@example.com",
		"Подтвердите email — SmetaCheck",
		"Здравствуйте!\r\nПроверьте ссылку.\r\n",
	)
	if err != nil {
		t.Fatalf("buildSMTPMessage returned error: %v", err)
	}
	if from != "no-reply@smetacheck.kg" {
		t.Fatalf("unexpected envelope sender: %q", from)
	}
	if recipient != "user@example.com" {
		t.Fatalf("unexpected envelope recipient: %q", recipient)
	}

	text := string(message)
	for _, expected := range []string{
		"Subject: =?UTF-8?q?",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"From: SmetaCheck KG <no-reply@smetacheck.kg>",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("message does not contain %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "Подтвердите email") {
		t.Fatal("unicode subject must be MIME encoded")
	}
}

func TestBuildSMTPMessageRejectsHeaderInjection(t *testing.T) {
	_, _, _, err := buildSMTPMessage(
		"no-reply@smetacheck.kg",
		"SmetaCheck\r\nBcc: attacker@example.com",
		"user@example.com",
		"Verify",
		"body",
	)
	if err == nil {
		t.Fatal("expected header injection to be rejected")
	}
}

func TestSMTPModeDefaults(t *testing.T) {
	t.Setenv("SMTP_TLS_MODE", "")
	if got := smtpTLSMode("465"); got != "implicit" {
		t.Fatalf("port 465: expected implicit, got %q", got)
	}
	if got := smtpTLSMode("587"); got != "starttls" {
		t.Fatalf("port 587: expected starttls, got %q", got)
	}
}

func TestSMTPModeExplicit(t *testing.T) {
	t.Setenv("SMTP_TLS_MODE", "none")
	if got := smtpTLSMode("2525"); got != "none" {
		t.Fatalf("expected explicit none mode, got %q", got)
	}
}
