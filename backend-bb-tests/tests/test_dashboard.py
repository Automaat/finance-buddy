"""Black-box tests for /api/dashboard — heavy pandas read endpoint."""

from __future__ import annotations

import httpx


def test_get_dashboard_returns_documented_shape(client: httpx.Client) -> None:
    response = client.get("/api/dashboard")
    assert response.status_code == 200, response.text
    body = response.json()
    required = {
        "net_worth_history",
        "current_net_worth",
        "change_vs_last_month",
        "total_assets",
        "total_liabilities",
        "allocation",
        "retirement_account_value",
        "metric_cards",
        "allocation_analysis",
        "investment_time_series",
        "wrapper_time_series",
        "category_time_series",
        "tile_deltas",
    }
    missing = required - body.keys()
    assert not missing, f"Dashboard response is missing fields: {sorted(missing)}"


def test_dashboard_nested_keys_present(client: httpx.Client) -> None:
    response = client.get("/api/dashboard")
    assert response.status_code == 200, response.text
    body = response.json()

    metric_card_keys = {
        "property_sqm",
        "emergency_fund_months",
        "retirement_income_monthly",
        "mortgage_remaining",
        "mortgage_months_left",
        "retirement_total",
        "investment_contributions",
        "investment_returns",
    }
    assert metric_card_keys.issubset(body["metric_cards"].keys())

    allocation_analysis_keys = {
        "by_category",
        "by_wrapper",
        "rebalancing",
        "total_investment_value",
    }
    assert allocation_analysis_keys.issubset(body["allocation_analysis"].keys())

    tile_delta_keys = {"net_worth", "assets", "liabilities"}
    assert tile_delta_keys.issubset(body["tile_deltas"].keys())

    wrapper_ts_keys = {"ike", "ikze", "ppk"}
    assert wrapper_ts_keys.issubset(body["wrapper_time_series"].keys())

    category_ts_keys = {"stock", "bond"}
    assert category_ts_keys.issubset(body["category_time_series"].keys())
