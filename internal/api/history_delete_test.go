package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
	"github.com/rohitkumar-co-in/leadomi-sip/internal/models"
)

func TestHistoryDeletionBoundaries(t *testing.T) {
	for _, kind := range []string{"sms", "call"} {
		for _, test := range []struct {
			name, scope, role, providerState string
			confirm                          bool
			upstream                         int
			want                             int
			kept                             bool
			deletes                          int
		}{
			{"local", "local", "admin", "", true, 204, 200, false, 0},
			{"both", "both", "admin", "", true, 204, 200, false, 1},
			{"provider failure", "both", "admin", "", true, 500, 502, true, 1},
			{"already removed", "both", "admin", "", true, 404, 200, false, 0},
			{"no confirmation", "both", "admin", "", false, 204, 400, true, 0},
			{"operator denied", "both", "user", "", true, 204, 403, true, 0},
			{"provider active", "both", "admin", "queued", true, 204, 409, true, 0},
			{"local record active", "both", "admin", "", true, 204, 409, true, 0},
			{"invalid provider ID", "both", "admin", "", true, 204, 409, true, 0},
			{"verification failure", "both", "admin", "", true, 401, 502, true, 0},
			{"bad scope", "everything", "admin", "", true, 204, 400, true, 0},
		} {
			t.Run(kind+"/"+test.name, func(t *testing.T) {
				setup := setupTestAPI(t)
				prefix, state, collection := "SM", "delivered", "Messages"
				if kind == "call" {
					prefix, state, collection = "CA", "completed", "Calls"
				}
				sid := prefix + strings.Repeat("a", 32)
				if test.name == "local record active" {
					if kind == "call" {
						state = "ringing"
					} else {
						state = "queued"
					}
				}
				if test.name == "invalid provider ID" {
					sid = "invalid"
				}
				var id int64
				if kind == "sms" {
					message := &models.Message{MessageSID: sid, Direction: "inbound", FromNumber: "+441234567890", ToNumber: "+441234567891", Body: "Private message", Status: state}
					if err := setup.DB.Messages.Create(context.Background(), message); err != nil {
						t.Fatal(err)
					}
					id = message.ID
				} else {
					call := &models.CDR{CallSID: sid, Direction: "inbound", FromNumber: "+441234567890", ToNumber: "+441234567891", StartedAt: time.Now(), Disposition: state}
					if err := setup.DB.CDRs.Create(context.Background(), call); err != nil {
						t.Fatal(err)
					}
					id = call.ID
				}
				deleted, requests := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if !strings.HasSuffix(r.URL.Path, "/"+collection+"/"+sid+".json") {
						t.Errorf("Unexpected provider path %s", r.URL.Path)
						w.WriteHeader(400)
						return
					}
					if r.Method == "DELETE" {
						deleted++
						w.WriteHeader(test.upstream)
						return
					}
					if test.upstream == 404 {
						w.WriteHeader(404)
						return
					}
					if test.upstream == 401 {
						w.WriteHeader(401)
						return
					}
					providerState := state
					if test.providerState != "" {
						providerState = test.providerState
					}
					json.NewEncoder(w).Encode(map[string]string{"sid": sid, "status": providerState})
				}))
				defer server.Close()
				h := newBusinessHandler(&Dependencies{DB: setup.DB, Config: &config.Config{TwilioAccountSID: "AC" + strings.Repeat("b", 32), TwilioAuthToken: "private-token"}})
				h.apiBase = server.URL
				body, _ := json.Marshal(map[string]interface{}{"scope": test.scope, "confirm": test.confirm})
				r := withURLParams(httptest.NewRequest("DELETE", "/activity", strings.NewReader(string(body))), map[string]string{"kind": kind, "id": "1"})
				r = r.WithContext(context.WithValue(r.Context(), contextKeyUser, &models.User{Role: test.role}))
				response := httptest.NewRecorder()
				h.DeleteActivity(response, r)
				assertStatus(t, response, test.want)
				if deleted != test.deletes {
					t.Fatalf("Unexpected provider deletions %d", deleted)
				}
				if test.scope == "local" && requests != 0 {
					t.Fatal("Local deletion contacted Twilio")
				}
				var exists int
				table := "messages"
				if kind == "call" {
					table = "cdrs"
				}
				if err := setup.DB.Conn().QueryRow("SELECT count(*) FROM "+table+" WHERE id=?", id).Scan(&exists); err != nil {
					t.Fatal(err)
				}
				if (exists == 1) != test.kept {
					t.Fatal("Local history preservation failed")
				}
				if strings.Contains(response.Body.String(), "private-token") || strings.Contains(response.Body.String(), "Private message") {
					t.Fatal("Private data exposed in response")
				}
			})
		}
	}
}
