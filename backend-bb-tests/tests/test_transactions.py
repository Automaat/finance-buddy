"""Black-box tests for /api/transactions and /api/accounts/{id}/transactions."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import ACCOUNT_MARCIN_BANK, ACCOUNT_MARCIN_IKE, PERSONA_MARCIN


def _account_id_by_name(client: httpx.Client, name: str) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    for account in (*body["assets"], *body["liabilities"]):
        if account["name"] == name:
            return int(account["id"])
    raise AssertionError(f"Account {name!r} not found in /api/accounts response")


@pytest.mark.golden
def test_list_all_transactions_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/transactions")
    assert response.status_code == 200, response.text
    payload = response.json()
    # created_at is the stable SEED_CREATED_AT marker — safe to golden as-is.
    assert_matches_golden("transactions_list", payload, update=update_golden)


@pytest.mark.golden
def test_transaction_counts_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/transactions/counts")
    assert response.status_code == 200, response.text
    assert_matches_golden("transactions_counts", response.json(), update=update_golden)


def test_get_account_transactions_happy_path(client: httpx.Client) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    response = client.get(f"/api/accounts/{account_id}/transactions")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["transaction_count"] >= 1
    assert all(t["account_id"] == account_id for t in body["transactions"])


def test_get_account_transactions_invalid_account_type(client: httpx.Client) -> None:
    # Bank accounts cannot have transactions — service returns 400.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    response = client.get(f"/api/accounts/{account_id}/transactions")
    assert response.status_code == 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_create_transaction_happy_path(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    payload = {
        "amount": 750.0,
        "date": "2025-06-15",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "transaction_type": "employee",
    }
    created_id: int | None = None
    try:
        response = client.post(f"/api/accounts/{account_id}/transactions", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["account_id"] == account_id
        assert body["amount"] == pytest.approx(750.0)
        assert body["date"] == "2025-06-15"
        assert body["owner_user_id"] == owner_ids[PERSONA_MARCIN]
        assert body["transaction_type"] == "employee"
    finally:
        if created_id is not None:
            client.delete(f"/api/accounts/{account_id}/transactions/{created_id}")


def test_create_transaction_validation_error(client: httpx.Client) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    # Negative amount fails validate_positive_amount.
    payload = {
        "amount": -100.0,
        "date": "2025-06-20",
        "owner_user_id": None,
        "transaction_type": "employee",
    }
    response = client.post(f"/api/accounts/{account_id}/transactions", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def _post_and_track(
    client: httpx.Client, account_id: int, payload: dict, created_ids: list[int]
) -> httpx.Response:
    resp = client.post(f"/api/accounts/{account_id}/transactions", json=payload)
    if resp.status_code == 201:
        created_ids.append(resp.json()["id"])
    return resp


@pytest.mark.mutates
def test_create_two_same_day_transactions_persist(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    # Regression for #396: multiple active transactions may share
    # (account_id, date) — e.g. same-day fees, dividends, split imports.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    base = {
        "date": "2025-09-12",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    created_ids: list[int] = []
    try:
        first = _post_and_track(
            client,
            account_id,
            {**base, "amount": 100.0, "transaction_type": "employee"},
            created_ids,
        )
        assert first.status_code == 201, first.text
        second = _post_and_track(
            client,
            account_id,
            {**base, "amount": 250.0, "transaction_type": "employer"},
            created_ids,
        )
        assert second.status_code == 201, second.text
        listing = client.get(f"/api/accounts/{account_id}/transactions")
        assert listing.status_code == 200, listing.text
        same_day = [t for t in listing.json()["transactions"] if t["date"] == "2025-09-12"]
        amounts = sorted(t["amount"] for t in same_day)
        assert amounts == [pytest.approx(100.0), pytest.approx(250.0)]
    finally:
        for tid in created_ids:
            client.delete(f"/api/accounts/{account_id}/transactions/{tid}")


@pytest.mark.mutates
def test_create_two_same_day_same_type_transactions_persist(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    # Same (account_id, date) and same transaction_type — e.g. two same-day
    # fees or two employee contributions split across imports.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    base = {
        "date": "2025-09-19",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "transaction_type": "employee",
    }
    created_ids: list[int] = []
    try:
        first = _post_and_track(client, account_id, {**base, "amount": 50.0}, created_ids)
        assert first.status_code == 201, first.text
        second = _post_and_track(client, account_id, {**base, "amount": 75.0}, created_ids)
        assert second.status_code == 201, second.text
        assert first.json()["id"] != second.json()["id"]
    finally:
        for tid in created_ids:
            client.delete(f"/api/accounts/{account_id}/transactions/{tid}")


@pytest.mark.mutates
def test_delete_transaction_happy_path(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_IKE)
    create_resp = client.post(
        f"/api/accounts/{account_id}/transactions",
        json={
            "amount": 123.45,
            "date": "2025-08-15",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "transaction_type": "employee",
        },
    )
    assert create_resp.status_code == 201, create_resp.text
    transaction_id = create_resp.json()["id"]

    response = client.delete(f"/api/accounts/{account_id}/transactions/{transaction_id}")
    assert response.status_code == 204, response.text
