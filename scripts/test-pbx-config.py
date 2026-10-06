#!/usr/bin/env python3
"""Verify PBX credential rendering without a server or provider API."""
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "bridge", Path(__file__).resolve().parent.parent / "deploy/asterisk/bridge.py")
bridge = importlib.util.module_from_spec(spec)
spec.loader.exec_module(bridge)


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
