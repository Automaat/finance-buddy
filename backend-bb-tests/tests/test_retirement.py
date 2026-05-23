"""Black-box tests for /api/retirement — stats, PPK, limits, generation."""

from __future__ import annotations

from collections.abc import Iterator
from contextlib import closing, contextmanager

import httpx
import psycopg2
import pytest

from fixtures.seed import PERSONA_MARCIN


@pytest.mark.golden
def test_get_retirement_stats_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/retirement/stats", params={"year": 2025})
    assert response.status_code == 200, response.text
    assert_matches_golden("retirement_stats_2025", response.json(), update=update_golden)


def test_get_retirement_stats_shape(client: httpx.Client) -> None:
    response = client.get("/api/retirement/stats", params={"year": 2025})
    assert response.status_code == 200, response.text
    body = response.json()
    assert isinstance(body, list)
    if body:
        required = {
            "year",
            "account_wrapper",
            "owner_user_id",
            "total_contributed",
            "employee_contributed",
            "employer_contributed",
        }
        assert required.issubset(body[0].keys())


@pytest.mark.golden
def test_get_ppk_stats_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/retirement/ppk-stats")
    assert response.status_code == 200, response.text
    assert_matches_golden("retirement_ppk_stats", response.json(), update=update_golden)


def test_get_ppk_stats_shape(client: httpx.Client) -> None:
    response = client.get("/api/retirement/ppk-stats")
    assert response.status_code == 200, response.text
    body = response.json()
    assert isinstance(body, list)


@pytest.mark.golden
def test_get_limits_for_year_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/retirement/limits/2025")
    assert response.status_code == 200, response.text
    assert_matches_golden("retirement_limits_2025", response.json(), update=update_golden)


def test_get_limits_for_year_shape(client: httpx.Client) -> None:
    response = client.get("/api/retirement/limits/2025")
    assert response.status_code == 200, response.text
    body = response.json()
    assert isinstance(body, list)
    assert body, "Seeded retirement_limits is empty"
    required = {"id", "year", "account_wrapper", "owner_user_id", "limit_amount"}
    assert required.issubset(body[0].keys())


def test_put_retirement_limit_upsert(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    # Idempotent upsert — picking a year we own the row for so this is safe
    # to re-run. Use 2025/IKE/Marcin which is seeded.
    marcin = owner_ids[PERSONA_MARCIN]
    payload = {
        "year": 2025,
        "account_wrapper": "IKE",
        "owner_user_id": marcin,
        "limit_amount": 23472.0,
        "notes": "Black-box upsert test",
    }
    response = client.put(
        f"/api/retirement/limits/2025/IKE/{marcin}",
        json=payload,
    )
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["year"] == 2025
    assert body["account_wrapper"] == "IKE"
    assert body["owner_user_id"] == marcin
    assert body["limit_amount"] == 23472.0


def test_put_retirement_limit_validation_error(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    # year out of allowed range triggers validator
    marcin = owner_ids[PERSONA_MARCIN]
    payload = {
        "year": 1999,
        "account_wrapper": "IKE",
        "owner_user_id": marcin,
        "limit_amount": 1000.0,
    }
    response = client.put(
        f"/api/retirement/limits/1999/IKE/{marcin}",
        json=payload,
    )
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_post_ppk_contributions_validation_error(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    # Invalid month → 422
    payload = {"owner_user_id": owner_ids[PERSONA_MARCIN], "month": 13, "year": 2025}
    response = client.post("/api/retirement/ppk-contributions/generate", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def _ppk_account_id(client: httpx.Client) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    for account in response.json()["assets"]:
        if account["account_wrapper"] == "PPK":
            return int(account["id"])
    raise AssertionError("No PPK account found in /api/accounts response")


@contextmanager
def _clean_ppk_account(dsn: str, account_id: int) -> Iterator[None]:
    """Soft-delete the account's existing transactions for the test, then
    restore them so the session-scoped seed is intact for downstream
    golden tests. Newly-inserted rows during the test are hard-deleted on
    exit."""
    with closing(psycopg2.connect(dsn)) as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id FROM transactions WHERE account_id = %s AND is_active = true",
                (account_id,),
            )
            existing_ids = [row[0] for row in cur.fetchall()]
            cur.execute(
                "UPDATE transactions SET is_active = false WHERE id = ANY(%s)",
                (existing_ids,),
            )
        conn.commit()
        try:
            yield
        finally:
            with conn.cursor() as cur:
                cur.execute(
                    "DELETE FROM transactions WHERE account_id = %s AND id <> ALL(%s)",
                    (account_id, existing_ids),
                )
                cur.execute(
                    "UPDATE transactions SET is_active = true WHERE id = ANY(%s)",
                    (existing_ids,),
                )
            conn.commit()


@pytest.mark.mutates
def test_ppk_generation_includes_welcome_subsidy_when_opted_in(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _ppk_account_id(client)
    # Seed has a prior 'government' transaction on this PPK account; the
    # context manager soft-deletes it for the test (so welcome eligibility
    # fires) and restores it after, so the golden suite isn't disturbed.
    with _clean_ppk_account(database_url, account_id):
        payload = {
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "month": 3,
            "year": 2025,
            "include_welcome_subsidy": True,
        }
        response = client.post("/api/retirement/ppk-contributions/generate", json=payload)
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["welcome_applied"] is True
        assert body["government_amount"] == pytest.approx(250.0)
        assert body["total_amount"] == pytest.approx(
            body["employee_amount"] + body["employer_amount"] + body["government_amount"]
        )
        # Employee + employer + welcome government row.
        assert len(body["transactions_created"]) >= 3


@pytest.mark.mutates
def test_ppk_welcome_is_idempotent_across_months(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _ppk_account_id(client)
    with _clean_ppk_account(database_url, account_id):
        base = {
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "year": 2025,
            "include_welcome_subsidy": True,
        }
        first = client.post("/api/retirement/ppk-contributions/generate", json={**base, "month": 4})
        assert first.status_code == 200, first.text
        assert first.json()["welcome_applied"] is True

        # Second call (different month) still requests the welcome; the store
        # must skip it because a government txn already exists.
        second = client.post(
            "/api/retirement/ppk-contributions/generate", json={**base, "month": 5}
        )
        assert second.status_code == 200, second.text
        assert second.json()["welcome_applied"] is False
        assert second.json()["government_amount"] == pytest.approx(0.0)


@pytest.mark.mutates
def test_ppk_combined_welcome_and_annual_in_one_call(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    # Both subsidies in a single generate call must both land. Welcome and
    # annual share transaction_type='government', so the eligibility check
    # must distinguish them (by amount) — otherwise the welcome inserted
    # first masks the annual check.
    account_id = _ppk_account_id(client)
    with _clean_ppk_account(database_url, account_id):
        response = client.post(
            "/api/retirement/ppk-contributions/generate",
            json={
                "owner_user_id": owner_ids[PERSONA_MARCIN],
                "month": 8,
                "year": 2025,
                "include_welcome_subsidy": True,
                "include_annual_subsidy": True,
            },
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["welcome_applied"] is True
        assert body["annual_applied"] is True
        assert body["government_amount"] == pytest.approx(250.0 + 240.0)
        # Employee + employer + welcome + annual.
        assert len(body["transactions_created"]) == 4


@pytest.mark.mutates
def test_ppk_annual_subsidy_idempotent_within_year(
    client: httpx.Client, database_url: str, owner_ids: dict[str, int]
) -> None:
    account_id = _ppk_account_id(client)
    with _clean_ppk_account(database_url, account_id):
        base = {
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "year": 2025,
            "include_annual_subsidy": True,
        }
        first = client.post("/api/retirement/ppk-contributions/generate", json={**base, "month": 6})
        assert first.status_code == 200, first.text
        assert first.json()["annual_applied"] is True
        assert first.json()["government_amount"] == pytest.approx(240.0)

        second = client.post(
            "/api/retirement/ppk-contributions/generate", json={**base, "month": 7}
        )
        assert second.status_code == 200, second.text
        assert second.json()["annual_applied"] is False
        assert second.json()["government_amount"] == pytest.approx(0.0)
