package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/btafoya/gosip/internal/config"
	"github.com/btafoya/gosip/internal/db"
)

func TestSetupStoresTwilioOutsideDatabase(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{DataDir: t.TempDir()}
	handler := NewSystemHandler(&Dependencies{DB: setup.DB, Config: cfg})
	body := `{"twilio_account_sid":"AC-test","twilio_auth_token":"token-test","admin_email":"env-test@example.com","admin_password":"test-password-123"}`
	rr := httptest.NewRecorder()
	handler.SetupWizard(rr, httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(body)))
	assertStatus(t, rr, http.StatusOK)
	for _, key := range []string{"twilio_account_sid", "twilio_auth_token"} {
		if _, err := setup.DB.Config.Get(context.Background(), key); err != db.ErrConfigNotFound {
			t.Fatal("Twilio credential stored in database")
		}
	}
	restarted := &config.Config{DataDir: cfg.DataDir}
	if err := restarted.LoadTwilioEnv(); err != nil {
		t.Fatal(err)
	}
	if sid, token := restarted.TwilioCredentials(); sid != "AC-test" || token != "token-test" {
		t.Fatal("saved credentials not restored")
	}
	configResponse := httptest.NewRecorder()
	handler.GetConfig(configResponse, httptest.NewRequest(http.MethodGet, "/api/system/config", nil))
	if bytes.Contains(configResponse.Body.Bytes(), []byte("token-test")) {
		t.Fatal("configuration response exposed token")
	}
}

func TestWebhookUsesRotatedEnvironmentToken(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{DataDir: t.TempDir()}
	if err := cfg.SaveTwilioEnv("AC-test", "first"); err != nil {
		t.Fatal(err)
	}
	handler := NewWebhookHandler(&Dependencies{DB: setup.DB, Config: cfg})
	sign := func(token string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "https://example.com/api/webhooks/sms/incoming", strings.NewReader("Body=hello"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		mac := hmac.New(sha1.New, []byte(token))
		mac.Write([]byte("https://example.com/api/webhooks/sms/incomingBodyhello"))
		req.Header.Set("X-Twilio-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		return req
	}
	if !handler.validateSignature(sign("first")) {
		t.Fatal("valid environment-token signature rejected")
	}
	if err := cfg.SaveTwilioEnv("", "second"); err != nil {
		t.Fatal(err)
	}
	if handler.validateSignature(sign("first")) || !handler.validateSignature(sign("second")) {
		t.Fatal("webhook validation did not follow token rotation")
	}
}

func TestSetupUsesRuntimeCredentialsWithoutCopyingSecrets(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "AC-runtime")
	t.Setenv("TWILIO_AUTH_TOKEN", "runtime-token")
	setup := setupTestAPI(t)
	cfg := &config.Config{DataDir: t.TempDir()}
	if err := cfg.LoadTwilioEnv(); err != nil {
		t.Fatal(err)
	}
	handler := NewSystemHandler(&Dependencies{DB: setup.DB, Config: cfg})
	body := `{"admin_email":"runtime@example.com","admin_password":"test-password-123"}`
	rr := httptest.NewRecorder()
	handler.SetupWizard(rr, httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(body)))
	assertStatus(t, rr, http.StatusOK)
	if _, err := os.Stat(cfg.TwilioEnvPath()); !os.IsNotExist(err) {
		t.Fatal("setup copied runtime secrets to a file")
	}
	if _, err := setup.DB.Config.Get(context.Background(), "twilio_auth_token"); err != db.ErrConfigNotFound {
		t.Fatal("setup copied runtime secret to SQLite")
	}
}
