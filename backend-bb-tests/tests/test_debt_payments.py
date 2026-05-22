"""Black-box tests for /api/payments and /api/accounts/{id}/payments."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import ACCOUNT_MARCIN_BANK, ACCOUNT_MARCIN_MORTGAGE, PERSONA_MARCIN


def _account_id_by_name(client: httpx.Client, name: str) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    for account in (*body["assets"], *body["liabilities"]):
        if account["name"] == name:
            return int(account["id"])
    raise AssertionError(f"Account {name!r} not found in /api/accounts response")


@pytest.mark.golden
def test_list_all_payments_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/payments")
    assert response.status_code == 200, response.text
    assert_matches_golden("payments_list", response.json(), update=update_golden)


@pytest.mark.golden
def test_payment_counts_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/payments/counts")
    assert response.status_code == 200, response.text
    assert_matches_golden("payments_counts", response.json(), update=update_golden)


def test_get_account_payments_happy_path(client: httpx.Client) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_MORTGAGE)
    response = client.get(f"/api/accounts/{account_id}/payments")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["payment_count"] >= 2
    assert all(p["account_id"] == account_id for p in body["payments"])


def test_get_account_payments_invalid_account_type(client: httpx.Client) -> None:
    # Bank accounts are not liabilities — service returns 400.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    response = client.get(f"/api/accounts/{account_id}/payments")
    assert response.status_code == 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_create_payment_happy_path(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_MORTGAGE)
    payload = {
        "amount": 1800.0,
        "date": "2025-05-15",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    created_id: int | None = None
    try:
        response = client.post(f"/api/accounts/{account_id}/payments", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["account_id"] == account_id
        assert body["amount"] == pytest.approx(1800.0)
        assert body["date"] == "2025-05-15"
        assert body["owner_user_id"] == owner_ids[PERSONA_MARCIN]
    finally:
        if created_id is not None:
            client.delete(f"/api/accounts/{account_id}/payments/{created_id}")


def test_create_payment_validation_error(client: httpx.Client) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_MORTGAGE)
    payload = {
        "amount": -50.0,
        "date": "2025-05-20",
        "owner_user_id": None,
    }
    response = client.post(f"/api/accounts/{account_id}/payments", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_delete_payment_happy_path(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_MORTGAGE)
    create_resp = client.post(
        f"/api/accounts/{account_id}/payments",
        json={
            "amount": 999.99,
            "date": "2025-07-15",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
        },
    )
    assert create_resp.status_code == 201, create_resp.text
    payment_id = create_resp.json()["id"]

    response = client.delete(f"/api/accounts/{account_id}/payments/{payment_id}")
    assert response.status_code == 204, response.text
