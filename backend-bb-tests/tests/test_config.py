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
