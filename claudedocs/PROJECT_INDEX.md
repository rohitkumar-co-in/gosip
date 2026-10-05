# Leadomi SIP project index

| Location | Responsibility |
|---|---|
| `cmd/gosip` | Application startup |
| `internal/api` | Auth, signed webhooks, business workflows/history |
| `internal/db/migrations` | Embedded SQLite schema migrations |
| `internal/config` | Stable runtime environment handling |
| `pkg/sip` | Compatibility SIP utilities and legacy standalone implementation |
| `deploy/asterisk` | Production SIP/audio and MESSAGE bridge |
| `frontend/src/views` | Leadomi administrator console |
| `scripts/business-backup.py` | Root-only daily online backups |
| `docker-compose.coolify.yml` | Existing Coolify deployment |
| `docker-compose.production.yml` | Fresh direct Compose deployment |
| `docs` | Current installation, configuration, user/API/operations guides |

Start with [README](../README.md) or the [documentation index](../docs/README.md).
