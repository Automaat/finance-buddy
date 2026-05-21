"""Black-box test harness for the Finance Buddy backend.

Boots a real HTTP server against a real Postgres (testcontainers) and serves
the same suite as a parity oracle for both the Python and Go backends.

Knobs (all optional, all via environment):
    BB_BASE_URL       Skip launching uvicorn; hit this URL instead. Useful when
                      pointing the suite at a running Go backend during cutover.
    BB_DATABASE_URL   Skip launching Postgres; use this DSN. Required if
                      BB_BASE_URL is set (so seed can talk to the same DB).
    BB_BACKEND_DIR    Override the backend source directory used to launch
                      uvicorn. Defaults to ../backend relative to this file.
    BB_UPDATE_GOLDEN  Truthy → overwrite the golden/ snapshots with the live
                      response (used to refresh after intentional changes).

Default flow (everything unset) — one-time per session:
    1. Start a Postgres testcontainer.
    2. Run `alembic upgrade head` against it.
    3. Seed deterministic fixtures.
    4. Launch uvicorn pointed at that Postgres.
    5. Yield an httpx client to tests.
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
DEFAULT_BACKEND_DIR = REPO_ROOT / "backend"
STARTUP_TIMEOUT_S = 30


def _truthy(value: str | None) -> bool:
    return value is not None and value.lower() in {"1", "true", "yes", "on"}


def _wait_for_http(url: str, timeout_s: float) -> None:
    deadline = time.monotonic() + timeout_s
    last_err: Exception | None = None
    while time.monotonic() < deadline:
        try:
            response = httpx.get(url, timeout=2.0)
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
def backend_dir() -> Path:
    override = os.environ.get("BB_BACKEND_DIR")
    return Path(override).resolve() if override else DEFAULT_BACKEND_DIR


@pytest.fixture(scope="session")
def _migrated_db(database_url: str, backend_dir: Path) -> str:
    """Run alembic upgrade head once per session."""
    if os.environ.get("BB_BASE_URL"):
        # Backend is external; assume it already manages its own schema.
        return database_url
    env = {
        **os.environ,
        "DATABASE_URL": database_url,
        "APP_PASSWORD": "bb",
        "CORS_ORIGINS": "http://localhost:3000",
    }
    subprocess.run(
        ["uv", "run", "alembic", "upgrade", "head"],
        cwd=backend_dir,
        env=env,
        check=True,
    )
    return database_url


@pytest.fixture(scope="session")
def seeded_db(_migrated_db: str) -> str:
    """Seed deterministic fixture data once per session."""
    seed(_migrated_db)
    return _migrated_db


@pytest.fixture(scope="session")
def base_url(seeded_db: str, backend_dir: Path) -> Iterator[str]:
    """Yield the base URL for the API — externally supplied or locally launched."""
    external = os.environ.get("BB_BASE_URL")
    if external:
        yield external.rstrip("/")
        return

    port = _pick_free_port()
    env = {
        **os.environ,
        "DATABASE_URL": seeded_db,
        "APP_PASSWORD": "bb",
        "CORS_ORIGINS": "http://localhost:3000",
    }
    proc = subprocess.Popen(
        [
            "uv",
            "run",
            "uvicorn",
            "app.main:app",
            "--host",
            "127.0.0.1",
            "--port",
            str(port),
            "--log-level",
            "warning",
        ],
        cwd=backend_dir,
        env=env,
        stdout=sys.stdout,
        stderr=sys.stderr,
    )
    url = f"http://127.0.0.1:{port}"
    try:
        _wait_for_http(f"{url}/health", STARTUP_TIMEOUT_S)
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
    """Session-scoped httpx client pointed at the live backend."""
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        yield http


@pytest.fixture(scope="session")
def update_golden() -> bool:
    return _truthy(os.environ.get("BB_UPDATE_GOLDEN"))
