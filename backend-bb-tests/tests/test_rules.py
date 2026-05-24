"""Smoke tests for /api/rules — Polish constants metadata endpoint."""

from __future__ import annotations

import re

import httpx


def test_rules_returns_metadata_shape(client: httpx.Client) -> None:
    r = client.get("/api/rules")
    assert r.status_code == 200, r.text
    rules = r.json().get("rules", [])
    assert len(rules) > 0
    required = {
        "key",
        "name",
        "category",
        "value",
        "unit",
        "year",
        "effective_date",
        "source_url",
        "last_checked_date",
        "description",
    }
    for row in rules:
        missing = required - row.keys()
        assert not missing, f"rule {row.get('key')!r} missing {sorted(missing)}"
        # Source URL must be an https link, never blank.
        assert row["source_url"].startswith("https://"), row
        # Dates must be naive YYYY-MM-DD.
        assert re.fullmatch(r"\d{4}-\d{2}-\d{2}", row["effective_date"]), row
        assert re.fullmatch(r"\d{4}-\d{2}-\d{2}", row["last_checked_date"]), row


def test_rules_filter_by_category(client: httpx.Client) -> None:
    r = client.get("/api/rules", params={"category": "ike_limit"})
    assert r.status_code == 200, r.text
    rules = r.json()["rules"]
    assert len(rules) >= 1
    for row in rules:
        assert row["category"] == "ike_limit"


def test_rules_unknown_category_returns_empty(client: httpx.Client) -> None:
    r = client.get("/api/rules", params={"category": "bogus"})
    assert r.status_code == 200, r.text
    assert r.json()["rules"] == []
