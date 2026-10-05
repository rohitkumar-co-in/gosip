package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTwilioEnvPersistenceAndRotation(t *testing.T) {
	cfg := &Config{DataDir: t.TempDir()}
	if err := cfg.SaveTwilioEnv("AC-test", "token-first"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cfg.TwilioEnvPath())
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("credentials must have owner-only file permissions")
	}
	restarted := &Config{DataDir: cfg.DataDir, TwilioAccountSID: "environment-default"}
	if err := restarted.LoadTwilioEnv(); err != nil {
		t.Fatal(err)
	}
	if sid, token := restarted.TwilioCredentials(); sid != "AC-test" || token != "token-first" {
		t.Fatal("restart did not restore saved credentials")
	}
	if err := restarted.SaveTwilioEnv("", "token-rotated"); err != nil {
		t.Fatal(err)
	}
	if sid, token := restarted.TwilioCredentials(); sid != "AC-test" || token != "token-rotated" {
		t.Fatal("token rotation did not preserve SID")
	}
}

func TestTwilioEnvRejectsInjectionAndPreservesCredentials(t *testing.T) {
	cfg := &Config{DataDir: t.TempDir()}
	if err := cfg.SaveTwilioEnv("AC-test", "original"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTwilioEnv("", "bad\nOTHER_SECRET=oops"); err == nil {
		t.Fatal("newline injection accepted")
	}
	cfg.DataDir = filepath.Join(cfg.DataDir, "not-a-directory")
	if err := os.WriteFile(cfg.DataDir, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTwilioEnv("", "replacement"); err == nil {
		t.Fatal("save to invalid directory succeeded")
	}
	if _, token := cfg.TwilioCredentials(); token != "original" {
		t.Fatal("failed save changed active token")
	}
}

func TestTwilioEnvMalformedFileDoesNotPartiallyUpdate(t *testing.T) {
	cfg := &Config{DataDir: t.TempDir(), TwilioAccountSID: "AC-original", TwilioAuthToken: "original"}
	if err := os.WriteFile(cfg.TwilioEnvPath(), []byte("TWILIO_ACCOUNT_SID=AC-changed\nINVALID\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.LoadTwilioEnv(); err == nil {
		t.Fatal("malformed file accepted")
	}
	if sid, token := cfg.TwilioCredentials(); sid != "AC-original" || token != "original" {
		t.Fatal("malformed file partially changed credentials")
	}
}

func TestRuntimeTwilioCredentialsDoNotCreateFile(t *testing.T) {
	t.Setenv("TWILIO_ACCOUNT_SID", "AC-runtime")
	t.Setenv("TWILIO_AUTH_TOKEN", "runtime-token")
	cfg := &Config{DataDir: t.TempDir()}
	if err := cfg.LoadTwilioEnv(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTwilioEnv("", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.TwilioEnvPath()); !os.IsNotExist(err) {
		t.Fatal("runtime credentials were copied to a file")
	}
	if err := cfg.SaveTwilioEnv("AC-runtime", "different-token"); err == nil {
		t.Fatal("UI replaced runtime-managed credentials")
	}
	if sid, token := cfg.TwilioCredentials(); sid != "AC-runtime" || token != "runtime-token" {
		t.Fatal("runtime credentials changed")
	}
}
