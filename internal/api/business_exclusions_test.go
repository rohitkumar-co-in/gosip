package api

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExcludedNumbersFailClosed(t *testing.T) {
	setup := setupTestAPI(t)
	deps := &Dependencies{DB: setup.DB, Config: &config.Config{PBXURL: "http://pbx.test"}}
	h := newBusinessHandler(deps)
	requests := 0
	h.client = &http.Client{Transport: businessTransport(func(*http.Request) (*http.Response, error) { requests++; return nil, fmt.Errorf("unexpected network") })}
	invoke := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		router := chi.NewRouter()
		router.Get("/excluded", h.Exclusions)
		router.Put("/excluded", h.Exclusions)
		router.Delete("/excluded", h.Exclusions)
		router.Post("/accounts", h.Provision)
		router.ServeHTTP(rec, request)
		return rec
	}
	assertStatus(t, invoke("GET", "/excluded", ""), 200)
	assertStatus(t, invoke("PUT", "/excluded", `{"number":"447000000000"}`), 400)
	assertStatus(t, invoke("PUT", "/excluded", `{"number":"+447000000000","reason":"Keep external service"}`), 200)
	assertStatus(t, invoke("PUT", "/excluded", `{"number":"+447000000000","reason":"Updated"}`), 200)
	entries, err := numberExclusions(context.Background(), setup.DB)
	if err != nil || len(entries) != 1 || entries[0].Reason != "Updated" || entries[0].CreatedAt == "" {
		t.Fatal("Protection did not persist idempotently")
	}
	assertStatus(t, invoke("POST", "/accounts", `{"name":"Blocked","username":"blocked","number":"+447000000000","number_review":"old-review"}`), 409)
	assertStatus(t, invoke("POST", "/accounts", `{"name":"Blocked","username":"blocked","number":"+447000000000","force":true}`), 400)
	if requests != 0 {
		t.Fatal("Protection performed external requests")
	}
	// Existing and new/global route writes are protected, including reorder.
	did := createTestDID(t, setup.DB, "+447000000000")
	route := createTestRoute(t, setup, "Protected route", &did.ID)
	rh := NewRouteHandler(deps)
	router := chi.NewRouter()
	router.Post("/routes", rh.Create)
	router.Put("/routes/{id}", rh.Update)
	router.Delete("/routes/{id}", rh.Delete)
	router.Post("/reorder", rh.Reorder)
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/routes", fmt.Sprintf(`{"name":"Blocked","did_id":%d,"condition_type":"default","action_type":"reject"}`, did.ID)},
		{"POST", "/routes", `{"name":"Global","condition_type":"default","action_type":"reject"}`},
		{"PUT", fmt.Sprintf("/routes/%d", route.ID), `{"name":"Changed","enabled":false}`},
		{"DELETE", fmt.Sprintf("/routes/%d", route.ID), ""},
		{"POST", "/reorder", fmt.Sprintf(`{"priorities":{"%d":99}}`, route.ID)},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))
		assertStatus(t, rec, 409)
	}
	saved, _ := setup.DB.Routes.GetByID(context.Background(), route.ID)
	if saved.Name != "Protected route" || !saved.Enabled || saved.Priority != 1 {
		t.Fatal("Protected routing changed")
	}
	assertStatus(t, invoke("DELETE", "/excluded?number=%2B447000000000", ""), 200)
	if err = checkNumberProtection(context.Background(), setup.DB, did.Number); err != nil {
		t.Fatal(err)
	}
	setup.DB.Config.Set(context.Background(), "business_excluded_numbers", "corrupt")
	assertStatus(t, invoke("POST", "/accounts", `{"name":"Blocked","username":"blocked","number":"+447000000000"}`), 409)
	assertStatus(t, invoke("GET", "/excluded", ""), 503)
	if requests != 0 {
		t.Fatal("Invalid protection allowed network side effects")
	}
}
