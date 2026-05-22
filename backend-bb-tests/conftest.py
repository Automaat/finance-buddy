"""Black-box test harness for the Finance Buddy Go backend.

Boots backend-go against a real Postgres (testcontainers) and runs the suite
as a regression oracle. (Pre-decommission this also served the Python
backend; that backend is gone — backend-go is the only target now.)

Knobs (all optional, all via environment):
    BB_BASE_URL       Skip launching backend-go; hit this URL instead. Useful
                      when the suite drives an already-running backend-go.
    BB_DATABASE_URL   Skip launching Postgres; use this DSN. Required if
                      BB_BASE_URL is set (so seed can talk to the same DB).
    BB_UPDATE_GOLDEN  Truthy → overwrite the golden/ snapshots with the live
                      response (used to refresh after intentional changes).

Default flow (everything unset) — one-time per session:
    1. Start a Postgres testcontainer.
    2. Build + launch backend-go (it applies internal/db/schema.sql itself).
    3. Seed deterministic fixtures.
    4. Yield an httpx client to tests.
"""

from __future__ import annotations

import os
import socket
import subprocess
import sys
import time
from collections.abc import Iterator
from pathlib import Path

import httpx
import pytest
from testcontainers.postgres import PostgresContainer

from fixtures.seed import seed

REPO_ROOT = Path(__file__).resolve().parent.parent
BACKEND_GO_DIR = REPO_ROOT / "backend-go"
STARTUP_TIMEOUT_S = 30

# backend-go now requires auth config to start and gates every /api route.
# The suite launches it with these credentials and logs in once per session;
# overridable when driving an externally-running backend via BB_BASE_URL.
ADMIN_USERNAME = os.environ.get("BB_ADMIN_USERNAME", "admin")
ADMIN_PASSWORD = os.environ.get("BB_ADMIN_PASSWORD", "bb-test-admin-pw")
JWT_SECRET = os.environ.get("BB_JWT_SECRET", "bb-test-jwt-secret")


def _truthy(value: str | None) -> bool:
    return value is not None and value.lower() in {"1", "true", "yes", "on"}


def _wait_for_http(url: str, timeout_s: float) -> None:
    deadline = time.monotonic() + timeout_s
    last_err: Exception | None = None
    with httpx.Client(timeout=2.0) as probe:
        while time.monotonic() < deadline:
            try:
                response = probe.get(url)
                response.close()
                if response.status_code == 200:
                    return
            except (httpx.HTTPError, OSError) as err:
                last_err = err
            time.sleep(0.25)
    raise RuntimeError(f"Timed out waiting for {url} after {timeout_s}s (last error: {last_err!r})")


def _pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


@pytest.fixture(scope="session")
def database_url() -> Iterator[str]:
    """Provide a Postgres DSN — either externally supplied or testcontainers-spun."""
    external = os.environ.get("BB_DATABASE_URL")
    if external:
        yield external
        return

    if os.environ.get("BB_BASE_URL"):
        raise RuntimeError("BB_BASE_URL is set but BB_DATABASE_URL is not. Seed needs DB access.")

    container = PostgresContainer("postgres:18-alpine", username="bb", password="bb", dbname="bb")
    container.start()
    try:
        dsn = container.get_connection_url().replace("postgresql+psycopg2://", "postgresql://")
        yield dsn
    finally:
        container.stop()


@pytest.fixture(scope="session")
def _go_binary() -> Path:
    """Build the backend-go binary once per session."""
    binary = Path(subprocess.check_output(["mktemp"], text=True).strip())
    subprocess.run(
        ["go", "build", "-o", str(binary), "./cmd/api"],
        cwd=BACKEND_GO_DIR,
        check=True,
    )
    return binary


@pytest.fixture(scope="session")
def base_url(database_url: str, _go_binary: Path) -> Iterator[str]:
    """Yield the base URL — externally supplied, or a locally launched
    backend-go. backend-go applies the baseline schema on startup, then the
    fixtures are seeded against the now-migrated database."""
    external = os.environ.get("BB_BASE_URL")
    if external:
        seed(database_url)
        yield external.rstrip("/")
        return

    port = _pick_free_port()
    env = {
        **os.environ,
        "DATABASE_URL": database_url,
        "CORS_ORIGINS": "http://localhost:3000",
        "FB_ADDR": f":{port}",
        "FB_JWT_SECRET": JWT_SECRET,
        "FB_ADMIN_USERNAME": ADMIN_USERNAME,
        "FB_ADMIN_PASSWORD": ADMIN_PASSWORD,
    }
    proc = subprocess.Popen(
        [str(_go_binary)],
        env=env,
        stdout=sys.stdout,
        stderr=sys.stderr,
    )
    url = f"http://127.0.0.1:{port}"
    try:
        _wait_for_http(f"{url}/health", STARTUP_TIMEOUT_S)
        seed(database_url)
        yield url
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)


@pytest.fixture(scope="session")
def client(base_url: str) -> Iterator[httpx.Client]:
    """Session-scoped httpx client, logged in as admin so the session cookie
    rides every request to the now-gated /api routes."""
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        response = http.post(
            "/api/auth/login",
            json={"username": ADMIN_USERNAME, "password": ADMIN_PASSWORD},
        )
        response.raise_for_status()
        yield http


@pytest.fixture(scope="session")
def owner_ids(client: httpx.Client) -> dict[str, int]:
    """Map household-member display name -> user id, for owner_user_id fields."""
    response = client.get("/api/users")
    response.raise_for_status()
    return {u["name"]: u["id"] for u in response.json()}


@pytest.fixture(scope="session")
def update_golden() -> bool:
    return _truthy(os.environ.get("BB_UPDATE_GOLDEN"))
