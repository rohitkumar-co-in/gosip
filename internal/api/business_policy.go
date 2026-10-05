package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rohitkumar-co-in/gosip/internal/db"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var businessSMSMu sync.Mutex

type businessPolicy struct {
	Prefixes []string `json:"allowed_prefixes"`
	SMSLimit int      `json:"sms_per_minute"`
}

func loadBusinessPolicy(ctx context.Context, deps *Dependencies) (businessPolicy, error) {
	policy := businessPolicy{Prefixes: []string{}, SMSLimit: 20}
	raw, err := deps.DB.Config.Get(ctx, "business_policy")
	if errors.Is(err, db.ErrConfigNotFound) {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	err = json.Unmarshal([]byte(raw), &policy)
	return policy, err
}
func businessDestinationAllowed(ctx context.Context, deps *Dependencies, number string) bool {
	if deps.Config == nil || deps.Config.PBXURL == "" {
		return true
	}
	policy, err := loadBusinessPolicy(ctx, deps)
	if err != nil {
		return false
	}
	if len(policy.Prefixes) == 0 {
		return true
	}
	for _, prefix := range policy.Prefixes {
		if strings.HasPrefix(number, prefix) {
			return true
		}
	}
	return false
}
func (h *BusinessHandler) Policy(w http.ResponseWriter, r *http.Request) {
	policy, err := loadBusinessPolicy(r.Context(), h.deps)
	if err != nil {
		WriteInternalError(w)
		return
	}
	if r.Method == "PUT" {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&policy) != nil || len(policy.Prefixes) > 30 || policy.SMSLimit < 1 || policy.SMSLimit > 120 {
			WriteValidationError(w, "Choose 1–120 SMS per minute and up to 30 international prefixes", nil)
			return
		}
		for _, prefix := range policy.Prefixes {
			if !regexp.MustCompile(`^\+[1-9][0-9]{0,3}$`).MatchString(prefix) {
				WriteValidationError(w, "Use international prefixes such as +44 or +91", nil)
				return
			}
		}
		raw, _ := json.Marshal(policy)
		if h.deps.DB.Config.Set(r.Context(), "business_policy", string(raw)) != nil {
			WriteInternalError(w)
			return
		}
		h.audit(r.Context(), getUserIDFromContext(r.Context()), "usage_policy", "business", "updated")
	}
	WriteJSON(w, 200, policy)
}
func businessSMSLimit(ctx context.Context, deps *Dependencies, number string) bool {
	if deps.Config == nil || deps.Config.PBXURL == "" {
		return true
	}
	policy, err := loadBusinessPolicy(ctx, deps)
	if err != nil {
		return false
	}
	var count int
	err = deps.DB.Conn().QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE direction='outbound' AND from_number=? AND datetime(created_at)>=datetime(?)", number, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)).Scan(&count)
	return err == nil && count < policy.SMSLimit
}
