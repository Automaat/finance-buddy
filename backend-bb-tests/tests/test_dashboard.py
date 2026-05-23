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


def _dashboard_totals(client: httpx.Client) -> tuple[float, float, float]:
    response = client.get("/api/dashboard")
    assert response.status_code == 200, response.text
    body = response.json()
    return (
        float(body["total_assets"]),
        float(body["total_liabilities"]),
        float(body["current_net_worth"]),
    )


@pytest.mark.mutates
def test_history_preserved_after_soft_delete_account(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    # Regression for #394: soft-deleting an account must not retroactively
    # change historical net worth. The account's value still contributes to
    # every snapshot row it was already part of.
    _ = owner_ids[PERSONA_MARCIN]  # ensure seed is loaded
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    before_assets, before_liab, before_nw = _dashboard_totals(client)
    before_history = client.get("/api/dashboard").json()["net_worth_history"]

    response = client.delete(f"/api/accounts/{account_id}")
    assert response.status_code == 204, response.text
    try:
        after_assets, after_liab, after_nw = _dashboard_totals(client)
        after_history = client.get("/api/dashboard").json()["net_worth_history"]

        # /api/accounts hides the soft-deleted row.
        listing = client.get("/api/accounts").json()
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
