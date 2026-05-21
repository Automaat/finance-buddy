"""Golden-file helpers for capturing and comparing API responses."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

GOLDEN_DIR = Path(__file__).resolve().parent / "golden"


def _path_for(name: str) -> Path:
    if "/" in name or name.startswith("."):
        raise ValueError(f"Golden name must be a flat slug, got {name!r}")
    return GOLDEN_DIR / f"{name}.json"


def _normalize(payload: Any) -> str:
    return json.dumps(payload, indent=2, sort_keys=True, ensure_ascii=False) + "\n"


def assert_matches_golden(name: str, actual: Any, update: bool = False) -> None:
    """Compare ``actual`` against golden/<name>.json or write it when ``update`` is true.

    Both sides are normalized through the same json.dumps(sort_keys=True, indent=2)
    pipeline before comparison so key-order, whitespace, or float-repr differences
    don't fail the assertion. Numeric tolerance, if needed, must be handled in the
    test by normalizing the payload before calling this helper.
    """
    path = _path_for(name)
    serialized = _normalize(actual)

    if update or not path.exists():
        GOLDEN_DIR.mkdir(parents=True, exist_ok=True)
        path.write_text(serialized, encoding="utf-8")
        if not update:
            raise AssertionError(
                f"Golden {name!r} did not exist; wrote it. "
                "Re-run the test to compare against the new snapshot."
            )
        return

    expected_raw = path.read_text(encoding="utf-8")
    expected = _normalize(json.loads(expected_raw))
    if serialized != expected:
        raise AssertionError(
            f"Response for {name!r} diverged from golden {path}.\n"
            "Set BB_UPDATE_GOLDEN=1 to refresh after intentional changes."
        )
