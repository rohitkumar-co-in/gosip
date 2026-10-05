"""Probe HTTP assets and SIP signaling for a deployed Leadomi SIP container."""
import argparse
import json
import re
import socket
import urllib.request
import uuid

parser = argparse.ArgumentParser()
parser.add_argument('--http', required=True)
parser.add_argument('--sip-host', default='127.0.0.1')
parser.add_argument('--sip-port', type=int, default=5060)
args = parser.parse_args()

def fetch(path):
    with urllib.request.urlopen(args.http.rstrip('/') + path, timeout=10) as response:
        assert response.status == 200
        return response.read().decode()

assert json.loads(fetch('/api/health'))['status'] == 'healthy'
html = fetch('/')
assets = re.findall(r'(?:src|href)="(/assets/[^\"]+)"', html)
assert assets, 'No production frontend assets found'
for asset in assets:
    assert fetch(asset), f'Empty asset: {asset}'
print('HTTP health, frontend and assets passed')

for transport, socktype in [('UDP', socket.SOCK_DGRAM), ('TCP', socket.SOCK_STREAM)]:
    with socket.socket(socket.AF_INET, socktype) as client:
        client.settimeout(8)
        client.connect((args.sip_host, args.sip_port))
        local_host, local_port = client.getsockname()
        request = '\r\n'.join([
            f'OPTIONS sip:{args.sip_host}:{args.sip_port} SIP/2.0',
            f'Via: SIP/2.0/{transport} {local_host}:{local_port};branch=z9hG4bK{uuid.uuid4().hex};rport',
            'Max-Forwards: 70',
            f'From: <sip:probe@{local_host}>;tag=deployment-probe',
            f'To: <sip:{args.sip_host}>',
            f'Call-ID: {uuid.uuid4().hex}@deployment-probe',
            'CSeq: 1 OPTIONS',
            'Content-Length: 0', '', '',
        ]).encode()
        client.sendall(request)
        response = client.recv(8192).decode()
        assert response.startswith('SIP/2.0 200'), response.splitlines()[0]
        print(f'SIP {transport} OPTIONS passed')
