"""Dump the FastAPI OpenAPI spec to api/openapi.v1.json (repo root).

Used as the frozen contract for the Python→Go backend migration. The Go
backend must satisfy this schema byte-for-byte. A CI check re-runs this
script and fails if the live FastAPI output diverges from the committed
file.

Run from backend/:
    uv run python scripts/dump_openapi.py
"""

from __future__ import annotations

import importlib
import json
import os
import sys
from pathlib import Path

BACKEND_DIR = Path(__file__).resolve().parent.parent
REPO_ROOT = BACKEND_DIR.parent
OUTPUT = REPO_ROOT / "api" / "openapi.v1.json"


def _prepare_environment() -> None:
    os.environ.setdefault("DATABASE_URL", "postgresql://placeholder/placeholder")
    os.environ.setdefault("APP_PASSWORD", "placeholder")
    os.environ.setdefault("CORS_ORIGINS", "http://localhost:3000")
    if str(BACKEND_DIR) not in sys.path:
        sys.path.insert(0, str(BACKEND_DIR))


def main() -> None:
    _prepare_environment()
    app_module = importlib.import_module("app.main")
    spec = app_module.app.openapi()
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with OUTPUT.open("w", encoding="utf-8") as f:
        json.dump(spec, f, indent=2, sort_keys=True, ensure_ascii=False)
        f.write("\n")
    print(f"Wrote {OUTPUT.relative_to(REPO_ROOT)}")


if __name__ == "__main__":
    main()
