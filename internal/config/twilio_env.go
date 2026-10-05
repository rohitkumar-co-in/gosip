package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrTwilioRuntimeManaged = errors.New("Twilio credentials are managed by runtime environment variables")

// TwilioEnvPath is outside frontend assets and is retained on the data volume.
func (c *Config) TwilioEnvPath() string {
	return filepath.Join(c.DataDir, ".env")
}

func (c *Config) TwilioCredentials() (string, string) {
	if c == nil {
		return "", ""
	}
	c.twilioMu.RLock()
	defer c.twilioMu.RUnlock()
	return c.TwilioAccountSID, c.TwilioAuthToken
}

// LoadTwilioEnv gives explicit process credentials priority over the UI-managed file.
// Parsing does not execute shell commands or expand variables.
func (c *Config) LoadTwilioEnv() error {
	if sid, token := os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_AUTH_TOKEN"); sid != "" || token != "" {
		if sid == "" || token == "" {
			return errors.New("both Twilio runtime environment variables are required")
		}
		c.twilioMu.Lock()
		c.TwilioAccountSID, c.TwilioAuthToken = sid, token
		c.twilioMu.Unlock()
		return nil
	}
	file, err := os.Open(c.TwilioEnvPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open Twilio environment file: %w", err)
	}
	defer file.Close()
	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || (key != "TWILIO_ACCOUNT_SID" && key != "TWILIO_AUTH_TOKEN") {
			return errors.New("invalid Twilio environment file entry")
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "\"") {
			value, err = strconv.Unquote(value)
			if err != nil {
				return errors.New("invalid quoted Twilio environment value")
			}
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return errors.New("invalid Twilio environment value")
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read Twilio environment file: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("protect Twilio environment file: %w", err)
	}
	c.twilioMu.Lock()
	defer c.twilioMu.Unlock()
	if value, ok := values["TWILIO_ACCOUNT_SID"]; ok {
		c.TwilioAccountSID = value
	}
	if value, ok := values["TWILIO_AUTH_TOKEN"]; ok {
		c.TwilioAuthToken = value
	}
	return nil
}

// SaveTwilioEnv atomically saves credentials with owner-only permissions.
// Empty fields preserve the current value, supporting token-only rotations.
func (c *Config) SaveTwilioEnv(accountSID, authToken string) error {
	if c == nil {
		return errors.New("application configuration unavailable")
	}
	if strings.ContainsAny(accountSID+authToken, "\r\n\x00") {
		return errors.New("invalid Twilio credential value")
	}
	c.twilioMu.Lock()
	defer c.twilioMu.Unlock()
	if sid, token := os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_AUTH_TOKEN"); sid != "" || token != "" {
		if sid == "" || token == "" {
			return errors.New("both Twilio runtime environment variables are required")
		}
		if (accountSID != "" && accountSID != sid) || (authToken != "" && authToken != token) {
			return ErrTwilioRuntimeManaged
		}
		c.TwilioAccountSID, c.TwilioAuthToken = sid, token
		return nil
	}
	if accountSID == "" {
		accountSID = c.TwilioAccountSID
	}
	if authToken == "" {
		authToken = c.TwilioAuthToken
	}
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	file, err := os.CreateTemp(c.DataDir, ".twilio-env-*")
	if err != nil {
		return fmt.Errorf("create credential file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	defer file.Close()
	content := "TWILIO_ACCOUNT_SID=" + strconv.Quote(accountSID) + "\nTWILIO_AUTH_TOKEN=" + strconv.Quote(authToken) + "\n"
	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("write credential file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync credential file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close credential file: %w", err)
	}
	if err := os.Rename(tempPath, c.TwilioEnvPath()); err != nil {
		return fmt.Errorf("replace credential file: %w", err)
	}
	c.TwilioAccountSID, c.TwilioAuthToken = accountSID, authToken
	return nil
}
