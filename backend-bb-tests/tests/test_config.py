"""Smoke tests for /api/config — proves seed + GET work end-to-end."""

from __future__ import annotations

import httpx
import pytest


@pytest.mark.golden
def test_get_config_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/config")
    assert response.status_code == 200, response.text
    assert_matches_golden("config_get", response.json(), update=update_golden)


def test_get_config_required_fields(client: httpx.Client) -> None:
    response = client.get("/api/config")
    assert response.status_code == 200
    body = response.json()
    required = {
        "birth_date",
        "retirement_age",
        "retirement_monthly_salary",
        "allocation_real_estate",
        "allocation_stocks",
        "allocation_bonds",
        "allocation_gold",
        "allocation_commodities",
        "monthly_expenses",
        "monthly_mortgage_payment",
    }
    missing = required - body.keys()
    assert not missing, f"Config response is missing fields: {sorted(missing)}"


def test_put_config_round_trip(client: httpx.Client) -> None:
    # Round-trips the full config payload back through PUT. Regression guard
    # against placeholder/column-list drift in the upsert SQL — if any column
    # is missing from either the INSERT list, VALUES list, or RETURNING/scan,
    # Postgres errors and this fails before the parity suite even runs.
    #
    # Seed allocations don't sum to 100% market, so we satisfy the validator
    # by overriding them; the assertions below cover only the new FIRE fields.
    body = client.get("/api/config").json()
    body["allocation_stocks"] = 60
    body["allocation_bonds"] = 30
    body["allocation_gold"] = 5
    body["allocation_commodities"] = 5
    body["barista_monthly_income"] = "3000.00"
    body["coast_fire_target_age"] = 65
    body["expected_return_rate"] = "0.08"
    response = client.put("/api/config", json=body)
    assert response.status_code == 200, response.text
    out = response.json()
    assert out["barista_monthly_income"] == "3000.00"
    assert out["coast_fire_target_age"] == 65
    assert out["expected_return_rate"] == "0.0800"
