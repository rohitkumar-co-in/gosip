package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btafoya/gosip/internal/db"
)

// Number assignments are server-side settings, never supplied by the phone.
func deviceOutboundNumber(ctx context.Context, deps *Dependencies, username string) (string, error) {
	value, err := deps.DB.Config.Get(ctx, "pbx_device_numbers")
	if errors.Is(err, db.ErrConfigNotFound) {
		return deps.Config.OutboundCallerID, nil
	}
	if err != nil {
		return "", err
	}
	var assignments map[string]string
	if err := json.Unmarshal([]byte(value), &assignments); err != nil {
		return "", err
	}
	if assignments == nil {
		return "", fmt.Errorf("invalid phone number assignments")
	}
	if number, ok := assignments[username]; ok {
		if !e164Destination.MatchString(number) {
			return "", fmt.Errorf("invalid assigned phone number")
		}
		return number, nil
	}
	return deps.Config.OutboundCallerID, nil
}
