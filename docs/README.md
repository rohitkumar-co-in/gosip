# Leadomi SIP documentation

These documents describe the supported Asterisk + Twilio business deployment.
They do not advertise unfinished features retained in legacy source files.

| Guide | Purpose |
|---|---|
| [Installation](INSTALLATION.md) | Fresh VPS, Twilio SIP domain, first admin |
| [Configuration](CONFIGURATION.md) | Environment, metadata, certificates, network |
| [Coolify](COOLIFY_DEPLOYMENT.md) | Deploy and update safely |
| [Administration](ADMINISTRATION.md) | SIP users, owned number assignment, web admins |
| [User guide](USER_GUIDE.md) | Console, phones, history, troubleshooting |
| [Softphones](TWILIO_SOFTPHONES.md) | Android Linphone setup and manual tests |
| [Credentials](TWILIO_CREDENTIALS.md) | Where secrets live and how rotation works |
| [Operations](BUSINESS_OPERATIONS.md) | Health, limits, incident recovery |
| [Backup](BACKUP.md) | Daily backups, restore, off-server copies |
| [API](API.md) | Supported business endpoints and request fields |
| [SIP basics](tutorials/SIP_BASICS.md) | Voice versus SMS and TLS/SRTP |

Read-only inventory never configures numbers. Assignments require a reviewed
connection snapshot and explicit administrator confirmation. Keep existing
working users, passwords, volumes, and provider settings during upgrades.
