CREATE TABLE business_audit (
    id INTEGER PRIMARY KEY,
    actor_id INTEGER,
    action TEXT NOT NULL,
    subject TEXT NOT NULL,
    outcome TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE sip_accounts (
    device_id INTEGER PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    number TEXT NOT NULL DEFAULT '',
    credential_sid TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    state TEXT NOT NULL DEFAULT 'pending',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX sip_accounts_number ON sip_accounts(number) WHERE number <> '';
