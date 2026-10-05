# VPS calls and SMS with Asterisk

The Android phone registers to the VPS over SIP TLS (5061). Asterisk handles
registration, authenticated calling, SRTP audio and basic SIP SIMPLE chat.
GoSIP handles signed Twilio webhooks, Twilio's SMS API and persistent history.
Audio passes through the VPS (`direct_media=no`).

## Android account

Use Linphone's third-party SIP account mode with the GoSIP device username and
password, VPS hostname, TLS port 5061 and SRTP encryption (SDES). Keep certificate
verification on. Use G.711 PCMU/PCMA and RFC 2833/4733 DTMF, rather than DTLS/ZRTP.
For one-to-one text messages, use a recipient such as
`sip:+441234567890@sip.example.com`. Dial international `+` numbers too.

Advanced Linphone group chat/encryption requires Flexisip and is outside this
bridge. Zoiper editions vary in messaging support; Linphone is the supported
test target. This does not add carrier SMS to Android's native Messages app.
Allow microphone, notifications and background operation. Exclude Linphone
from battery optimization and test with the screen locked. There is no mobile
push service here to wake an app that has disconnected.

## Coolify runtime variables

The Compose file adds `pbx`, sharing GoSIP's existing volume read-only. Its
`pbx-data` volume persists the delivery queue. Preserve both active volumes.

- `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`: GoSIP only, never stored in its DB.
- `TWILIO_SIP_DOMAIN`: dedicated Programmable Voice domain with Secure Media.
- `PBX_TWILIO_USER`, `PBX_TWILIO_PASSWORD`: outbound SIP trunk credential; its
  username must match a GoSIP device for the outgoing webhook.
- `GOSIP_OUTBOUND_CALLER_ID`: SMS/voice-enabled Twilio DID present in GoSIP.
- `GOSIP_PUBLIC_URL`: canonical HTTPS URL for signed webhook validation.
- `PBX_DOMAIN`, `PBX_PUBLIC_IP`: public VPS hostname and IPv4 address.
- `GOSIP_PBX_SECRET`: random bridge API secret of at least 32 characters.
- `GOSIP_PBX_TRUNK_USER`, `GOSIP_PBX_TRUNK_PASSWORD`: separate inbound Twilio
  digest credential, not a phone password.
- `GOSIP_PBX_URL`: `http://pbx:8088`, enables PBX registration status in GoSIP.
- `GOSIP_PBX_SIP_DOMAIN`: `sip.example.com:5061`, switches incoming calls to PBX.
  Leave empty while testing to retain the prior direct-Twilio call route.

Keep secrets out of source and Docker build arguments. GoSIP device/password
changes synchronize within 10 seconds, using the existing `gosip` realm HA1.
Allow inbound TCP 5061 and UDP 10000–10100 in the AWS security group, plus
outbound Twilio TLS and UDP media. HTTPS 443 serves webhooks. AMI 5038 and
FastAGI 4573 bind to container loopback; bridge HTTP 8088 is internal and requires
its secret. No anonymous outbound calls are configured. Twilio incoming calls
require digest authentication and can only ring local device names.

`/data/gosip-pbx/certs` contains only this domain's certificate and key, with
root-only permissions. Export from Traefik's ACME storage with
`scripts/export-pbx-certificate.py DOMAIN ACME-JSON TARGET-DIR`. Install an hourly
systemd timer to refresh them. The bridge reloads TLS when the certificate
changes. The exporter never changes the Coolify proxy or its ACME storage.

## Twilio callbacks and routing

Use POST callbacks on the canonical GoSIP URL:

- Number Voice URL: `/api/webhooks/voice/incoming`.
- Number SMS URL: `/api/webhooks/sms/incoming`.
- Number/domain voice status: `/api/webhooks/voice/status`.
- SIP domain Voice URL: `/api/webhooks/voice/outgoing`.

Clear stale Voice/SMS Application SID overrides when using URL callbacks.
Enable Secure Media and map the outbound credential list to Calls. Outbound
voice requires signed requests from the configured account, a known username
and international destinations. Twilio geographic dialing permissions apply.
GoSIP ring routes select devices; unanswered calls fall back to Twilio voicemail.

Incoming SMS is stored in GoSIP and queued for devices in the DID's enabled
ring routes. Offline messages persist until a contact registers. Delivery is
handed to Asterisk's SIP transport without end-to-end read receipts. Initial
installation does not replay the old inbox. A failed outgoing Twilio SMS sends
a failure notification to the connected phone; GoSIP retains detailed status.

## Verification

`scripts/test-pbx-registration.py HOST PRIVATE-STATE.json` verifies public TLS
and digest credentials with expiry zero, without calls or messages.
`scripts/pbx-integration-test.py` runs an isolated PBX and dummy backend on the
VPS to test both SMS directions, Unicode/newlines and offline delivery without
Twilio API calls. Its Docker paths target the deployment host; it requires
the staging image, certificate directory and bridge script already prepared.

Manually test incoming/outgoing calls, two-way audio, DTMF, incoming/outgoing SMS,
reply to a received message, and operation with the screen locked. Registration
alone does not prove audio. GoSIP's original SIP call-control API is not connected
to Asterisk; native softphone call controls remain available.

References: [Linphone features](https://www.linphone.org/en/features/),
[Twilio secure media](https://www.twilio.com/docs/voice/api/secure-media),
[Twilio SIP IPs](https://www.twilio.com/docs/sip-trunking/ip-addresses),
[Asterisk configuration](https://docs.asterisk.org/Asterisk_20_Documentation/API_Documentation/Module_Configuration/res_pjsip/).
