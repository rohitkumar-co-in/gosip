-- Accept Twilio lifecycle states as well as legacy call dispositions.
-- Rebuilding the table must preserve voicemail links and actor snapshots.
CREATE TEMP TABLE voice_history_links AS
SELECT id, cdr_id FROM voicemails WHERE cdr_id IS NOT NULL;
DROP TRIGGER activity_call_actor;
DROP TRIGGER activity_call_delete;

CREATE TABLE cdrs_new (
    id INTEGER PRIMARY KEY,
    call_sid TEXT UNIQUE,
    direction TEXT CHECK(direction IN ('inbound', 'outbound')),
    from_number TEXT NOT NULL,
    to_number TEXT NOT NULL,
    did_id INTEGER REFERENCES dids(id) ON DELETE SET NULL,
    device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
    started_at DATETIME NOT NULL,
    answered_at DATETIME,
    ended_at DATETIME,
    duration INTEGER DEFAULT 0,
    disposition TEXT CHECK(disposition IN ('answered', 'voicemail', 'missed', 'blocked', 'busy', 'failed', 'queued', 'initiated', 'ringing', 'in-progress', 'completed', 'no-answer', 'canceled')),
    recording_url TEXT,
    spam_score REAL
);
INSERT INTO cdrs_new SELECT * FROM cdrs;
DROP TABLE cdrs;
ALTER TABLE cdrs_new RENAME TO cdrs;
UPDATE voicemails SET cdr_id=(SELECT cdr_id FROM voice_history_links WHERE id=voicemails.id)
WHERE id IN (SELECT id FROM voice_history_links);
DROP TABLE voice_history_links;

CREATE INDEX idx_cdrs_started ON cdrs(started_at DESC);
CREATE INDEX idx_cdrs_disposition ON cdrs(disposition);
CREATE INDEX idx_cdrs_did ON cdrs(did_id);
CREATE TRIGGER activity_call_actor AFTER INSERT ON cdrs BEGIN
 INSERT INTO activity_actors VALUES('call',NEW.id,COALESCE(
 (SELECT username FROM devices WHERE id=NEW.device_id),
 (SELECT 'Assigned to '||d.username FROM devices d JOIN sip_accounts a ON a.device_id=d.id
 WHERE a.number=CASE WHEN NEW.direction='inbound' THEN NEW.to_number ELSE NEW.from_number END AND a.enabled=1),
 'System / unknown'));
END;
CREATE TRIGGER activity_call_delete AFTER DELETE ON cdrs BEGIN
 DELETE FROM activity_actors WHERE kind='call' AND record_id=OLD.id;
END;
