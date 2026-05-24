"""Smoke tests for /api/scenarios — create, list, clone, delete round-trip."""

from __future__ import annotations

import httpx


def test_scenarios_round_trip(client: httpx.Client) -> None:
    # Start empty.
    r = client.get("/api/scenarios?kind=retirement")
    assert r.status_code == 200, r.text
    assert isinstance(r.json()["scenarios"], list)

    inputs = {
        "current_age": 35,
        "retirement_age": 65,
        "ike_ikze_accounts": [],
        "ppk_accounts": [],
        "brokerage_accounts": [],
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
        "expected_salary_growth": 3.0,
        "inflation_rate": 3.0,
    }

    # Create.
    r = client.post(
        "/api/scenarios",
        json={"name": "Plan A", "kind": "retirement", "inputs_json": inputs},
    )
    assert r.status_code == 201, r.text
    created = r.json()
    assert created["name"] == "Plan A"
    assert created["kind"] == "retirement"
    assert created["inputs_json"]["current_age"] == 35
    sid = created["id"]

    # Clone (no override name → " (copy)" suffix).
    r = client.post(f"/api/scenarios/{sid}/clone", json={})
    assert r.status_code == 201, r.text
    clone = r.json()
    assert clone["name"] == "Plan A (copy)"
    assert clone["inputs_json"] == inputs
    cid = clone["id"]

    # List now has both, newest first by updated_at.
    r = client.get("/api/scenarios?kind=retirement")
    assert r.status_code == 200, r.text
    names = [s["name"] for s in r.json()["scenarios"]]
    assert "Plan A" in names
    assert "Plan A (copy)" in names

    # Clone with custom name.
    r = client.post(f"/api/scenarios/{sid}/clone", json={"name": "Plan B"})
    assert r.status_code == 201, r.text
    assert r.json()["name"] == "Plan B"
    bid = r.json()["id"]

    # Delete all three so subsequent runs start clean.
    for delete_id in (sid, cid, bid):
        r = client.delete(f"/api/scenarios/{delete_id}")
        assert r.status_code == 204, r.text


def test_scenarios_validation_rejects_bad_inputs(client: httpx.Client) -> None:
    cases = [
        ({"name": "", "kind": "retirement", "inputs_json": {"a": 1}}, "name"),
        ({"name": "x", "kind": "bogus", "inputs_json": {"a": 1}}, "kind"),
        ({"name": "x", "kind": "retirement", "inputs_json": []}, "inputs_json"),
        ({"name": "x", "kind": "retirement", "inputs_json": None}, "inputs_json"),
    ]
    for body, want_field in cases:
        r = client.post("/api/scenarios", json=body)
        assert r.status_code == 422, r.text
        assert r.json()["detail"][0]["loc"][1] == want_field


def test_scenarios_get_unknown_returns_404(client: httpx.Client) -> None:
    r = client.get("/api/scenarios/999999")
    assert r.status_code == 404, r.text
    r = client.post("/api/scenarios/999999/clone", json={})
    assert r.status_code == 404, r.text
    r = client.delete("/api/scenarios/999999")
    assert r.status_code == 404, r.text
