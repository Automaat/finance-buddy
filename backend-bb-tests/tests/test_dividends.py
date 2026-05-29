"""Black-box tests for /api/holdings/dividends — CRUD plus the dividend
income folding into /api/investment/returns (issue #681)."""

from __future__ import annotations

import httpx
import pytest


def _investment_account_id(client: httpx.Client) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    for account in response.json().get("assets", []):
        if account.get("category") in ("stock", "etf", "bond", "fund"):
            return int(account["id"])
    raise AssertionError("no seeded investment account to attach a dividend to")


def _create_security(client: httpx.Client, request: pytest.FixtureRequest) -> int:
    symbol = f"DIV{abs(hash(request.node.name)) % 100000}"
    response = client.post(
        "/api/holdings/securities",
        json={
            "symbol": symbol,
            "name": "Dividend Test Co",
            "asset_type": "stock",
            "currency": "PLN",
        },
    )
    assert response.status_code == 201, response.text
    return int(response.json()["id"])


def test_dividend_crud_and_net(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    account_id = _investment_account_id(client)
    security_id = _create_security(client, request)
    dividend_id: int | None = None
    try:
        created = client.post(
            "/api/holdings/dividends",
            json={
                "account_id": account_id,
                "security_id": security_id,
                "pay_date": "2026-03-15",
                "gross_amount": "100.00",
                "withholding_tax": "19.00",
            },
        )
        assert created.status_code == 201, created.text
        body = created.json()
        dividend_id = int(body["id"])
        assert body["net_amount"] == "81.00"
        assert body["gross_amount"] == "100.00"
        assert body["withholding_tax"] == "19.00"

        listed = client.get(f"/api/holdings/dividends?security_id={security_id}")
        assert listed.status_code == 200, listed.text
        ids = {d["id"] for d in listed.json()["dividends"]}
        assert dividend_id in ids
    finally:
        if dividend_id is not None:
            assert client.delete(f"/api/holdings/dividends/{dividend_id}").status_code == 204
        client.delete(f"/api/holdings/securities/{security_id}")

    gone = client.get(f"/api/holdings/dividends?security_id={security_id}")
    assert gone.status_code == 200
    assert all(d["id"] != dividend_id for d in gone.json()["dividends"])


def test_dividend_validation_rejects_tax_above_gross(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    account_id = _investment_account_id(client)
    security_id = _create_security(client, request)
    try:
        response = client.post(
            "/api/holdings/dividends",
            json={
                "account_id": account_id,
                "security_id": security_id,
                "pay_date": "2026-03-15",
                "gross_amount": "50.00",
                "withholding_tax": "60.00",
            },
        )
        assert response.status_code == 422, response.text
    finally:
        client.delete(f"/api/holdings/securities/{security_id}")


def test_returns_folds_dividend_income(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    account_id = _investment_account_id(client)
    security_id = _create_security(client, request)
    dividend_id: int | None = None
    try:
        baseline = client.get(f"/api/investment/returns?scope=account&id={account_id}")
        assert baseline.status_code == 200, baseline.text
        base_net = baseline.json()["dividends_received_net"]

        created = client.post(
            "/api/holdings/dividends",
            json={
                "account_id": account_id,
                "security_id": security_id,
                "pay_date": "2026-01-15",
                "gross_amount": "200.00",
                "withholding_tax": "0.00",
            },
        )
        assert created.status_code == 201, created.text
        dividend_id = int(created.json()["id"])

        after = client.get(f"/api/investment/returns?scope=account&id={account_id}")
        assert after.status_code == 200, after.text
        assert after.json()["dividends_received_net"] == pytest.approx(base_net + 200.0)
    finally:
        if dividend_id is not None:
            client.delete(f"/api/holdings/dividends/{dividend_id}")
        client.delete(f"/api/holdings/securities/{security_id}")
