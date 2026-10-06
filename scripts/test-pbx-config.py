#!/usr/bin/env python3
"""Verify PBX credential rendering without a server or provider API."""
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
import sqlite3
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "bridge", Path(__file__).resolve().parent.parent / "deploy/asterisk/bridge.py")
bridge = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bridge)


class InboxRecovery(unittest.TestCase):
    def test_reset_inbox_reuses_ids_without_skipping_or_replaying(self):
        with tempfile.TemporaryDirectory() as directory:
            path = str(Path(directory) / 'inbox.db')
            inbox = sqlite3.connect(path)
            inbox.execute('CREATE TABLE messages(id INTEGER PRIMARY KEY,did_id INTEGER,from_number TEXT,body TEXT,media_urls TEXT,direction TEXT)')
            inbox.execute("INSERT INTO messages VALUES(1,2,'+441234567890','new text','[]','inbound')")
            inbox.commit()
            queue = sqlite3.connect(':memory:')
            queue.executescript("CREATE TABLE meta(key TEXT PRIMARY KEY,value TEXT); INSERT INTO meta VALUES('cursor','50'); CREATE TABLE deliveries(message_id INTEGER,username TEXT,sender TEXT,body TEXT,delivered INTEGER DEFAULT 0,PRIMARY KEY(message_id,username));")
            queue.execute("INSERT INTO deliveries VALUES(1,'phone','+441234567890','old text',1)")
            with patch.object(bridge, 'DB', path), patch.object(bridge, 'targets', return_value=['phone']):
                bridge.enqueue(queue)
                self.assertEqual(queue.execute('SELECT value FROM meta').fetchone()[0], '1')
                self.assertEqual(queue.execute('SELECT body,delivered FROM deliveries').fetchone(), ('new text', 0))
                queue.execute('UPDATE deliveries SET delivered=1')
                bridge.enqueue(queue)
                self.assertEqual(queue.execute('SELECT delivered FROM deliveries').fetchone()[0], 1)
                # Removing every message and receiving another at the same ID
                # must recover after the bridge observes the emptied inbox.
                inbox.execute('DELETE FROM messages')
                inbox.commit()
                bridge.enqueue(queue)
                inbox.execute("INSERT INTO messages VALUES(1,2,'+441234567890','next text','[]','inbound')")
                inbox.commit()
                bridge.enqueue(queue)
                self.assertEqual(queue.execute('SELECT body,delivered FROM deliveries').fetchone(), ('next text', 0))
            inbox.close()
            queue.close()


class CredentialRendering(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        state = patch.object(bridge, "STATE", Path(temporary.name))
        state.start()
        self.addCleanup(state.stop)
        env = patch.dict(os.environ, {
            "PBX_PUBLIC_IP": "192.0.2.1", "TWILIO_SIP_DOMAIN": "test.sip.twilio.com",
            "PBX_TWILIO_USER": "provider", "PBX_TWILIO_PASSWORD": "Provider@#Test12",
            "GOSIP_PBX_TRUNK_USER": "trunk", "GOSIP_PBX_TRUNK_PASSWORD": "Incoming@#Test12",
            "PBX_TWILIO_DEVICE_PASSWORDS": '{"phone":"Device@#Test12"}',
            "GOSIP_OUTBOUND_CALLER_ID": "+441234567890",
        })
        env.start()
        self.addCleanup(env.stop)

    def test_password_punctuation_survives_rendering(self):
        config, _ = bridge.render([
            {"id": 1, "username": "phone", "password_hash": "a" * 32}
        ], {"phone": "+441234567890"})
        for password in ("Provider@#Test12", "Incoming@#Test12", "Device@#Test12"):
            self.assertIn("password=" + password + "\n", config)

    def test_password_cannot_inject_configuration(self):
        for value in ("secret;comment", "secret\n[admin]", "secret\rpassword=x",
                      "secret\\escape", 'secret"quote', "secret space", "x" * 129):
            with self.subTest(value=value), patch.dict(os.environ, {"PBX_TWILIO_PASSWORD": value}):
                with self.assertRaises(ValueError):
                    bridge.render([])

    def test_username_rules_remain_strict(self):
        with patch.dict(os.environ, {"PBX_TWILIO_USER": "provider@host"}):
            with self.assertRaises(ValueError):
                bridge.render([])


if __name__ == "__main__":
    unittest.main()
