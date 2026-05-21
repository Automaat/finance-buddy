"""Run backend-bb-tests against the Go backend for a specific endpoint group.

Bootstrap:
    testcontainers Postgres -> alembic upgrade head -> bb seed
    -> backend-go binary on a random port -> pytest

Pre-built binary required (`go run` is hostile to mixed Go installs):
    cd backend-go && go build -o /tmp/backend-go-bin ./cmd/api

Usage:
    cd backend  # so we can use its uv env for alembic + testcontainers + psycopg2
    uv run --with httpx --with "testcontainers[postgres]" --with psycopg2-binary \\
        python ../scripts/test-go-parity.py --tests tests/test_config.py

Override the binary path with BACKEND_GO_BIN.

The bb-tests harness honors BB_BASE_URL + BB_DATABASE_URL; we set both here
and shell out to pytest from the backend-bb-tests/ dir.
"""

from __future__ import annotations

import argparse
import logging
import os
import shutil
import socket
import subprocess
import sys
import time
from pathlib import Path

import httpx
from testcontainers.postgres import PostgresContainer

REPO_ROOT = Path(__file__).resolve().parent.parent
BACKEND_DIR = REPO_ROOT / "backend"
BACKEND_GO_DIR = REPO_ROOT / "backend-go"
BB_TESTS_DIR = REPO_ROOT / "backend-bb-tests"
STARTUP_TIMEOUT_S = 30

logger = logging.getLogger("test-go-parity")


def pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def wait_for_http(url: str, timeout_s: float) -> None:
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
    raise RuntimeError(f"Timed out waiting for {url} after {timeout_s}s (last: {last_err!r})")


def alembic_upgrade(dsn: str) -> None:
    uv = shutil.which("uv")
    if not uv:
        raise RuntimeError("uv not on PATH")
    env = {
        **os.environ,
        "DATABASE_URL": dsn,
        "APP_PASSWORD": "bb",
        "CORS_ORIGINS": "http://localhost:3000",
    }
    subprocess.run(
        [uv, "run", "alembic", "upgrade", "head"], cwd=BACKEND_DIR, env=env, check=True
    )


def seed_db(dsn: str) -> None:
    sys.path.insert(0, str(BB_TESTS_DIR))
    from fixtures.seed import seed  # local import after sys.path tweak

    seed(dsn)


def start_go_backend(dsn: str, port: int) -> subprocess.Popen[bytes]:
    """Run the already-built backend-go binary.

    `go run` is intentionally avoided — it compiles in-place and picks up
    whatever GOROOT/GOTOOLCHAIN the shell happens to leak in, which is a
    nightmare across mise + system go installs. Build the binary out-of-band
    (see the run instructions in the docstring) and let this script just
    exec it.
    """
    binary = os.environ.get("BACKEND_GO_BIN", "/tmp/backend-go-bin")
    if not os.path.exists(binary):
        raise RuntimeError(
            f"backend-go binary not found at {binary}; "
            "build it first: cd backend-go && go build -o /tmp/backend-go-bin ./cmd/api"
        )
    env = {
        **os.environ,
        "DATABASE_URL": dsn,
        "CORS_ORIGINS": "http://localhost:3000",
        "FB_ADDR": f":{port}",
    }
    return subprocess.Popen([binary], cwd=BACKEND_GO_DIR, env=env)


def run_pytest(base_url: str, dsn: str, tests: str) -> int:
    uv = shutil.which("uv")
    if not uv:
        raise RuntimeError("uv not on PATH")
    env = {
        **os.environ,
        "BB_BASE_URL": base_url,
        "BB_DATABASE_URL": dsn,
        "BB_ALLOW_DESTRUCTIVE_SEED": "1",
    }
    args = [uv, "run", "pytest", "-v"]
    if tests:
        # If the value looks like a list of test paths, pass them as positional
        # args; otherwise treat as a -k expression.
        parts = tests.split()
        if all(p.startswith("tests/") or p.endswith(".py") for p in parts):
            args.extend(parts)
        else:
            args.extend(["-k", tests])
    result = subprocess.run(args, cwd=BB_TESTS_DIR, env=env, check=False)
    return result.returncode


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--tests",
        default="test_config.py test_health.py",
        help='Pytest -k pattern or "tests/test_X.py" filename. Default covers the cutover scope.',
    )
    args = parser.parse_args()

    container = PostgresContainer(
        "postgres:18-alpine", username="parity", password="parity", dbname="parity"
    )
    container.start()
    try:
        dsn = container.get_connection_url().replace("postgresql+psycopg2://", "postgresql://")
        logger.info("postgres up at %s", dsn)
        alembic_upgrade(dsn)
        logger.info("alembic upgrade head ok")
        seed_db(dsn)
        logger.info("seed ok")

        port = pick_free_port()
        proc = start_go_backend(dsn, port)
        try:
            base_url = f"http://127.0.0.1:{port}"
            wait_for_http(f"{base_url}/health", STARTUP_TIMEOUT_S)
            logger.info("backend-go up at %s", base_url)
            return run_pytest(base_url, dsn, args.tests)
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=5)
    finally:
        container.stop()


if __name__ == "__main__":
    sys.exit(main())
