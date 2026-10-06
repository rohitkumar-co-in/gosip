package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
	"github.com/rohitkumar-co-in/leadomi-sip/internal/models"
)

func TestSeparatePhoneNumbersForVoiceAndSMS(t *testing.T) {
	setup := setupTestAPI(t)
	cfg := &config.Config{PBXSecret: "bridge-secret", TwilioAccountSID: "AC-test", TwilioAuthToken: "test-token", TwilioSIPDomain: "test.sip.twilio.com", OutboundCallerID: "+447575746354", PublicURL: "https://sip.example.test"}
	ctx := context.Background()
	if err := setup.DB.Config.Set(ctx, "pbx_device_numbers", `{"phone2":"+447576582200"}`); err != nil {
		t.Fatal(err)
	}
	sent := make(chan string, 2)
	client := &MockTwilioClient{SendSMSFunc: func(from, to, body string, media []string) (string, error) {
		sent <- from
		return "SM-" + from, nil
	}}
	deps := &Dependencies{Config: cfg, DB: setup.DB, Twilio: client}
	for _, phone := range []struct{ username, number string }{{"phone1", cfg.OutboundCallerID}, {"phone2", "+447576582200"}} {
		createTestDevice(t, setup.DB, phone.username, phone.username)
		did := &models.DID{Number: phone.number, VoiceEnabled: true, SMSEnabled: true}
		if err := setup.DB.DIDs.Create(ctx, did); err != nil {
			t.Fatal(err)
		}
		fields := url.Values{"AccountSid": {"AC-test"}, "From": {"sip:" + phone.username + "@test.sip.twilio.com"}, "To": {"sip:+441234567890@test.sip.twilio.com:5061;transport=tls;secure=true"}}
		rr := httptest.NewRecorder()
		NewWebhookHandler(deps).VoiceOutgoing(rr, signedSoftphoneRequest(cfg.PublicURL+"/api/webhooks/voice/outgoing", fields))
		if !strings.Contains(rr.Body.String(), `callerId="`+phone.number+`"`) {
			t.Fatal(rr.Body.String())
		}
		body, _ := json.Marshal(map[string]string{"username": phone.username, "to_number": "+441234567890", "body": "test", "from_number": cfg.OutboundCallerID})
		r := httptest.NewRequest("POST", "/api/pbx/messages", strings.NewReader(string(body)))
		r.Header.Set("Authorization", "Bearer "+cfg.PBXSecret)
		rr = httptest.NewRecorder()
		(&PBXHandler{deps: deps}).SendMessage(rr, r)
		assertStatus(t, rr, 202)
		select {
		case from := <-sent:
			if from != phone.number {
				t.Fatalf("%s SMS used %s instead of %s", phone.username, from, phone.number)
			}
		case <-time.After(time.Second):
			t.Fatal("SMS was not submitted to mock")
		}
	}
	if err := setup.DB.Config.Set(ctx, "pbx_device_numbers", `{"phone2":"invalid"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := deviceOutboundNumber(ctx, deps, "phone2"); err == nil {
		t.Fatal("Invalid assignment must not fall back to another number")
	}
}
