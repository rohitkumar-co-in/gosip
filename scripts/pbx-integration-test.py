#!/usr/bin/env python3
"""Run on the VPS against an isolated PBX and dummy HTTP backend.

Uses synthetic phone numbers and no Twilio API. Requires sudo Docker access.
"""
import base64
import hashlib
import http.server
import json
import os
from pathlib import Path
import re
import secrets
import socket
import sqlite3
import ssl
import subprocess
import threading
import time

os.umask(0o077)
root=Path('/home/ubuntu/gosip-pbx-integration')
root.mkdir(exist_ok=True)
username='testphone'
password=secrets.token_hex(20)
secret=secrets.token_hex(32)
container='gosip-pbx-integration'
docker=lambda *args:subprocess.check_output(['sudo','-n','docker',*args],text=True).strip()
gateway=docker('network','inspect','bridge','--format','{{range .IPAM.Config}}{{.Gateway}}{{end}}')
source=root/'gosip.db'
if source.exists():source.unlink()
db=sqlite3.connect(source)
db.executescript('CREATE TABLE devices(id INTEGER,username TEXT,password_hash TEXT); CREATE TABLE routes(did_id INTEGER,enabled INTEGER,action_type TEXT,action_data TEXT); CREATE TABLE messages(id INTEGER,did_id INTEGER,direction TEXT,from_number TEXT,body TEXT,media_urls TEXT,status TEXT);')
db.execute('INSERT INTO devices VALUES(1,?,?)',(username,hashlib.md5((username+':gosip:'+password).encode()).hexdigest()))
db.execute('INSERT INTO routes VALUES(1,1,?,?)',('ring',json.dumps({'devices':[1]})))
db.commit()
received=[]
invitations=[]
class Backend(http.server.BaseHTTPRequestHandler):
    def log_message(self,*_):pass
    def do_POST(self):
        assert self.path=='/api/pbx/messages'
        assert self.headers.get('Authorization')=='Bearer '+secret
        payload=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        received.append(payload)
        self.send_response(202);self.end_headers();self.wfile.write(b'{"id":99}')
backend=http.server.ThreadingHTTPServer((gateway,18088),Backend)
threading.Thread(target=backend.serve_forever,daemon=True).start()
env={'GOSIP_PBX_SECRET':secret,'GOSIP_PBX_TRUNK_USER':'twilio-in','GOSIP_PBX_TRUNK_PASSWORD':secrets.token_hex(20),'PBX_DOMAIN':'sip.leadomi.com','PBX_PUBLIC_IP':'18.134.241.218','PBX_TWILIO_USER':'dummy','PBX_TWILIO_PASSWORD':'dummy','TWILIO_SIP_DOMAIN':'dummy.sip.twilio.com','GOSIP_OUTBOUND_CALLER_ID':'+441234567890','GOSIP_URL':'http://'+gateway+':18088','PBX_TWILIO_NETWORKS':'192.0.2.0/24'}
envpath=root/'test.env';envpath.write_text('\n'.join(k+'='+v for k,v in env.items())+'\n')
try:docker('rm','-f',container)
except subprocess.CalledProcessError:pass
docker('run','-d','--name',container,'--env-file',str(envpath),'-p','127.0.0.1:16061:5061','-v',str(root)+':/gosip:ro','-v','/data/gosip-pbx/certs:/certs:ro','-v','/home/ubuntu/bridge.py:/opt/gosip/bridge.py:ro','gosip-pbx:staging')
time.sleep(5)
context=ssl.create_default_context()
sock=context.wrap_socket(socket.create_connection(('127.0.0.1',16061),timeout=10),server_hostname='sip.leadomi.com')
sock.settimeout(15)
buffer=b''
def receive():
    global buffer
    while b'\r\n\r\n' not in buffer:buffer+=sock.recv(8192)
    head,body=buffer.split(b'\r\n\r\n',1)
    length=int(re.search(br'(?im)^Content-Length:\s*(\d+)',head)[1])
    while len(body)<length:body+=sock.recv(8192)
    buffer=body[length:]
    return head.decode(),body[:length].decode()
def response(head):
    if head.startswith('ACK '):return
    headers={key.lower():value.strip() for key,_,value in (line.partition(':') for line in head.splitlines()[1:])}
    if head.startswith('INVITE ') and ';tag=' not in headers['to']:headers['to']+=';tag=test-phone'
    status='486 Busy Here' if head.startswith('INVITE ') else '200 OK'
    sock.sendall(('SIP/2.0 '+status+'\r\n'+''.join(key+': '+headers[key.lower()]+'\r\n' for key in ['Via','From','To','Call-ID','CSeq'])+'Content-Length: 0\r\n\r\n').encode())
def request(method,user,to,body='',auth='',expiry=300,seq=1,callid=None):
    uri='sip:'+to+'@sip.leadomi.com'
    if method=='REGISTER':uri='sip:sip.leadomi.com'
    content_type='application/sdp' if method=='INVITE' else 'text/plain'
    packet=(f'{method} {uri} SIP/2.0\r\nVia: SIP/2.0/TLS 127.0.0.1:{sock.getsockname()[1]};branch=z9hG4bK{secrets.token_hex(10)};rport\r\nMax-Forwards: 70\r\nFrom: <sip:{user}@sip.leadomi.com>;tag=integration\r\nTo: <sip:{to}@sip.leadomi.com>\r\nCall-ID: {callid or secrets.token_hex(10)}\r\nCSeq: {seq} {method}\r\nContact: <sip:{username}@127.0.0.1:{sock.getsockname()[1]};transport=tls>\r\nExpires: {expiry}\r\n{auth}Content-Type: {content_type}\r\nContent-Length: {len(body.encode())}\r\n\r\n{body}')
    sock.sendall(packet.encode())
    while True:
        head,content=receive()
        if not head.startswith('SIP/2.0'):
            if head.startswith('INVITE '):invitations.append((head,content))
            response(head)
            continue
        if head.startswith('SIP/2.0 100'):continue
        return head,content,uri
def authenticated(method,user,to,body='',expiry=300,authuser=None,authpassword=None):
    callid=secrets.token_hex(10)
    head,_,uri=request(method,user,to,body,expiry=expiry,callid=callid)
    assert head.startswith('SIP/2.0 401'),head.splitlines()[0]
    fields=dict(re.findall(r'(\w+)="([^"]*)"',next(line for line in head.splitlines() if line.lower().startswith('www-authenticate:'))))
    realm,nonce=fields['realm'],fields['nonce'];cnonce=secrets.token_hex(8)
    md5=lambda v:hashlib.md5(v.encode()).hexdigest()
    authuser=authuser or user
    ha1=md5(authuser+':'+realm+':'+(authpassword or password))
    digest=md5(ha1+':'+nonce+':00000001:'+cnonce+':auth:'+md5(method+':'+uri))
    authorization=f'Authorization: Digest username="{authuser}", realm="{realm}", nonce="{nonce}", uri="{uri}", response="{digest}", algorithm=MD5, qop=auth, nc=00000001, cnonce="{cnonce}"\r\n'
    return request(method,user,to,body,authorization,expiry,2,callid)[0]
try:
    head=authenticated('REGISTER',username,username)
    assert head.startswith('SIP/2.0 200'),head.splitlines()[0]
    print('PASS authenticated TLS registration with existing HA1 format')
    body='Unicode hello ✓\nSecond line'
    head=authenticated('MESSAGE',username,'+441111111111',body)
    assert head.startswith('SIP/2.0 202'),head.splitlines()[0]
    for _ in range(15):
        if received:break
        time.sleep(.2)
    assert received==[{'username':username,'to_number':'+441111111111','body':body}],received
    print('PASS SIP MESSAGE → authenticated HTTP SMS bridge; Unicode/newlines intact')
    # Real GoSIP SMS rows encode a nil media URL slice as JSON null.
    db.execute("INSERT INTO messages VALUES(1,1,'inbound','+442222222222',?, 'null','received')",('Incoming test\n✓',));db.commit()
    while True:
        head,content=receive()
        response(head)
        if head.startswith('MESSAGE'):
            assert content=='Incoming test\n✓',content
            assert '+442222222222@' in head,head
            break
    print('PASS inbox → Asterisk → registered TLS phone with sender identity')
    authenticated('REGISTER',username,username,expiry=0)
    db.execute("INSERT INTO messages VALUES(2,1,'inbound','+443333333333','Offline test', '[]','received')");db.commit()
    time.sleep(5)
    pending=docker('exec',container,'python3','-c',"import sqlite3;print(sqlite3.connect('/var/lib/gosip-pbx/delivery.db').execute('SELECT count(*) FROM deliveries WHERE delivered=0').fetchone()[0])")
    assert pending=='1',pending
    print('PASS offline message persists in delivery queue')
    authenticated('REGISTER',username,username)
    while True:
        head,content=receive();response(head)
        if head.startswith('MESSAGE'):
            assert content=='Offline test';break
    print('PASS queued SMS delivered after phone re-registers')
    password=secrets.token_hex(20)
    db.execute('UPDATE devices SET password_hash=? WHERE id=1',(hashlib.md5((username+':gosip:'+password).encode()).hexdigest(),));db.commit()
    time.sleep(12)
    head=authenticated('REGISTER',username,username)
    assert head.startswith('SIP/2.0 200'),head.splitlines()[0]
    print('PASS device password update synchronizes and reloads without restart')
    # Simulate a Twilio source address only after phone registration tests.
    docker('exec',container,'python3','-c',"from pathlib import Path;p=Path('/etc/asterisk/pjsip.conf');p.write_text(p.read_text().replace('endpoint=twilio-in\\n','endpoint=twilio-in\\nmatch="+gateway+"/32\\n'))")
    docker('exec',container,'asterisk','-rx','module reload res_pjsip.so')
    docker('exec',container,'asterisk','-rx','module reload res_pjsip_endpoint_identifier_ip.so')
    time.sleep(1)
    sdp=('v=0\r\no=- 1 1 IN IP4 '+gateway+'\r\ns=Integration\r\nc=IN IP4 '+gateway+'\r\nt=0 0\r\nm=audio 18000 RTP/SAVP 0 8\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:8 PCMA/8000\r\na=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:'+base64.b64encode(secrets.token_bytes(30)).decode()+'\r\na=sendrecv\r\n')
    head=authenticated('INVITE','twilio-in',username,sdp,authpassword=env['GOSIP_PBX_TRUNK_PASSWORD'])
    assert head.startswith(('SIP/2.0 486','SIP/2.0 403','SIP/2.0 603')),head.splitlines()[0]
    assert len(invitations)==1 and 'a=crypto:' in invitations[0][1],invitations
    assert 'c=IN IP4 18.134.241.218' in invitations[0][1],invitations[0][1]
    print('PASS authenticated inbound trunk → local phone; SRTP and public media address offered')
finally:
    sock.close();db.close();backend.shutdown()
    docker('rm','-f',container)
