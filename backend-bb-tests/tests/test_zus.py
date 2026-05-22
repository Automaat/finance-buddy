"""Black-box tests for /api/zus — pension projection + prefill."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import PERSONA_MARCIN


@pytest.mark.golden
def test_get_zus_prefill_matches_golden(
    client: httpx.Client, update_golden: bool, owner_ids: dict[str, int]
) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/zus/prefill", params={"owner_user_id": owner_ids[PERSONA_MARCIN]})
    assert response.status_code == 200, response.text
    assert_matches_golden("zus_prefill_marcin", response.json(), update=update_golden)


def test_get_zus_prefill_shape(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    response = client.get("/api/zus/prefill", params={"owner_user_id": owner_ids[PERSONA_MARCIN]})
    assert response.status_code == 200, response.text
    body = response.json()
    required = {
        "birth_date",
        "retirement_age",
        "gender",
        "current_gross_monthly_salary",
        "owner_user_id",
        "salary_history",
    }
    assert required.issubset(body.keys())


@pytest.mark.golden
def test_post_zus_calculate_matches_golden(
    client: httpx.Client, update_golden: bool, owner_ids: dict[str, int]
) -> None:
    from _golden import assert_matches_golden

    # Deterministic input — no DB reads in calculate, pure computation.
    payload = {
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "birth_date": "1990-06-15",
        "gender": "M",
        "retirement_age": 65,
        "current_gross_monthly_salary": 18000.0,
        "salary_growth_rate": 3.0,
        "inflation_rate": 3.0,
        "valorization_rate_konto": 5.0,
        "valorization_rate_subkonto": 4.0,
        "has_ofe": False,
        "kapital_poczatkowy": 0.0,
        "work_start_year": 2014,
        "salary_history": [],
    }
    response = client.post("/api/zus/calculate", json=payload)
    assert response.status_code == 200, response.text
    assert_matches_golden("zus_calculate_marcin", response.json(), update=update_golden)


def test_post_zus_calculate_happy_path(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    payload = {
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "birth_date": "1990-06-15",
        "gender": "M",
        "retirement_age": 65,
        "current_gross_monthly_salary": 18000.0,
        "work_start_year": 2014,
    }
    response = client.post("/api/zus/calculate", json=payload)
    assert response.status_code == 200, response.text
    body = response.json()
    required = {
        "inputs",
        "yearly_projections",
        "life_expectancy_months",
        "konto_at_retirement",
        "subkonto_at_retirement",
        "monthly_pension_gross",
        "monthly_pension_net",
        "replacement_rate",
        "sensitivity",
    }
    assert required.issubset(body.keys())


def test_post_zus_calculate_validation_error(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    # Invalid gender → validator raises
    payload = {
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "birth_date": "1990-06-15",
        "gender": "X",
        "retirement_age": 65,
        "current_gross_monthly_salary": 18000.0,
        "work_start_year": 2014,
    }
    response = client.post("/api/zus/calculate", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()
