# Business phone system

The deployed system uses Asterisk for TLS SIP and SRTP audio, GoSIP for the
administrator console and SMS bridge, and Twilio for telephone calls and SMS.
Each SIP user has one registered phone and one assigned Twilio number.

## Add or change a phone user

1. Open **SIP Users**, then **Add SIP user**.
2. Enter a name and a unique SIP username; choose an existing Twilio number
   with voice and SMS capability. Active users cannot share a number.
3. Save. The server adds a SIP credential to its dedicated Twilio credential
   list, saves the separate outbound credential in the private PBX volume,
   updates the selected number's webhooks and its default incoming route,
   and assigns its outgoing caller ID and SMS sender.
4. Download the phone setup file before closing the credential panel.
   The generated phone password is shown once and stored only as a SIP hash.
5. In Linphone, use the generated SIP identity and set **both** Registrar URI
   and Outbound SIP Proxy URI to `sip:sip.leadomi.com:5061;transport=tls`.
   Enable SRTP and disable CPIM in basic conversations.

Configuration applies within 10 seconds. Calls use full international `+numbers`;
SMS recipients use `sip:+number@sip.leadomi.com`.

**Edit / reset password** can reassign a number or generate a new phone password.
Changing a number replaces its default routing rules. **Disable** blocks new
calls and messages, removes the phone endpoint and revokes its Twilio credential.
A call already in progress can continue. History is kept. A disabled user's
number can be assigned to a replacement user; it is not released from Twilio.

Failed provisioning is marked **error**, and that account fails closed until
**Configure / retry** succeeds. External API writes and local database writes
cannot share a transaction. A retry checks for an existing Twilio username and
updates it rather than creating duplicates. If a newly created account failed
before showing its password, retry generates a fresh phone password.

Numbers in a shared Twilio Messaging Service require number-level inbound
webhooks to be enabled already; provisioning will not change a shared service's
policy automatically. Number purchase and porting remain in the Twilio console.

## Access and usage controls

Phone users do not need a web console account. **Administrators** controls web
console access, which includes all employees' message and call history. Only
trusted administrators should receive that access. Account password changes
revoke existing sessions immediately. New administrator passwords require
12–72 characters. The Twilio Account SID and Auth Token remain in Coolify's
environment; they are not sent to phones or stored in SQLite.

In **Settings**, restrict outgoing calls and SMS to the international prefixes
your business needs, for example `+44, +91`. An empty list allows all destinations.
SMS defaults to 20 submissions per minute per sending number, and each SIP user
is limited to two concurrent outgoing calls. These limits are not a monetary
spending cap. Configure Twilio usage alerts and account spending controls in
Twilio separately.

Removed console options include the old manual device wizard, SIP trunk editor,
WebRTC/browser calling and call-control UI that do not control this Asterisk
deployment. Call routing, call/SMS history and voicemail remain available.

## Backups and recovery

`gosip-backup.timer` runs the root-only `scripts/business-backup.py` daily at
02:15 UTC (07:45 IST), with up to five minutes of scheduling jitter. Archives are
kept for 30 days under `/var/backups/gosip` (directory 0700, files 0600). Each
contains online SQLite snapshots of `gosip.db` and `delivery.db` plus the PBX SIP
credential map, including credentials originally supplied through environment.
SQLite integrity is checked before each archive is accepted. Dashboard shows
the last successful daily backup. On-demand database backups are separate and
can be created and verified in the dashboard.

For full recovery, retain the Coolify application configuration and its environment
separately, along with access to the DNS account. The backup archive does not
contain the Twilio API Auth Token or incoming-trunk secret. Export archives to
an access-controlled off-server location: copies on the same VPS do not protect
against loss of that VPS. No off-server destination has been configured.

To validate a restore, extract only the three expected archive members into a
private temporary directory, run `PRAGMA integrity_check` on both databases and
validate the credential JSON. For an actual restore, stop both application
containers, preserve current volumes, restore `gosip.db` to the GoSIP volume and
`delivery.db` and `credentials.json` to the PBX volume, all with mode 0600. Recreate
containers from the matching Git revision and Coolify environment, renew/export
TLS certificates, and verify webhooks and phone registrations. Twilio is external:
compare its current number webhooks and credential list with the restored state
before enabling calls. Do not restore a database into a running application.

This is one VPS with persistent local volumes, not a redundant phone service.
Android background restrictions can delay ringing or messages while Linphone is
closed; keep it connected and allow background activity. Offline SMS is queued;
delivery acceptance is tracked, but read receipts/typing notifications are not
supported. Calls and messages should be tested manually after a deployment.
