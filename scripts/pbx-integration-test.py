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
import urllib.request

os.umask(0o077)
root=Path('/home/ubuntu/gosip-pbx-integration')
root.mkdir(exist_ok=True)
username='testphone'
password=secrets.token_hex(20)
second_password=password
secret=secrets.token_hex(32)
container='gosip-pbx-integration'
docker=lambda *args:subprocess.check_output(['sudo','-n','docker',*args],text=True).strip()
gateway=docker('network','inspect','bridge','--format','{{range .IPAM.Config}}{{.Gateway}}{{end}}')
source=root/'gosip.db'
if source.exists():source.unlink()
db=sqlite3.connect(source)
db.executescript('CREATE TABLE config(key TEXT PRIMARY KEY,value TEXT); CREATE TABLE devices(id INTEGER,username TEXT,password_hash TEXT); CREATE TABLE routes(did_id INTEGER,enabled INTEGER,action_type TEXT,action_data TEXT); CREATE TABLE messages(id INTEGER,did_id INTEGER,direction TEXT,from_number TEXT,body TEXT,media_urls TEXT,status TEXT);')
db.execute('INSERT INTO devices VALUES(1,?,?)',(username,hashlib.md5((username+':gosip:'+password).encode()).hexdigest()))
db.execute('INSERT INTO devices VALUES(2,?,?)',('testphone2',hashlib.md5(('testphone2:gosip:'+password).encode()).hexdigest()))
db.execute('INSERT INTO config VALUES(?,?)',('pbx_device_numbers',json.dumps({'testphone2':'+442345678901'})))
db.execute('INSERT INTO routes VALUES(1,1,?,?)',('ring',json.dumps({'devices':[1]})))
db.execute('INSERT INTO routes VALUES(2,1,?,?)',('ring',json.dumps({'devices':[2]})))
db.executescript("CREATE TABLE sip_accounts(device_id INTEGER PRIMARY KEY,enabled INTEGER,state TEXT); INSERT INTO sip_accounts VALUES(1,1,'ready'); INSERT INTO sip_accounts VALUES(2,1,'ready');")
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
env={'GOSIP_PBX_SECRET':secret,'GOSIP_PBX_TRUNK_USER':'twilio-in','GOSIP_PBX_TRUNK_PASSWORD':'Trunk@#'+secrets.token_hex(20),'PBX_DOMAIN':'sip.leadomi.com','PBX_PUBLIC_IP':'18.134.241.218','PBX_TWILIO_USER':'dummy','PBX_TWILIO_PASSWORD':'Provider@#'+secrets.token_hex(20),'TWILIO_SIP_DOMAIN':'dummy.sip.twilio.com','GOSIP_OUTBOUND_CALLER_ID':'+441234567890','GOSIP_URL':'http://'+gateway+':18088','PBX_TWILIO_NETWORKS':'192.0.2.0/24'}
envpath=root/'test.env';envpath.write_text('\n'.join(k+'='+v for k,v in env.items())+'\n')
env['PBX_TWILIO_DEVICE_PASSWORDS']=json.dumps({'testphone2':secrets.token_hex(20)})
envpath.write_text('\n'.join(k+'='+v for k,v in env.items())+'\n')
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
    trunk=docker('exec',container,'asterisk','-rx','pjsip show aor twilio-out')
    assert ';transport=tls;secure=true' in trunk, 'Twilio contact URI parameters were parsed as config comments'
    transport=docker('exec',container,'asterisk','-rx','pjsip show transport tls')
    assert re.search(r'allow_wildcard_certs\s*:\s*Yes',transport,re.I), 'Twilio wildcard certificate support missing'
    assert re.search(r'verify_server\s*:\s*Yes',transport,re.I), 'Server certificate validation must stay enabled'
    print('PASS Twilio TLS contact parameters and verified wildcard certificate configuration')
    for identifier, number, trunk in [(1, '+441234567890', 'twilio-out'), (2, '+442345678901', 'twilio-out-2')]:
        rules=docker('exec',container,'asterisk','-rx',f'dialplan show +441111111111@from-phone-{identifier}')
        assert f'CALLERID(num)={number}' in rules and f'@{trunk},60' in rules
    second=docker('exec',container,'asterisk','-rx','pjsip show endpoint twilio-out-2')
    assert re.search(r'from_user\s*:\s*testphone2',second), 'Second phone must retain its identity through Twilio'
    print('PASS separate phone call contexts use assigned caller IDs and SIP identities')
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
    # Real Leadomi SIP SMS rows encode a nil media URL slice as JSON null.
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
    first_password=password
    username='testphone2';password=second_password
    head=authenticated('REGISTER',username,username)
    assert head.startswith('SIP/2.0 200'),head.splitlines()[0]
    db.execute("INSERT INTO messages VALUES(3,2,'inbound','+444444444444','Second phone only','null','received')");db.commit()
    while True:
        head,content=receive();response(head)
        if head.startswith('MESSAGE') and content=='Second phone only':
            assert 'sip:testphone2@' in head;break
    queued=docker('exec',container,'python3','-c',"import sqlite3;print(sqlite3.connect('/var/lib/gosip-pbx/delivery.db').execute('SELECT username FROM deliveries WHERE message_id=3').fetchall())")
    assert queued=="[('testphone2',)]",queued
    print('PASS second DID incoming SMS reaches only its assigned phone')
    private_port=json.loads(docker('inspect',container))[0]['NetworkSettings']['Networks']['bridge']['IPAddress']
    payload=json.dumps({'username':'testphone2','password':second_password}).encode()
    req=urllib.request.Request('http://'+private_port+':8088/credentials',data=payload,headers={'Authorization':'Bearer '+secret,'Content-Type':'application/json'})
    assert urllib.request.urlopen(req,timeout=5).status==200
    stored=docker('exec',container,'python3','-c',"from pathlib import Path;import json;p=Path('/var/lib/gosip-pbx/credentials.json');print(oct(p.stat().st_mode & 0o777),len(json.loads(p.read_text())))")
    assert stored=='0o600 1',stored
    print('PASS authenticated private credential storage is persistent and mode 0600')
    db.execute("UPDATE sip_accounts SET enabled=0,state='disabled' WHERE device_id=2");db.commit();time.sleep(12)
    endpoints=docker('exec',container,'asterisk','-rx','pjsip show endpoints')
    assert 'testphone2/' not in endpoints,endpoints
    print('PASS disabled SIP user removed from active phone endpoints')
    username='testphone';password=first_password
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
