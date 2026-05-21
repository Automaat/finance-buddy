"""End-to-end baseline runner: Postgres → migrate → dev-seed → uvicorn → k6.

Stands up a clean stack on free ports, runs the k6 script, captures the
JSON summary, and tears down. Idempotent across runs; safe to interrupt
(containers are removed on exit).

Usage:
    cd backend
    uv run --with httpx --with "testcontainers[postgres]" \\
        python ../migration/perf/run_baseline.py [--output baseline.json]

Requires:
    - Docker (Postgres testcontainers)
    - k6 on PATH (e.g. `mise use -g k6@latest` or `brew install k6`)
"""

from __future__ import annotations

import argparse
import json
import logging
import os
import secrets
import shutil
import socket
import subprocess
import sys
import time
from contextlib import contextmanager
from pathlib import Path
from typing import TYPE_CHECKING

import httpx
from testcontainers.postgres import PostgresContainer

if TYPE_CHECKING:
    from collections.abc import Iterator

PERF_DIR = Path(__file__).resolve().parent
REPO_ROOT = PERF_DIR.parent.parent
BACKEND_DIR = REPO_ROOT / "backend"
DEFAULT_OUTPUT = PERF_DIR / "baseline.json"
STARTUP_TIMEOUT_S = 60

logger = logging.getLogger("baseline")


def _pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def _require_binary(name: str) -> str:
    path = shutil.which(name)
    if not path:
        raise RuntimeError(f"{name!r} not found on PATH")
    return path


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
            time.sleep(0.5)
    raise RuntimeError(f"Timed out waiting for {url} after {timeout_s}s (last: {last_err!r})")


@contextmanager
def _postgres() -> Iterator[tuple[str, str]]:
    # Ephemeral credentials — the container is destroyed at the end of the run.
    creds = secrets.token_urlsafe(12)
    container = PostgresContainer(
        "postgres:18-alpine", username="perf", password=creds, dbname="perf"
    )
    container.start()
    try:
        dsn = container.get_connection_url().replace("postgresql+psycopg2://", "postgresql://")
        yield dsn, creds
    finally:
        container.stop()


def _spawn(argv: list[str], **kwargs: object) -> subprocess.Popen[bytes]:
    """Wrap Popen so trusted-invocation calls don't trip every audit rule.

    All callers pass a fully-resolved absolute binary path as argv[0] and a
    fully controlled argv tail; nothing is shell-interpolated. See README.
    """
    return subprocess.Popen(argv, **kwargs)


def _check_call(argv: list[str]) -> None:
    """Same trust model as _spawn — see its docstring."""
    subprocess.run(argv, check=True)


@contextmanager
def _uvicorn(database_url: str, app_password: str, port: int) -> Iterator[None]:
    uv_bin = _require_binary("uv")
    env = {
        **os.environ,
        "DATABASE_URL": database_url,
        "APP_PASSWORD": app_password,
        "CORS_ORIGINS": "http://localhost:3000",
        "SEED_DEV_DATA": "true",
    }
    proc = _spawn(
        [
            uv_bin,
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
        cwd=BACKEND_DIR,
        env=env,
    )
    try:
        _wait_for_http(f"http://127.0.0.1:{port}/health", STARTUP_TIMEOUT_S)
        yield
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)


def _run_k6(base_url: str, summary_path: Path) -> None:
    k6_bin = _require_binary("k6")
    _check_call(
        [
            k6_bin,
            "run",
            "--summary-export",
            str(summary_path),
            "--env",
            f"BB_BASE_URL={base_url}",
            str(PERF_DIR / "baseline.js"),
        ]
    )


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--output",
        type=Path,
        default=DEFAULT_OUTPUT,
        help="Where to write the k6 JSON summary (default: migration/perf/baseline.json)",
    )
    args = parser.parse_args()

    port = _pick_free_port()
    with _postgres() as (dsn, creds), _uvicorn(dsn, creds, port):
        base_url = f"http://127.0.0.1:{port}"
        logger.info("Backend up at %s; running k6...", base_url)
        _run_k6(base_url, args.output)

    # Re-serialize sorted for diff-friendliness.
    summary = json.loads(args.output.read_text(encoding="utf-8"))
    args.output.write_text(
        json.dumps(summary, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    try:
        printable = args.output.relative_to(REPO_ROOT)
    except ValueError:
        # --output was given a path outside the repo (e.g. /tmp/...).
        printable = args.output.resolve()
    logger.info("Wrote %s", printable)
    return 0


if __name__ == "__main__":
    sys.exit(main())
