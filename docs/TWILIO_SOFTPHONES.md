# Twilio mobile softphones

GoSIP can route voice through a Twilio Programmable Voice SIP Domain. The
softphone registers directly with Twilio, which handles SIP signaling and audio.
GoSIP supplies signed call instructions, records calls, and handles SMS through
Twilio's Messaging API. This avoids the unfinished local GoSIP outbound SIP path.

Set these runtime variables in Coolify and redeploy:

- `TWILIO_SIP_DOMAIN`: the base domain, such as `example.sip.twilio.com`.
- `GOSIP_OUTBOUND_CALLER_ID`: a voice-enabled Twilio number present in GoSIP DIDs.
- `GOSIP_PUBLIC_URL`: the canonical HTTPS URL used by all Twilio webhooks.
- `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN`: the existing runtime credentials.

Enable SIP registration and Secure Media on the Twilio SIP Domain. Map a
credential list to both Calls and Registrations. Its Voice URL is
`/api/webhooks/voice/outgoing` on the canonical GoSIP URL; its voice status callback
is `/api/webhooks/voice/status`. GoSIP accepts only signed requests from the
configured account, known device usernames, and international E.164 destinations.
Twilio's existing geographic dialing permissions still apply.

The phone number's Voice URL is `/api/webhooks/voice/incoming`; its SMS URL is
`/api/webhooks/sms/incoming`, and its voice status callback is
`/api/webhooks/voice/status`. Use POST. Clear an obsolete Voice Application SID
when switching the number to URL-based routing. Preserve prior settings first.
Add a GoSIP device matching the SIP credential username and a default ring route
for the DID. Ringing uses TLS and SRTP, then voicemail after a busy/unanswered call.

On Android, add a third-party SIP account in Linphone. Use the exact credential
username/password, an edge-specific Twilio registration domain, TLS on port 5061,
and SRTP media encryption. Do not use DTLS or ZRTP for this Twilio connection.
Use G.711 PCMU/PCMA and RFC 2833/4733 DTMF. The base domain is used by GoSIP to
dial the registered endpoint regardless of which edge the phone registers with.

GoSIP shows these devices as "Managed by Twilio"; its local registration list
does not report Twilio registrations. Additional devices require credentials in
Twilio as well as the matching GoSIP record. Updating a local GoSIP device
password does not rotate the Twilio password.

Use GoSIP's Messages page for SMS. Softphone SIP chat is not connected to this
Messaging API integration. Keep the app registered while manually testing
incoming/outgoing calls and two-way audio. Background Android ringing depends on
the app remaining connected; test with the screen locked as well. This setup does
not configure emergency calling, conferences, or Twilio push notification delivery.

References:

- https://www.twilio.com/docs/voice/api/sip-registration
- https://www.twilio.com/docs/voice/api/secure-media
- https://www.linphone.org/en/docs/login-sip-account/
