"""Black-box tests for /api/investment — stock and bond aggregates."""

from __future__ import annotations

import httpx
import pytest


@pytest.mark.golden
def test_get_stock_stats_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/investment/stock-stats")
    assert response.status_code == 200, response.text
    assert_matches_golden("investment_stock_stats", response.json(), update=update_golden)


def test_get_stock_stats_shape(client: httpx.Client) -> None:
    response = client.get("/api/investment/stock-stats")
    assert response.status_code == 200, response.text
    body = response.json()
    required = {"category", "total_value", "total_contributed", "returns", "roi_percentage"}
    assert required.issubset(body.keys())
    assert body["category"] == "stock"


@pytest.mark.golden
def test_get_bond_stats_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/investment/bond-stats")
    assert response.status_code == 200, response.text
    assert_matches_golden("investment_bond_stats", response.json(), update=update_golden)


def test_get_bond_stats_shape(client: httpx.Client) -> None:
    response = client.get("/api/investment/bond-stats")
    assert response.status_code == 200, response.text
    body = response.json()
    required = {"category", "total_value", "total_contributed", "returns", "roi_percentage"}
    assert required.issubset(body.keys())
    assert body["category"] == "bond"
