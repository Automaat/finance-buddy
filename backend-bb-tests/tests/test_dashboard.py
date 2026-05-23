"""Black-box tests for /api/dashboard — heavy pandas read endpoint."""

from __future__ import annotations

import httpx
import psycopg2
import pytest

from fixtures.seed import ACCOUNT_MARCIN_BANK, PERSONA_MARCIN


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


def _account_id_by_name(client: httpx.Client, name: str) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    for account in (*body["assets"], *body["liabilities"]):
        if account["name"] == name:
            return int(account["id"])
    raise AssertionError(f"Account {name!r} not found in /api/accounts response")


def _restore_account(dsn: str, account_id: int) -> None:
    with psycopg2.connect(dsn) as conn, conn.cursor() as cur:
        cur.execute("UPDATE accounts SET is_active = true WHERE id = %s", (account_id,))


def _dashboard_body(client: httpx.Client) -> dict:
    response = client.get("/api/dashboard")
    assert response.status_code == 200, response.text
    return response.json()


def _dashboard_totals(client: httpx.Client) -> tuple[float, float, float]:
    body = _dashboard_body(client)
    return (
        float(body["total_assets"]),
        float(body["total_liabilities"]),
        float(body["current_net_worth"]),
    )


def test_dashboard_date_range_filters_time_series(client: httpx.Client) -> None:
    # Seed has snapshots at 2025-11-30, 2025-12-31, 2026-01-31. A range that
    # only includes the middle snapshot should trim every time series to one
    # point per series — but leave snapshot tiles untouched.
    full = client.get("/api/dashboard")
    assert full.status_code == 200, full.text
    full_body = full.json()

    response = client.get(
        "/api/dashboard",
        params={"date_from": "2025-12-01", "date_to": "2025-12-31"},
    )
    assert response.status_code == 200, response.text
    body = response.json()

    history_dates = [p["date"] for p in body["net_worth_history"]]
    assert history_dates == ["2025-12-31"], history_dates

    for key in ("investment_time_series",):
        dates = [p["date"] for p in body[key]]
        assert dates == ["2025-12-31"], f"{key} dates={dates}"

    for wrapper in ("ike", "ikze", "ppk"):
        dates = [p["date"] for p in body["wrapper_time_series"][wrapper]]
        assert dates == ["2025-12-31"], f"wrapper {wrapper} dates={dates}"

    for category in ("stock", "bond"):
        dates = [p["date"] for p in body["category_time_series"][category]]
        assert dates == ["2025-12-31"], f"category {category} dates={dates}"

    # Tiles are unaffected.
    assert body["current_net_worth"] == full_body["current_net_worth"]
    assert body["total_assets"] == full_body["total_assets"]
    assert body["total_liabilities"] == full_body["total_liabilities"]


def test_dashboard_invalid_date_range_rejected(client: httpx.Client) -> None:
    response = client.get("/api/dashboard", params={"date_from": "not-a-date"})
    assert response.status_code == 400, response.text

    response = client.get(
        "/api/dashboard",
        params={"date_from": "2026-01-01", "date_to": "2024-01-01"},
    )
    assert response.status_code == 400, response.text


@pytest.mark.mutates
def test_history_preserved_after_soft_delete_account(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    # Regression for #394: soft-deleting an account must not retroactively
    # change historical net worth. The account's value still contributes to
    # every snapshot row it was already part of.
    _ = owner_ids[PERSONA_MARCIN]  # ensure seed is loaded
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    before = _dashboard_body(client)
    before_assets = float(before["total_assets"])
    before_liab = float(before["total_liabilities"])
    before_nw = float(before["current_net_worth"])
    before_history = before["net_worth_history"]

    response = client.delete(f"/api/accounts/{account_id}")
    assert response.status_code == 204, response.text
    try:
        after = _dashboard_body(client)
        after_assets = float(after["total_assets"])
        after_liab = float(after["total_liabilities"])
        after_nw = float(after["current_net_worth"])
        after_history = after["net_worth_history"]

        # /api/accounts hides the soft-deleted row.
        listing_resp = client.get("/api/accounts")
        assert listing_resp.status_code == 200, listing_resp.text
        listing = listing_resp.json()
        live_names = {a["name"] for a in (*listing["assets"], *listing["liabilities"])}
        assert ACCOUNT_MARCIN_BANK not in live_names

        # Aggregate totals across the full history must be unchanged — the
        # soft-deleted account still contributes to every snapshot it was
        # in before the delete.
        assert after_assets == pytest.approx(before_assets), (
            "total_assets shifted after soft-delete — history regressed"
        )
        assert after_liab == pytest.approx(before_liab)
        assert after_nw == pytest.approx(before_nw)
        assert len(after_history) == len(before_history)
        for b, a in zip(before_history, after_history, strict=True):
            assert a["date"] == b["date"]
            assert a["value"] == pytest.approx(b["value"])
    finally:
        _restore_account(database_url, account_id)
