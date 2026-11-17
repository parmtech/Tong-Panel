"""In-process synthetic ASGI fixture. No listener or external dependencies."""
import json
import os


async def app(scope, receive, send):
    if scope["type"] != "http":
        return
    method = scope.get("method", "GET")
    path = scope.get("path", "/")
    status = 200
    headers = [(b"content-type", b"application/json")]
    if method != "GET":
        status = 405
        headers.append((b"allow", b"GET"))
        body = b'{"error":"method not allowed"}'
    elif path == "/healthz":
        ready = os.environ.get("P3_FIXTURE_READY", "1") != "0"
        status = 200 if ready else 503
        body = json.dumps({"ready": ready}).encode()
    elif path == "/":
        release = os.environ.get("P3_FIXTURE_RELEASE", "fixture-release")
        body = json.dumps({"release": release}).encode()
    else:
        status = 404
        body = b'{"error":"not found"}'
    await send({"type": "http.response.start", "status": status, "headers": headers})
    await send({"type": "http.response.body", "body": body})
