package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
	"github.com/rohitkumar-co-in/leadomi-sip/internal/models"
)

func TestPBXMessageAuthorization(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{PBXSecret: "a-private-bridge-secret", OutboundCallerID: "+447575746354"}
	deps := &Dependencies{Config: cfg, DB: setup.DB, Twilio: &MockTwilioClient{}}
	createTestDevice(t, setup.DB, "Phone", "phone")
	did := &models.DID{Number: cfg.OutboundCallerID, SMSEnabled: true}
	if err := setup.DB.DIDs.Create(context.Background(), did); err != nil {
		t.Fatal(err)
	}
	h := &PBXHandler{deps: deps}
	for _, test := range []struct {
		name, secret, username, number, body string
		status                               int
	}{
		{"missing secret", "", "phone", "+441234567890", "hello", 403},
		{"wrong secret", "wrong", "phone", "+441234567890", "hello", 403},
		{"unknown sender", cfg.PBXSecret, "unknown", "+441234567890", "hello", 403},
		{"invalid number", cfg.PBXSecret, "phone", "999", "hello", 400},
		{"too long", cfg.PBXSecret, "phone", "+441234567890", strings.Repeat("x", 1601), 400},
		{"accepted", cfg.PBXSecret, "phone", "+441234567890", "hello\nworld", 202},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"username": test.username, "to_number": test.number, "body": test.body})
			r := httptest.NewRequest("POST", "/api/pbx/messages", strings.NewReader(string(body)))
			r.Header.Set("Authorization", "Bearer "+test.secret)
			rr := httptest.NewRecorder()
			h.SendMessage(rr, r)
			assertStatus(t, rr, test.status)
		})
	}
}

func TestSMSIncomingRequiresTwilioSignature(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{PublicURL: "https://sip.example.test", TwilioAuthToken: "test-token"}
	h := NewWebhookHandler(&Dependencies{Config: cfg, DB: setup.DB})
	did := &models.DID{Number: "+447575746354", SMSEnabled: true}
	if err := setup.DB.DIDs.Create(context.Background(), did); err != nil {
		t.Fatal(err)
	}
	fields := url.Values{"From": {"+441234567890"}, "To": {did.Number}, "Body": {"hello"}, "MessageSid": {"SM-signed"}}
	address := cfg.PublicURL + "/api/webhooks/sms/incoming"
	r := httptest.NewRequest(http.MethodPost, address, strings.NewReader(fields.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.SMSIncoming(rr, r)
	assertStatus(t, rr, 403)
	rr = httptest.NewRecorder()
	h.SMSIncoming(rr, signedSoftphoneRequest(address, fields))
	assertStatus(t, rr, 200)
	count, _ := setup.DB.Messages.Count(context.Background())
	if count != 1 {
		t.Fatalf("stored %d messages", count)
	}
}

func TestPBXIncomingDialTarget(t *testing.T) {
	setup := setupTestAPI(t)
	device := createTestDevice(t, setup.DB, "Phone", "phone")
	cfg := &config.Config{PBXSIPDomain: "sip.example.test:5061", PBXTrunkUser: "trunk", PBXTrunkPassword: "a&b", TwilioSIPDomain: "old.sip.twilio.com"}
	h := NewWebhookHandler(&Dependencies{Config: cfg, DB: setup.DB})
	data, _ := json.Marshal(map[string]any{"devices": []int64{device.ID}})
	result := h.executeAction(context.Background(), &models.Route{ActionType: "ring", ActionData: data}, &models.DID{ID: 1}, "+441234567890", "")
	if !strings.Contains(result, `username="trunk" password="a&amp;b"`) || !strings.Contains(result, "sip:phone@sip.example.test:5061;transport=tls;secure=true") {
		t.Fatal(result)
	}
}
