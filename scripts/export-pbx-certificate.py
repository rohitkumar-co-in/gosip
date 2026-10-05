#!/usr/bin/env python3
"""Export only this PBX certificate from Traefik ACME storage (run as root).

Install as an hourly systemd timer. This never changes Traefik's storage.
"""
import base64
import json
import os
from pathlib import Path
import sys

domain, source, target = sys.argv[1:]
os.umask(0o077)
directory = Path(target)
directory.mkdir(parents=True, exist_ok=True)
directory.chmod(0o700)
storage = json.loads(Path(source).read_text())
for resolver in storage.values():
    for certificate in resolver.get("Certificates", []):
        names = certificate["domain"]
        if domain == names.get("main") or domain in names.get("sans", []):
            for name, key in (("privkey.pem", "key"), ("fullchain.pem", "certificate")):
                data = base64.b64decode(certificate[key], validate=True)
                path = directory / name
                if path.exists() and path.read_bytes() == data:
                    continue
                temporary = directory / (name + ".new")
                temporary.write_bytes(data)
                temporary.chmod(0o600)
                temporary.replace(path)
            print("PBX certificate synchronized for " + domain)
            sys.exit(0)
sys.exit("Certificate for requested PBX domain was not found")
