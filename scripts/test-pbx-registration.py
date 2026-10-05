#!/usr/bin/env python3
"""Verify TLS and SIP digest with Expires: 0. Makes no calls or messages."""
import hashlib
import json
import re
import secrets
import socket
import ssl
import sys

host, state_path = sys.argv[1:]
state = json.load(open(state_path))
user, password = state["username"], state["password"]
uri = "sip:" + host
md5 = lambda text: hashlib.md5(text.encode()).hexdigest()
with ssl.create_default_context().wrap_socket(socket.create_connection((host,5061),timeout=12),server_hostname=host) as sock:
    sock.settimeout(12)
    callid, tag = secrets.token_hex(12), secrets.token_hex(8)
    def exchange(sequence, authorization=""):
        request=(f"REGISTER {uri} SIP/2.0\r\nVia: SIP/2.0/TLS 127.0.0.1:{sock.getsockname()[1]};branch=z9hG4bK{secrets.token_hex(8)};rport\r\n"
          f"Max-Forwards: 70\r\nFrom: <sip:{user}@{host}>;tag={tag}\r\nTo: <sip:{user}@{host}>\r\nCall-ID: {callid}\r\nCSeq: {sequence} REGISTER\r\n"
          f"Contact: <sip:{user}@127.0.0.1:{sock.getsockname()[1]};transport=tls>;expires=0\r\nExpires: 0\r\n{authorization}Content-Length: 0\r\n\r\n")
        sock.sendall(request.encode())
        response=b""
        while b"\r\n\r\n" not in response:
            response+=sock.recv(8192)
        return response.decode()
    challenge=exchange(1)
    if not challenge.startswith("SIP/2.0 401"):sys.exit("Expected SIP authentication challenge")
    fields=dict(re.findall(r'(\w+)="([^"]*)"',next(line for line in challenge.splitlines() if line.lower().startswith("www-authenticate:"))))
    realm,nonce=fields["realm"],fields["nonce"]
    cnonce=secrets.token_hex(8)
    digest=md5(md5(user+":"+realm+":"+password)+":"+nonce+":00000001:"+cnonce+":auth:"+md5("REGISTER:"+uri))
    authorization=f'Authorization: Digest username="{user}", realm="{realm}", nonce="{nonce}", uri="{uri}", response="{digest}", algorithm=MD5, qop=auth, nc=00000001, cnonce="{cnonce}"\r\n'
    result=exchange(2,authorization).splitlines()[0]
    print("Public TLS certificate verified; SIP digest:",result)
    if result != "SIP/2.0 200 OK":sys.exit(1)
