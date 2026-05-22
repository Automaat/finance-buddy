"""Black-box tests for /api/simulations — pure compute, ideal for golden."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import PERSONA_MARCIN


@pytest.mark.golden
def test_get_simulations_prefill_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/simulations/prefill")
    assert response.status_code == 200, response.text
    assert_matches_golden("simulations_prefill", response.json(), update=update_golden)


def test_get_simulations_prefill_shape(client: httpx.Client) -> None:
    response = client.get("/api/simulations/prefill")
    assert response.status_code == 200, response.text
    body = response.json()
    required = {
        "current_age",
        "retirement_age",
        "balances",
        "ppk_rates",
        "monthly_salaries",
        "ppk_balances",
    }
    assert required.issubset(body.keys())


@pytest.mark.golden
def test_post_mortgage_vs_invest_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    payload = {
        "remaining_principal": 280000.0,
        "annual_interest_rate": 7.25,
        "remaining_months": 300,
        "total_monthly_budget": 4000.0,
        "expected_annual_return": 7.0,
        "inflation_rate": 3.0,
        "enable_variable_rate": False,
    }
    response = client.post("/api/simulations/mortgage-vs-invest", json=payload)
    assert response.status_code == 200, response.text
    assert_matches_golden("simulations_mortgage_vs_invest", response.json(), update=update_golden)


def test_post_mortgage_vs_invest_happy_path(client: httpx.Client) -> None:
    payload = {
        "remaining_principal": 280000.0,
        "annual_interest_rate": 7.25,
        "remaining_months": 300,
        "total_monthly_budget": 4000.0,
        "expected_annual_return": 7.0,
    }
    response = client.post("/api/simulations/mortgage-vs-invest", json=payload)
    assert response.status_code == 200, response.text
    body = response.json()
    required = {"inputs", "yearly_projections", "summary"}
    assert required.issubset(body.keys())
    assert {"regular_monthly_payment", "winning_strategy", "net_advantage"}.issubset(
        body["summary"].keys()
    )


def test_post_mortgage_vs_invest_validation_error(client: httpx.Client) -> None:
    # Negative principal → validator rejects
    payload = {
        "remaining_principal": -100.0,
        "annual_interest_rate": 7.0,
        "remaining_months": 300,
        "total_monthly_budget": 4000.0,
        "expected_annual_return": 7.0,
    }
    response = client.post("/api/simulations/mortgage-vs-invest", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


@pytest.mark.golden
def test_post_simulate_retirement_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    payload = {
        "current_age": 35,
        "retirement_age": 65,
        "ike_ikze_accounts": [
            {
                "enabled": True,
                "wrapper": "IKE",
                "owner": PERSONA_MARCIN,
                "balance": 46100.0,
                "auto_fill_limit": False,
                "monthly_contribution": 1500.0,
                "tax_rate": 0.0,
            }
        ],
        "ppk_accounts": [],
        "brokerage_accounts": [],
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
        "expected_salary_growth": 3.0,
        "inflation_rate": 3.0,
    }
    response = client.post("/api/simulations/retirement", json=payload)
    assert response.status_code == 200, response.text
    # 30-year compound projection leaves intermediate fields (annual_limit,
    # limit_utilized_pct, balances) unrounded. Python's float ** int and Go's
    # math.Pow diverge in the last ULP; compounded over 30 years that's a
    # ~1e-6 PLN gap — far below the 0.01 PLN tolerance this endpoint targets.
    # Round to 2 dp (== the documented tolerance) so the golden is
    # backend-agnostic. (_golden.py sanctions per-test normalization.)
    assert_matches_golden(
        "simulations_retirement", _round_floats(response.json()), update=update_golden
    )


def _round_floats(value: object) -> object:
    """Recursively round every float to 2 dp, leaving ints + bools intact —
    erases backend float-precision noise within the 0.01 PLN tolerance."""
    if isinstance(value, bool):
        return value
    if isinstance(value, float):
        return round(value, 2)
    if isinstance(value, int):
        return value
    if isinstance(value, dict):
        return {k: _round_floats(v) for k, v in value.items()}
    if isinstance(value, list):
        return [_round_floats(v) for v in value]
    return value


def test_post_simulate_retirement_happy_path(client: httpx.Client) -> None:
    payload = {
        "current_age": 35,
        "retirement_age": 65,
        "ike_ikze_accounts": [
            {
                "enabled": True,
                "wrapper": "IKE",
                "owner": PERSONA_MARCIN,
                "balance": 46100.0,
                "monthly_contribution": 1500.0,
            }
        ],
        "ppk_accounts": [],
        "brokerage_accounts": [],
    }
    response = client.post("/api/simulations/retirement", json=payload)
    assert response.status_code == 200, response.text
    body = response.json()
    required = {"inputs", "simulations", "summary"}
    assert required.issubset(body.keys())
    assert {
        "total_final_balance",
        "total_contributions",
        "estimated_monthly_income",
        "years_until_retirement",
    }.issubset(body["summary"].keys())


def test_post_simulate_retirement_validation_error(client: httpx.Client) -> None:
    # retirement_age <= current_age → model_validator rejects
    payload = {
        "current_age": 65,
        "retirement_age": 60,
        "ike_ikze_accounts": [],
        "ppk_accounts": [],
        "brokerage_accounts": [],
    }
    response = client.post("/api/simulations/retirement", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()
