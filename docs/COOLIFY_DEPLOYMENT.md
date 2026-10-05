# GoSIP deployment report

Date: 2026-10-05 (Asia/Calcutta)

## Repository structure

Inspected upstream main at 533cc6704720f6c9e2ec4d94a31f1b2af1598423.
The root has Dockerfile, go.mod, go.sum, Makefile, .env.example, README.md,
docker-compose.yml and docker-compose.dev.yml. cmd/gosip/main.go is the Go
entry point. frontend/package.json and frontend/pnpm-lock.yaml define the Vue
frontend, whose output is frontend/dist. SQL migrations are under
internal/db/migrations and embedded in the binary by internal/db/db.go.
There is no root migrations directory and originally no .dockerignore.
SQLite uses CGO through mattn/go-sqlite3; no database server is required.

Maintained repository: https://github.com/rohitkumar-co-in/gosip, branch main.
Local origin points there; upstream retains https://github.com/btafoya/gosip.

## Confirmed root cause

The old Coolify application gq2rsvuy5pxcyeg8veyrvmp3 used an inline Dockerfile
without checking out GoSIP. Its repository field was coollabsio/coolify.
Docker therefore received a tiny context without go.sum or frontend. Setting
Base Directory to / alone cannot supply source files in that deployment mode.
Coolify documentation: https://coolify.io/docs/applications/builds/dockerfile

## Exact build changes

Dockerfile:
- golang:1.21-alpine -> golang:1.23-alpine, matching go.mod and CI.
- Replace mutable Corepack pnpm@latest with npm install --global pnpm@10.11.0.
  CI uses pnpm major 10; this version passed the frozen lockfile install.
- Require frontend/pnpm-lock.yaml explicitly rather than a wildcard.
- Remove go build -a and add GOMAXPROCS=1 plus -p 1 to limit compiler concurrency.
- Remove COPY /app/migrations: migrations are embedded from internal/db/migrations.
- Retain Node 20, CGO/static linking, Alpine runtime dependencies, UID/GID 1000,
  frontend output copy, HTTP health check, exposed ports and /app/data volume.

Added .dockerignore excludes Git, node_modules, frontend/dist, .vite,
tsbuildinfo, runtime data/media, local environment files, private keys, databases,
logs, binaries and IDE files. It retains go.mod, go.sum, frontend source and
lockfile, cmd and embedded migrations. The exact patterns are in /.dockerignore.

The initial production frontend build also failed TypeScript checking. Fixed
unused imports/variables, added Vite ambient declarations, used device username
as an extension fallback, exposed browser origin through a script variable, and
removed access to an unsupported authenticated-user name property. Strict
checking remains enabled; package.json and the lockfile were not changed.

Added docker-compose.coolify.yml for Coolify. It builds Dockerfile from context
., exposes HTTP internally and publishes TCP/UDP 5060. Its external named-volume
declaration is rewritten by Coolify's parser; see actual storage below.
Standard ports_mappings in this Coolify version only accepts TCP,
and its custom Docker option parser ignores --publish, so Compose is necessary
for both SIP protocols without changing the proxy.

## Actual Coolify configuration

Project: 2a0vhfdrsslmii9lwvthmpr8
Environment: 3q7cgyfldbfvkrg1ybfsb966
Application: GoSIP, zgzcndvsggqprdlnm503jmpu
Server: localhost, k1wigqmvzpvcps8cryssrncl
Current EC2 public IP: 18.134.241.218 (the supplied 18.171.181.41 timed out)
Coolify: 4.3.23; Docker: 29.8.0

| Setting | Value |
| --- | --- |
| Repository | https://github.com/rohitkumar-co-in/gosip |
| Branch | main |
| Build pack | Docker Compose from Git |
| Base directory | / |
| Compose location | /docker-compose.coolify.yml |
| Docker build context | . (repository root) |
| Dockerfile | Dockerfile at repository root |
| Service | gosip |
| Public URL | https://gosip.18.134.241.218.sslip.io |
| Compose domain | https://gosip.18.134.241.218.sslip.io:8080 (target port) |
| HTTP | Internal 8080 through existing Traefik; no host 8080 mapping |
| SIP | Host 5060 to container 5060, TCP and UDP |
| Health check | wget /api/health, 30s interval, 10s timeout, 3 retries, 30s start period |
| Automatic deploy | Disabled; deploy main through Coolify after updates |

The root Dockerfile can still be built independently. Original Compose files
remain unchanged. The failed original Coolify resource was retained.

## Environment and persistent storage

Configured runtime values:

GOSIP_DATA_DIR=/app/data
GOSIP_HTTP_PORT=8080
GOSIP_SIP_PORT=5060
GOSIP_CORS_ORIGINS=https://gosip.18.134.241.218.sslip.io
GOSIP_VOLUME_NAME=zgzcndvsggqprdlnm503jmpu-gosip-data

Compose defaults TZ to America/New_York. Twilio credentials and administrator
setup are completed in the web setup wizard; no credentials were committed.

The active Compose production volume is zgzcndvsggqprdlnm503jmpu_gosip-data
(underscore), mounted at /app/data. Coolify's parser rewrites the supplied
external-volume reference and creates this deterministic application volume.
The initial Dockerfile deployment volume zgzcndvsggqprdlnm503jmpu-gosip-data
(hyphen) remains intact. Both databases were inspected: zero users/devices,
messages, DIDs and trunks. No existing databases or volumes were deleted.
Do not assume the external-volume environment variable controls the active
mount; inspect the generated Compose and actual Docker mounts before changes.

GOSIP_DB_PATH and GOSIP_EXTERNAL_IP occur in upstream examples but are not read
by this application. The actual database path derives from GOSIP_DATA_DIR.
TLS/SRTP configuration is loaded but not passed to the SIP server by main.go;
setting those environment variables alone does not activate encrypted listeners.
No RTP port range was invented; the inspected source exposes no configurable
RTP listener range. End-to-end Twilio calls, audio and encrypted SIP have not
been verified and require separate functional validation after setup.

## Commands and validation

Local:

    git clone https://github.com/btafoya/gosip .
    cd frontend
    npx.cmd --yes pnpm@10.11.0 install --frozen-lockfile
    npx.cmd --yes pnpm@10.11.0 build
    git diff --check
    git push -u origin main

Server (SSH ubuntu@18.134.241.218, existing PEM; sudo for Docker):

    cd /home/ubuntu/gosip-deploy-20261005
    sudo docker build --progress=plain -t gosip-tested:20261005 .
    python3 scripts/deployment-smoke.py --http http://127.0.0.1:18080 --sip-port 15060
    sudo docker exec gosip-smoke-20261005 sqlite3 /app/data/gosip.db 'PRAGMA integrity_check; SELECT COUNT(*) FROM schema_migrations;'

The frozen frontend install/build passed. The manual full Docker build passed;
Go compilation took about 96 seconds. A dedicated test container passed HTTP
health, frontend HTML and assets, TCP/UDP SIP OPTIONS (200 OK), SQLite integrity,
all nine migrations and UID/GID 1000 verification. Database and marker persisted
across container recreation with the same test volume. Test containers are now
stopped; their volume and source/build logs remain available.

Current server has 3.7GiB RAM and 2GiB persistent swap. Builds completed with
available memory; swap was retained. Coolify, Traefik, n8n and their databases
remained running and healthy. No destructive cleanup or infrastructure reinstall
was performed.

## Final verification

Compose deployment aye5zrntay4pysnzw9w4krv2 finished successfully at
2026-10-05 23:50:45 IST, using commit 961b52a2b05f6283f2a26496d84c9bbea6b91ac6.
Container gosip-zgzcndvsggqprdlnm503jmpu-181837172653 is healthy and explicitly
publishes both TCP and UDP 5060. Public HTTPS health, HTML and frontend assets
passed. Server-local TCP/UDP OPTIONS passed; an external UDP OPTIONS probe from
Windows also received SIP/2.0 200 OK. Active SQLite integrity and nine migrations
passed. Setup is incomplete. Twilio calls/SMS are not configured or validated.

## Existing-installation inspection requested after deployment

Deployment changes were paused on the user's new request to inspect the existing
business telephony installation. Active database contains zero users, devices,
DIDs, trunks, routes, registrations and messages. No TWILIO_ACCOUNT_SID or
TWILIO_AUTH_TOKEN environment variables are set; corresponding database config
keys are absent. The other preserved production volume is also unconfigured.
This does not match the user's description of multiple existing SIP users;
confirm the authoritative installation URL/server before modifying data/config.

Source blockers:
- pkg/sip/handlers.go returns 501 for authenticated outbound INVITEs.
- Inbound SIP ring routes send 302 redirects to registered contacts, rather than
  bridging SIP dialogs/media; NAT and bidirectional audio are not established.
- HTTP inbound ring TwiML targets device usernames at sip.gosip.local.
- SIPTrunkHandler exists but is not registered in the HTTP router.
- No active RTP listener/relay port allocation was found in the current startup.
- Web SMS APIs, Twilio REST sending, inbound SMS/status webhooks and SQLite
  message history exist, but there are no configured credentials or DIDs.
- Webhook signature validation reads the database twilio_auth_token key, so
  environment-only credentials are insufficient for its current implementation.

No application/container/database configuration was changed during this
inspection. Real calls, registration with user credentials, DTMF, audio and real
SMS tests remain pending identification/configuration of the correct installation.

## Credential-storage change

On the user's request, GoSIP now supports Twilio credentials supplied through
Coolify runtime variables without storing them in SQLite. The Twilio client,
setup wizard and webhook validation all use the same configured credentials.
Runtime variables take priority and cannot be overwritten through the web UI.
The wizard detects preconfigured credentials and skips re-entry.

Without runtime variables, UI credentials are saved atomically to /app/data/.env
with permissions 0600, and reloaded on restart. Existing Twilio database rows
are migrated only after their replacement credentials are available; a failed
file write preserves the old rows. No unrelated configuration secrets moved.
See TWILIO_CREDENTIALS.md for precedence, backup and security details.

The production SQLite database was backed up consistently to
/app/data/backups/pre-twilio-env-20261006.db with mode 0600; integrity_check passed.
Go tests passed for internal/config, internal/db, internal/api and internal/twilio,
including credential persistence, failed migration retention, token rotation,
webhook signatures and runtime credentials avoiding both SQLite and file copies.
Frontend production type checking and build also passed.
