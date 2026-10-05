package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/btafoya/gosip/internal/config"
	"github.com/btafoya/gosip/internal/models"
)

func signedSoftphoneRequest(address string, fields url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, address, strings.NewReader(fields.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	mac := hmac.New(sha1.New, []byte("test-token"))
	mac.Write([]byte(address))
	for _, key := range keys {
		mac.Write([]byte(key + fields.Get(key)))
	}
	r.Header.Set("X-Twilio-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	return r
}

func TestSoftphoneOutboundRouting(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{TwilioAccountSID: "AC-test", TwilioAuthToken: "test-token", TwilioSIPDomain: "test.sip.twilio.com", OutboundCallerID: "+447575746354", PublicURL: "https://sip.example.test"}
	h := NewWebhookHandler(&Dependencies{Config: cfg, DB: setup.DB})
	did := &models.DID{Number: cfg.OutboundCallerID, VoiceEnabled: true}
	if err := setup.DB.DIDs.Create(context.Background(), did); err != nil {
		t.Fatal(err)
	}
	createTestDevice(t, setup.DB, "Mobile", "mobile")
	for _, test := range []struct {
		name, from, to string
		accepted       bool
	}{
		{"valid", "sip:mobile@test.sip.dublin.twilio.com", "sip:+441234567890@test.sip.dublin.twilio.com", true},
		{"unknown device", "sip:stranger@test.sip.twilio.com", "sip:+441234567890@test.sip.twilio.com", false},
		{"foreign host", "sip:mobile@evil.test", "sip:+441234567890@test.sip.twilio.com", false},
		{"foreign destination", "sip:mobile@test.sip.twilio.com", "sip:+441234567890@evil.test", false},
		{"invalid number", "sip:mobile@test.sip.twilio.com", "sip:999@test.sip.twilio.com", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := url.Values{"AccountSid": {"AC-test"}, "CallSid": {"CA-" + test.name}, "From": {test.from}, "To": {test.to}}
			rr := httptest.NewRecorder()
			h.VoiceOutgoing(rr, signedSoftphoneRequest(cfg.PublicURL+"/api/webhooks/voice/outgoing", fields))
			if strings.Contains(rr.Body.String(), "<Number>") != test.accepted {
				t.Fatal(rr.Body.String())
			}
			if test.accepted && !strings.Contains(rr.Body.String(), `callerId="+447575746354"`) {
				t.Fatal("wrong caller ID")
			}
		})
	}
	rr := httptest.NewRecorder()
	h.VoiceOutgoing(rr, httptest.NewRequest(http.MethodPost, cfg.PublicURL+"/api/webhooks/voice/outgoing", nil))
	assertStatus(t, rr, http.StatusForbidden)
}

func TestSoftphoneRingAndVoicemailFallback(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{TwilioAuthToken: "test-token", TwilioSIPDomain: "test.sip.twilio.com", PublicURL: "https://sip.example.test"}
	h := NewWebhookHandler(&Dependencies{Config: cfg, DB: setup.DB})
	did := &models.DID{Number: "+447575746354", VoiceEnabled: true}
	if err := setup.DB.DIDs.Create(context.Background(), did); err != nil {
		t.Fatal(err)
	}
	device := createTestDevice(t, setup.DB, "Mobile", "mobile")
	action, _ := json.Marshal(map[string]any{"devices": []int64{device.ID}, "timeout": 30})
	body := h.executeAction(context.Background(), &models.Route{ActionType: "ring", ActionData: action}, did, "+441234567890", "CA-test")
	decoder := xml.NewDecoder(strings.NewReader(body))
	responses := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if tag, ok := token.(xml.StartElement); ok && tag.Name.Local == "Response" {
			responses++
		}
	}
	if responses != 1 || !strings.Contains(body, "sip:mobile@test.sip.twilio.com;transport=tls") {
		t.Fatal(body)
	}
	address := cfg.PublicURL + "/api/webhooks/voice/dial-complete?DidId=1"
	for _, status := range []string{"completed", "no-answer", "busy"} {
		rr := httptest.NewRecorder()
		h.VoiceDialComplete(rr, signedSoftphoneRequest(address, url.Values{"DialCallStatus": {status}}))
		if strings.Contains(rr.Body.String(), "<Record") != (status != "completed") {
			t.Fatal(rr.Body.String())
		}
	}
}
