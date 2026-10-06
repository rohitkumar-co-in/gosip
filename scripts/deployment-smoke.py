#!/usr/bin/env python3
"""Read-only HTTP/asset and optional PBX TLS checks; no calls or SMS."""
import argparse
import json
import re
import socket
import ssl
import urllib.error
import urllib.request

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--http', required=True)
parser.add_argument('--tls-host', help='Optional PBX hostname with a trusted certificate')
parser.add_argument('--tls-port', type=int, default=5061)
args = parser.parse_args()
base = args.http.rstrip('/')

def fetch(path):
    with urllib.request.urlopen(base + path, timeout=10) as response:
        assert response.status == 200
        return response.read().decode()

assert json.loads(fetch('/api/health'))['status'] == 'healthy'
html = fetch('/')
assets = re.findall(r'(?:src|href)="(/assets/[^\"]+)"', html)
assert assets, 'No production frontend assets found'
for asset in assets:
    assert fetch(asset), f'Empty asset: {asset}'
for route in ('/login', '/setup', '/devices', '/dids', '/excluded-numbers',
              '/routes', '/activity', '/calls', '/messages', '/voicemails',
              '/settings', '/users'):
    assert fetch(route) == html, f'SPA route fallback missing: {route}'
try:
    fetch('/api/business/accounts')
except urllib.error.HTTPError as error:
    assert error.code == 401, f'Unexpected protected API status: {error.code}'
else:
    raise AssertionError('Protected account API accepts anonymous access')
print('PASS HTTP health, frontend assets, 12 SPA routes and authentication boundary')
if args.tls_host:
    with socket.create_connection((args.tls_host, args.tls_port), timeout=10) as connection:
        with ssl.create_default_context().wrap_socket(connection, server_hostname=args.tls_host):
            print('PASS PBX TLS connection and trusted hostname certificate')
