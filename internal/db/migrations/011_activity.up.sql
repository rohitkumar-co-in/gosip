CREATE TABLE activity_actors (
 kind TEXT NOT NULL, record_id INTEGER NOT NULL, actor TEXT NOT NULL,
 PRIMARY KEY(kind,record_id)
);
INSERT INTO activity_actors SELECT 'call',id,'Historical record: actor not captured' FROM cdrs;
INSERT INTO activity_actors SELECT 'sms',id,'Historical record: actor not captured' FROM messages;
CREATE TRIGGER activity_call_actor AFTER INSERT ON cdrs BEGIN
 INSERT INTO activity_actors VALUES('call',NEW.id,COALESCE(
 (SELECT username FROM devices WHERE id=NEW.device_id),
 (SELECT 'Assigned to '||d.username FROM devices d JOIN sip_accounts a ON a.device_id=d.id
 WHERE a.number=CASE WHEN NEW.direction='inbound' THEN NEW.to_number ELSE NEW.from_number END AND a.enabled=1),
 'System / unknown'));
END;
CREATE TRIGGER activity_sms_actor AFTER INSERT ON messages BEGIN
 INSERT INTO activity_actors VALUES('sms',NEW.id,COALESCE(
 (SELECT 'Assigned to '||d.username FROM devices d JOIN sip_accounts a ON a.device_id=d.id
 WHERE a.number=CASE WHEN NEW.direction='inbound' THEN NEW.to_number ELSE NEW.from_number END AND a.enabled=1),
 'System / unknown'));
END;
CREATE TRIGGER activity_call_delete AFTER DELETE ON cdrs BEGIN
 DELETE FROM activity_actors WHERE kind='call' AND record_id=OLD.id;
END;
CREATE TRIGGER activity_sms_delete AFTER DELETE ON messages BEGIN
 DELETE FROM activity_actors WHERE kind='sms' AND record_id=OLD.id;
END;
