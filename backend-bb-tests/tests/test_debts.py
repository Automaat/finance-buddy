"""Black-box tests for /api/debts and /api/accounts/{id}/debts."""

from __future__ import annotations

import httpx
import psycopg2
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


def _hard_delete_account_and_debt(dsn: str, account_id: int) -> None:
    """Tear down a test-created liability account and any debts/payments on it."""
    with psycopg2.connect(dsn) as conn, conn.cursor() as cur:
        cur.execute("DELETE FROM debt_payments WHERE account_id = %s", (account_id,))
        cur.execute("DELETE FROM debts WHERE account_id = %s", (account_id,))
        cur.execute("DELETE FROM snapshot_values WHERE account_id = %s", (account_id,))
        cur.execute("DELETE FROM accounts WHERE id = %s", (account_id,))


def _create_temp_liability_account(client: httpx.Client, name: str, owner_id: int) -> int:
    payload = {
        "name": name,
        "type": "liability",
        "category": "installment",
        "owner_user_id": owner_id,
        "currency": "PLN",
        "purpose": "general",
        "receives_contributions": False,
    }
    response = client.post("/api/accounts", json=payload)
    assert response.status_code == 201, response.text
    return int(response.json()["id"])


@pytest.mark.golden
def test_list_debts_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/debts")
    assert response.status_code == 200, response.text
    assert_matches_golden("debts_list", response.json(), update=update_golden)


def test_list_debts_returns_seeded(client: httpx.Client) -> None:
    response = client.get("/api/debts")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["total_count"] >= 1
    names = {d["name"] for d in body["debts"]}
    assert "Apartment Mortgage" in names


def test_get_debt_by_id(client: httpx.Client) -> None:
    listing = client.get("/api/debts")
    assert listing.status_code == 200, listing.text
    debt_id = listing.json()["debts"][0]["id"]

    response = client.get(f"/api/debts/{debt_id}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == debt_id
    assert body["name"] == "Apartment Mortgage"


def test_get_debt_not_found(client: httpx.Client) -> None:
    response = client.get("/api/debts/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_create_debt_happy_path(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _create_temp_liability_account(
        client, "BB Temp Debt Account", owner_ids[PERSONA_MARCIN]
    )
    try:
        payload = {
            "name": "BB Test Loan",
            "debt_type": "installment_0percent",
            "start_date": "2024-01-01",
            "initial_amount": 5000.0,
            "interest_rate": 0.0,
            "currency": "PLN",
            "notes": "bb-create-happy",
        }
        response = client.post(f"/api/accounts/{account_id}/debts", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        assert body["name"] == "BB Test Loan"
        assert body["account_id"] == account_id
        assert body["debt_type"] == "installment_0percent"
        assert body["initial_amount"] == pytest.approx(5000.0)
    finally:
        _hard_delete_account_and_debt(database_url, account_id)


def test_create_debt_validation_error(client: httpx.Client) -> None:
    # Bank account is an asset, not liability — service returns 400.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    payload = {
        "name": "Invalid Debt",
        "debt_type": "mortgage",
        "start_date": "2024-01-01",
        "initial_amount": 1000.0,
        "interest_rate": 5.0,
        "currency": "PLN",
    }
    response = client.post(f"/api/accounts/{account_id}/debts", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_update_debt_happy_path(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _create_temp_liability_account(
        client, "BB Temp Update Account", owner_ids[PERSONA_MARCIN]
    )
    create_resp = client.post(
        f"/api/accounts/{account_id}/debts",
        json={
            "name": "Original Name",
            "debt_type": "installment_0percent",
            "start_date": "2024-02-01",
            "initial_amount": 2000.0,
            "interest_rate": 0.0,
            "currency": "PLN",
        },
    )
    assert create_resp.status_code == 201, create_resp.text
    debt_id = create_resp.json()["id"]

    try:
        response = client.put(
            f"/api/debts/{debt_id}",
            json={"name": "Updated Name", "interest_rate": 5.5},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == debt_id
        assert body["name"] == "Updated Name"
        assert body["interest_rate"] == pytest.approx(5.5)
    finally:
        _hard_delete_account_and_debt(database_url, account_id)


def test_update_debt_validation_error(client: httpx.Client) -> None:
    listing = client.get("/api/debts")
    assert listing.status_code == 200, listing.text
    debt_id = listing.json()["debts"][0]["id"]

    response = client.put(f"/api/debts/{debt_id}", json={"interest_rate": -1.0})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_delete_debt_happy_path(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _create_temp_liability_account(
        client, "BB Temp Delete Account", owner_ids[PERSONA_MARCIN]
    )
    create_resp = client.post(
        f"/api/accounts/{account_id}/debts",
        json={
            "name": "To Delete",
            "debt_type": "installment_0percent",
            "start_date": "2024-03-01",
            "initial_amount": 500.0,
            "interest_rate": 0.0,
            "currency": "PLN",
        },
    )
    assert create_resp.status_code == 201, create_resp.text
    debt_id = create_resp.json()["id"]

    try:
        response = client.delete(f"/api/debts/{debt_id}")
        assert response.status_code == 204, response.text
    finally:
        _hard_delete_account_and_debt(database_url, account_id)


def test_seeded_mortgage_account_exists(client: httpx.Client) -> None:
    # Sanity check — the mortgage account underpins the seeded debt.
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_MORTGAGE)
    assert account_id > 0


def _account_exists(dsn: str, name: str) -> bool:
    with psycopg2.connect(dsn) as conn, conn.cursor() as cur:
        cur.execute("SELECT 1 FROM accounts WHERE name = %s AND is_active = true", (name,))
        return cur.fetchone() is not None


@pytest.mark.mutates
def test_create_debt_atomic_happy_path(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    name = "BB Atomic Debt Account"
    payload = {
        "name": name,
        "debt_type": "mortgage",
        "start_date": "2024-04-01",
        "initial_amount": 250000.0,
        "interest_rate": 6.5,
        "currency": "PLN",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    response = client.post("/api/debts", json=payload)
    assert response.status_code == 201, response.text
    body = response.json()
    assert body["name"] == name
    assert body["debt_type"] == "mortgage"
    assert body["account_id"] > 0
    assert body["account_owner_user_id"] == owner_ids[PERSONA_MARCIN]

    try:
        # Account should exist and be wired to the new debt.
        listing = client.get(f"/api/debts?account_id={body['account_id']}")
        assert listing.status_code == 200, listing.text
        debts = listing.json()["debts"]
        assert len(debts) == 1
        assert debts[0]["id"] == body["id"]
    finally:
        _hard_delete_account_and_debt(database_url, body["account_id"])


@pytest.mark.mutates
def test_create_debt_atomic_rolls_back_on_validation_failure(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    # Negative interest_rate is rejected by debt validation. The account
    # insert must not be persisted — no orphan account should remain.
    name = "BB Atomic Rollback Account"
    payload = {
        "name": name,
        "debt_type": "mortgage",
        "start_date": "2024-04-01",
        "initial_amount": 100000.0,
        "interest_rate": -1.0,
        "currency": "PLN",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    response = client.post("/api/debts", json=payload)
    assert response.status_code >= 400, response.text
    assert not _account_exists(database_url, name)


@pytest.mark.mutates
def test_create_debt_atomic_rolls_back_on_bad_owner(
    client: httpx.Client, database_url: str
) -> None:
    # owner_user_id 9_999_999 doesn't exist — the FK violation fires inside
    # the atomic transaction, exercising the deferred rollback path.
    name = "BB Atomic Bad Owner Account"
    payload = {
        "name": name,
        "debt_type": "mortgage",
        "start_date": "2024-04-01",
        "initial_amount": 100000.0,
        "interest_rate": 5.0,
        "currency": "PLN",
        "owner_user_id": 9_999_999,
    }
    response = client.post("/api/debts", json=payload)
    assert response.status_code >= 400, response.text
    assert not _account_exists(database_url, name)


@pytest.mark.mutates
def test_create_debt_atomic_duplicate_account_name(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    name = "BB Atomic Duplicate Account"
    payload = {
        "name": name,
        "debt_type": "installment_0percent",
        "start_date": "2024-04-01",
        "initial_amount": 1000.0,
        "interest_rate": 0.0,
        "currency": "PLN",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    first = client.post("/api/debts", json=payload)
    assert first.status_code == 201, first.text
    account_id = first.json()["account_id"]
    try:
        second = client.post("/api/debts", json=payload)
        assert second.status_code == 409, second.text
    finally:
        _hard_delete_account_and_debt(database_url, account_id)
