# Twilio credentials

In Coolify, configure both TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN as runtime
variables on the GoSIP application, then redeploy. Do not enable them as build
arguments. GoSIP uses these values for both its Twilio API client and webhook
signature validation. The setup wizard detects configured credentials and lets
you create the administrator account without entering them again.

Explicit runtime credentials take precedence over file credentials. The web UI
cannot change runtime-managed credentials; rotate them in Coolify and redeploy.
They are not copied into SQLite or a credential file. Coolify and Docker
administrators can still access runtime environment variables.

If runtime credentials are absent, the setup/settings form saves Twilio values
to GOSIP_DATA_DIR/.env, which is /app/data/.env in Docker. The file has mode 0600,
is replaced atomically, and persists on the application data volume. It contains
only TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN. No shell evaluation or variable
expansion occurs when reading it. A failed save retains the active credentials.

Startup loads existing process/file credentials and migrates legacy Twilio
database configuration into the file if needed. Legacy rows are removed only
after the new credentials are available. Existing process/file values take
precedence over legacy database values. Migration failure prevents startup and
leaves legacy values intact. Old database backups may still contain old tokens;
this change does not delete backups or promise forensic erasure of SQLite pages.

The token is never returned by the settings or setup-status API. A database-only
backup no longer includes newly configured Twilio secrets. Securely back up the
credential file separately if using it, or maintain runtime values in Coolify.
Both environment variables and .env files contain plaintext secrets accessible
to sufficiently privileged server users; neither provides encryption at rest.

Credential configuration alone does not implement the repository's unfinished
outbound SIP call routing or establish two-way audio.
