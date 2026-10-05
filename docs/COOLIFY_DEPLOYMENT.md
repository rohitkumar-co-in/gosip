# GoSIP deployment investigation

Inspected upstream `main` at `533cc6704720f6c9e2ec4d94a31f1b2af1598423` on 2026-10-05.

## Repository and failure findings

The root contains Dockerfile, go.mod (Go 1.23.0), go.sum, docker-compose.yml,
docker-compose.dev.yml, .env.example, Makefile, README.md, cmd/gosip/main.go,
frontend/package.json, and frontend/pnpm-lock.yaml (lockfile version 9).
There was no .dockerignore. SQL migrations live in internal/db/migrations and
are embedded by internal/db/db.go; there is no root migrations directory.
The Vue frontend builds to frontend/dist, served by the Go HTTP router.
SQLite uses CGO through github.com/mattn/go-sqlite3; no external database server
is required.

The reported missing frontend and go.sum files exist in this checkout. A 5.35kB
context does not match this source tree. The actual Coolify resource has not yet
been inspected, so a wrong base directory, different source/ref, or Dockerfile
resource without Git remains a hypothesis. Coolify documents that its
Dockerfile-without-Git resource has no repository context:
https://coolify.io/docs/applications/builds/dockerfile

## Local changes

- Dockerfile: Go 1.21 -> 1.23, matching go.mod and CI.
- Install pinned pnpm 10.11.0 using npm instead of Corepack's mutable latest.
  CI selects pnpm major 10; the frozen lockfile install passed with this version.
- Require frontend/pnpm-lock.yaml explicitly in COPY.
- Remove forced Go rebuild (-a); compile with GOMAXPROCS=1 and -p 1 to reduce
  concurrent compiler work on the small server. Keep CGO and static linking.
- Remove COPY of nonexistent /app/migrations; migrations are embedded.
- Keep Node 20, runtime dependencies, non-root UID/GID 1000, health check,
  frontend asset copy, ports, and persistent /app/data volume.
- Add .dockerignore excluding Git, dependencies, built assets, generated build
  metadata, runtime databases/media, local environment files, keys and logs.
  Go modules, frontend source/lockfile, cmd, and embedded migrations remain included.
- Fix existing frontend type-check failures: remove unused imports/variables,
  add Vite ambient types, use device username as an extension fallback, expose
  browser origin through a script variable, and remove access to a nonexistent
  authenticated-user name property. Type checking remains enabled.

## Intended Coolify settings (not applied yet)

| Setting | Value |
| --- | --- |
| Source | Git repository https://github.com/btafoya/gosip |
| Branch | main, with the local fixes made available in the deployed source |
| Build pack | Dockerfile from Git |
| Base directory / build context | / (repository root) |
| Dockerfile location | /Dockerfile |
| Build stage | final stage; no builder-stage override |
| HTTP internal port | 8080 |
| Domain | Existing GoSIP domain, to be discovered; do not use the Coolify dashboard domain |
| SIP published ports | Host 5060 -> container 5060, separately for TCP and UDP |
| Health check | HTTP GET /api/health on port 8080; image includes wget |
| Persistent storage | Dedicated GoSIP volume mounted at /app/data |

SIP needs direct protocol-aware port publishing, not ordinary HTTP routing
through Traefik. Inspect the installed Coolify version's support for UDP before
choosing its port controls or Docker Compose. Verify port availability and AWS
security-group/firewall rules before applying mappings. Do not invent RTP ranges:
the inspected source exposes no configurable RTP listener range.

## Runtime environment and storage

Set GOSIP_DATA_DIR=/app/data, GOSIP_HTTP_PORT=8080 and GOSIP_SIP_PORT=5060.
Set GOSIP_CORS_ORIGINS to the actual public application origin. TZ can follow
the user's preference. Optional Twilio credentials are loaded from environment
or configured through the setup UI and persisted in SQLite.

Important discrepancy: GOSIP_DB_PATH and GOSIP_EXTERNAL_IP appear in deployment
examples but are not read by the current application code. DBPath() derives
/app/data/gosip.db from GOSIP_DATA_DIR. TLS/SRTP settings are loaded by config,
but main.go does not pass them into the SIP server, so enabling those environment
variables alone does not activate encrypted listeners. No TLS/SRTP functionality
has been claimed as working.

The persistent volume must be writable by UID/GID 1000. It includes gosip.db,
recordings, voicemails, backups, certs and moh. A fresh named volume normally
inherits the image directory contents/ownership; inspect actual ownership.
Do not replace or reinitialize any existing data volume.

## Validation and remaining work

Commands completed locally:

```powershell
git clone https://github.com/btafoya/gosip .
# From frontend:
npx.cmd --yes pnpm@10.11.0 install --frozen-lockfile
npx.cmd --yes pnpm@10.11.0 build
# From repository root:
git diff --check
```

Frozen dependency install passed without changing the lockfile. Production
frontend type checking and Vite build passed after the fixes. Docker and Go are
not installed in the local shell; no full image/backend build has been performed.
The PEM file exists, but two SSH attempts to ubuntu@18.171.181.41:22 timed out.
https://app.leadomi.com is reachable and redirects unauthenticated requests to login.

Once server access is restored, inspect Coolify's actual source/config and logs,
then build from the patched repository root:

```sh
docker build --progress=plain -t gosip-test .
```

Use a dedicated temporary test container and volume; verify startup migrations,
SQLite integrity, HTTP/frontend assets, actual TCP/UDP SIP responses, volume
writability and persistence across container recreation. /api/health itself does
not query SQLite or confirm that SIP sockets bound, so it is insufficient alone.
Then apply the derived Coolify settings, deploy and verify the public application.
Monitor memory while retaining the existing 2GiB swap and all unrelated services.

## Server validation update

The current EC2 address is 18.134.241.218; SSH with the existing PEM works there.
It has 3.7GiB RAM and 2GiB swap. The full patched Docker build passed from
/home/ubuntu/gosip-deploy-20261005, producing gosip-tested:20261005. The Go
compilation stage took approximately 96 seconds with concurrency limited to one.

A dedicated test container passed HTTP health, frontend HTML and asset requests,
SIP OPTIONS over TCP and UDP, SQLite integrity_check, all nine migrations, and
UID/GID 1000 verification. Data and a marker survived container recreation with
the same dedicated test volume. Production Coolify deployment remains pending.

Coolify API inspection confirms the original application
gq2rsvuy5pxcyeg8veyrvmp3 uses an inline Dockerfile, no repository checkout,
and git_repository=coollabsio/coolify. This explains the tiny build context and
missing GoSIP source files. Its root base directory alone cannot fix that mode.

The maintained repository will be https://github.com/rohitkumar-co-in/gosip,
branch main. Existing Coolify/n8n services, databases and volumes remain intact.
