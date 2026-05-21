"""Black-box tests for /api/retirement — stats, PPK, limits, generation."""

from __future__ import annotations

import httpx
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
            "owner",
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
    required = {"id", "year", "account_wrapper", "owner", "limit_amount"}
    assert required.issubset(body[0].keys())


def test_put_retirement_limit_upsert(client: httpx.Client) -> None:
    # Idempotent upsert — picking a year we own the row for so this is safe
    # to re-run. Use 2025/IKE/Marcin which is seeded.
    payload = {
        "year": 2025,
        "account_wrapper": "IKE",
        "owner": PERSONA_MARCIN,
        "limit_amount": 23472.0,
        "notes": "Black-box upsert test",
    }
    response = client.put(
        f"/api/retirement/limits/2025/IKE/{PERSONA_MARCIN}",
        json=payload,
    )
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["year"] == 2025
    assert body["account_wrapper"] == "IKE"
    assert body["owner"] == PERSONA_MARCIN
    assert body["limit_amount"] == 23472.0


def test_put_retirement_limit_validation_error(client: httpx.Client) -> None:
    # year out of allowed range triggers validator
    payload = {
        "year": 1999,
        "account_wrapper": "IKE",
        "owner": PERSONA_MARCIN,
        "limit_amount": 1000.0,
    }
    response = client.put(
        f"/api/retirement/limits/1999/IKE/{PERSONA_MARCIN}",
        json=payload,
    )
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_post_ppk_contributions_validation_error(client: httpx.Client) -> None:
    # Invalid month → 422
    payload = {"owner": PERSONA_MARCIN, "month": 13, "year": 2025}
    response = client.post("/api/retirement/ppk-contributions/generate", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()
