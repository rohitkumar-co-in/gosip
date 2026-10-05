package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rohitkumar-co-in/gosip/internal/config"
	"github.com/rohitkumar-co-in/gosip/internal/models"
)

func TestNumberReviewPreservesExternalConnections(t *testing.T) {
	setup := setupTestAPI(t)
	ctx := context.Background()
	sid := "AC" + strings.Repeat("a", 32)
	pn := "PN" + strings.Repeat("a", 32)
	mg := "MG" + strings.Repeat("a", 32)
	h := newBusinessHandler(&Dependencies{DB: setup.DB, Config: &config.Config{TwilioAccountSID: sid, TwilioAuthToken: "test"}})
	h.client = &http.Client{Transport: businessTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatal("Read-only number review performed a write")
		}
		body := fmt.Sprintf(`{"services":[{"sid":%q,"use_inbound_webhook_on_number":false,"inbound_request_url":"https://existing.example/sms"}],"meta":{}}`, mg)
		if strings.Contains(r.URL.Path, "/PhoneNumbers") {
			body = fmt.Sprintf(`{"phone_numbers":[{"sid":%q}],"meta":{}}`, pn)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	n := businessNumber{SID: pn, Number: "+441234567890", VoiceURL: "https://existing.example/voice", SMSURL: "https://existing.example/sms", VoiceApplication: "AP" + strings.Repeat("a", 32)}
	result, err := h.reviewNumber(ctx, n)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || len(result.Connections) != 4 || result.Fingerprint == "" {
		t.Fatal("Connected number was not detected and blocked")
	}
	old := result.Fingerprint
	n.VoiceURL = "https://changed.example/voice"
	changed, err := h.reviewNumber(ctx, n)
	if err != nil || changed.Fingerprint == old {
		t.Fatal("Changed provider configuration retained the old confirmation")
	}
}

func TestActivityAttributionSurvivesReassignment(t *testing.T) {
	setup := setupTestAPI(t)
	ctx := context.Background()
	device := &models.Device{Name: "Agent", Username: "agent", PasswordHash: "hash", DeviceType: "linphone"}
	if err := setup.DB.Devices.Create(ctx, device); err != nil {
		t.Fatal(err)
	}
	setup.DB.Conn().Exec("INSERT INTO sip_accounts(device_id,number,state) VALUES(?,'+441234567890','ready')", device.ID)
	call := &models.CDR{CallSID: "test-call", Direction: "outbound", FromNumber: "+441234567890", ToNumber: "+441234567891", DeviceID: &device.ID, StartedAt: time.Now(), Disposition: "answered"}
	if err := setup.DB.CDRs.Create(ctx, call); err != nil {
		t.Fatal(err)
	}
	msg := &models.Message{MessageSID: "test-message", Direction: "inbound", FromNumber: "+441234567891", ToNumber: "+441234567890", Body: "Hello", Status: "received"}
	if err := setup.DB.Messages.Create(ctx, msg); err != nil {
		t.Fatal(err)
	}
	setup.DB.Conn().Exec("UPDATE devices SET username='renamed' WHERE id=?", device.ID)
	setup.DB.Conn().Exec("UPDATE sip_accounts SET number='+441234567899' WHERE device_id=?", device.ID)
	handler := newBusinessHandler(&Dependencies{DB: setup.DB})
	response := httptest.NewRecorder()
	handler.Activity(response, httptest.NewRequest("GET", "/activity", nil))
	assertStatus(t, response, 200)
	var data struct {
		Data []struct{ Kind, Actor string }
	}
	if json.Unmarshal(response.Body.Bytes(), &data) != nil || len(data.Data) != 2 {
		t.Fatal("Activity records could not be loaded")
	}
	for _, item := range data.Data {
		if item.Kind == "call" && item.Actor != "agent" {
			t.Fatal("Call actor changed with current username")
		}
		if item.Kind == "sms" && item.Actor != "Assigned to agent" {
			t.Fatal("SMS historical owner changed after reassignment")
		}
	}
}

func TestPhonePasswordsAndRecordingURLBoundary(t *testing.T) {
	for _, value := range []string{"short", "twelve chars!", "UnicodePasswordé", "a\nlongpassword", ""} {
		if validPhonePassword(value) {
			t.Fatal("Invalid custom phone password accepted")
		}
	}
	if !validPhonePassword("CustomPhonePassword123!") {
		t.Fatal("Custom password rejected")
	}
	account := "AC" + strings.Repeat("a", 32)
	recording := "RE" + strings.Repeat("b", 32)
	good := "https://api.twilio.com/2010-04-01/Accounts/" + account + "/Recordings/" + recording
	if _, ok := recordingURL(good, account); !ok {
		t.Fatal("Owned recording URL rejected")
	}
	for _, raw := range []string{strings.Replace(good, "api.twilio.com", "evil.example", 1), strings.Replace(good, account, "AC"+strings.Repeat("c", 32), 1), good + "?redirect=evil", "http://127.0.0.1/private"} {
		if _, ok := recordingURL(raw, account); ok {
			t.Fatal("Unsafe recording URL accepted")
		}
	}
}
