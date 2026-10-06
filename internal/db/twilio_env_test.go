package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
)

func TestMigrateTwilioCredentialsPreservesValues(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()
	database.Config.Set(ctx, "twilio_account_sid", "AC-legacy")
	database.Config.Set(ctx, "twilio_auth_token", "legacy-token")
	cfg := &config.Config{DataDir: t.TempDir()}
	if err := database.Config.MigrateTwilioEnv(cfg); err != nil {
		t.Fatal(err)
	}
	restarted := &config.Config{DataDir: cfg.DataDir}
	if err := restarted.LoadTwilioEnv(); err != nil {
		t.Fatal(err)
	}
	if sid, token := restarted.TwilioCredentials(); sid != "AC-legacy" || token != "legacy-token" {
		t.Fatal("migration lost credentials")
	}
	for _, key := range []string{"twilio_account_sid", "twilio_auth_token"} {
		if _, err := database.Config.Get(ctx, key); err != ErrConfigNotFound {
			t.Fatal("legacy credentials remain in database")
		}
	}
}

func TestFailedTwilioMigrationKeepsDatabaseCredentials(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()
	database.Config.Set(ctx, "twilio_account_sid", "AC-legacy")
	database.Config.Set(ctx, "twilio_auth_token", "legacy-token")
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := database.Config.MigrateTwilioEnv(&config.Config{DataDir: blocked}); err == nil {
		t.Fatal("migration unexpectedly succeeded")
	}
	if token, err := database.Config.Get(ctx, "twilio_auth_token"); err != nil || token != "legacy-token" {
		t.Fatal("failed migration removed original credentials")
	}
}
