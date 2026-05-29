"""Negative-path black-box coverage for mutating endpoints (issue #682).

Every case asserts a 4xx (validation or not-found) and that the body is JSON.
All requests are designed to fail before any write, so no @pytest.mark.mutates
marker is needed.
"""

from __future__ import annotations

import httpx


def _first_account_id(client: httpx.Client) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    assets = response.json().get("assets", [])
    assert assets, "seed must provide at least one asset account"
    return int(assets[0]["id"])


MISSING_ID = 999_999_999


def _assert_4xx_json(response: httpx.Response, *expected: int) -> None:
    assert response.status_code in expected, f"{response.status_code}: {response.text}"
    # Error bodies are always JSON (detail string or Pydantic-shaped list).
    response.json()


def test_create_account_invalid_returns_422(client: httpx.Client) -> None:
    response = client.post(
        "/api/accounts",
        json={"name": "", "type": "asset", "category": "bank", "owner_user_id": 1},
    )
    _assert_4xx_json(response, 400, 422)


def test_update_account_invalid_wrapper_returns_422(client: httpx.Client) -> None:
    account_id = _first_account_id(client)
    response = client.put(f"/api/accounts/{account_id}", json={"account_wrapper": "BOGUS"})
    _assert_4xx_json(response, 400, 422)


def test_update_missing_account_returns_404(client: httpx.Client) -> None:
    response = client.put(f"/api/accounts/{MISSING_ID}", json={"name": "x"})
    _assert_4xx_json(response, 404)


def test_delete_missing_account_returns_404(client: httpx.Client) -> None:
    response = client.delete(f"/api/accounts/{MISSING_ID}")
    _assert_4xx_json(response, 404)


def test_create_transaction_empty_body_returns_422(client: httpx.Client) -> None:
    account_id = _first_account_id(client)
    response = client.post(f"/api/accounts/{account_id}/transactions", json={})
    _assert_4xx_json(response, 400, 422)


def test_create_transaction_missing_account_returns_404(client: httpx.Client) -> None:
    response = client.post(
        f"/api/accounts/{MISSING_ID}/transactions",
        json={"amount": 10, "date": "2026-01-01", "owner_user_id": 1},
    )
    _assert_4xx_json(response, 404)


def test_delete_missing_transaction_returns_404(client: httpx.Client) -> None:
    account_id = _first_account_id(client)
    response = client.delete(f"/api/accounts/{account_id}/transactions/{MISSING_ID}")
    _assert_4xx_json(response, 404)


def test_create_snapshot_without_values_returns_422(client: httpx.Client) -> None:
    response = client.post("/api/snapshots", json={"date": "2026-02-28", "notes": "x"})
    _assert_4xx_json(response, 400, 422)


def test_create_bond_invalid_type_returns_422(client: httpx.Client) -> None:
    response = client.post(
        "/api/bonds",
        json={
            "type": "ZZZ",
            "series": "X",
            "face_value": 100,
            "purchase_date": "2026-01-01",
            "first_year_rate": 5,
            "margin": 1,
        },
    )
    _assert_4xx_json(response, 400, 422)


def test_get_missing_bond_returns_404(client: httpx.Client) -> None:
    _assert_4xx_json(client.get(f"/api/bonds/{MISSING_ID}"), 404)


def test_delete_missing_bond_returns_404(client: httpx.Client) -> None:
    _assert_4xx_json(client.delete(f"/api/bonds/{MISSING_ID}"), 404)
