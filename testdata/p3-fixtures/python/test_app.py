import asyncio
import json
import os
import unittest
from unittest.mock import patch

from app import app


async def invoke(path="/", method="GET"):
    sent = []
    async def receive():
        return {"type": "http.request", "body": b"", "more_body": False}
    async def send(message):
        sent.append(message)
    await app({"type": "http", "method": method, "path": path}, receive, send)
    return sent[0]["status"], json.loads(sent[1]["body"])


class FixtureTests(unittest.TestCase):
    def test_ready_and_release_identity(self):
        with patch.dict(os.environ, {"P3_FIXTURE_RELEASE": "fixture-abc", "P3_FIXTURE_READY": "1"}):
            self.assertEqual(asyncio.run(invoke("/healthz")), (200, {"ready": True}))
            self.assertEqual(asyncio.run(invoke("/")), (200, {"release": "fixture-abc"}))

    def test_unhealthy_unknown_and_method_denial(self):
        with patch.dict(os.environ, {"P3_FIXTURE_READY": "0"}):
            self.assertEqual(asyncio.run(invoke("/healthz")), (503, {"ready": False}))
        self.assertEqual(asyncio.run(invoke("/missing")), (404, {"error": "not found"}))
        self.assertEqual(asyncio.run(invoke("/", "POST")), (405, {"error": "method not allowed"}))


if __name__ == "__main__":
    unittest.main()
