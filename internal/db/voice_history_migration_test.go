package db

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestVoiceHistoryMigrationPreservesRecords(t *testing.T) {
	database, err := New(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.conn.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") || entry.Name() >= "012_" {
			continue
		}
		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.conn.Exec(string(content)); err != nil {
			t.Fatal(err)
		}
		version, _ := strconv.Atoi(strings.SplitN(entry.Name(), "_", 2)[0])
		if _, err := database.conn.Exec("INSERT INTO schema_migrations(version) VALUES(?)", version); err != nil {
			t.Fatal(err)
		}
	}
	_, err = database.conn.Exec(`INSERT INTO cdrs(id,call_sid,direction,from_number,to_number,started_at,disposition) VALUES(42,'old-call','inbound','+441234567890','+441234567891',CURRENT_TIMESTAMP,'voicemail');
UPDATE activity_actors SET actor='Original agent' WHERE kind='call' AND record_id=42;
INSERT INTO voicemails(id,cdr_id,from_number) VALUES(7,42,'+441234567890');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatal(err)
	}
	var linked int
	var actor, disposition string
	if err := database.conn.QueryRow("SELECT cdr_id FROM voicemails WHERE id=7").Scan(&linked); err != nil || linked != 42 {
		t.Fatalf("Voicemail link lost: %d %v", linked, err)
	}
	if err := database.conn.QueryRow("SELECT actor FROM activity_actors WHERE kind='call' AND record_id=42").Scan(&actor); err != nil || actor != "Original agent" {
		t.Fatalf("Historical attribution lost: %s %v", actor, err)
	}
	if err := database.conn.QueryRow("SELECT disposition FROM cdrs WHERE id=42").Scan(&disposition); err != nil || disposition != "voicemail" {
		t.Fatalf("Historical call lost: %s %v", disposition, err)
	}
	for _, state := range []string{"queued", "initiated", "ringing", "in-progress", "completed", "no-answer", "canceled"} {
		if _, err := database.conn.Exec("INSERT INTO cdrs(call_sid,direction,from_number,to_number,started_at,disposition) VALUES(?,'outbound','+441234567890','+441234567891',CURRENT_TIMESTAMP,?)", state, state); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := database.conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("Migration left broken foreign keys")
	}
}
