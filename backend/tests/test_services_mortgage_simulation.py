import pytest
from pydantic import ValidationError

from app.schemas.simulations import MortgageVsInvestInputs
from app.services.simulations import simulate_mortgage_vs_invest


def _base_inputs(**overrides) -> MortgageVsInvestInputs:
    defaults = {
        "remaining_principal": 300_000,
        "annual_interest_rate": 6.5,
        "remaining_months": 240,
        "extra_monthly_amount": 1000,
        "expected_annual_return": 8.0,
    }
    defaults.update(overrides)
    return MortgageVsInvestInputs(**defaults)


def test_regular_payment_formula():
    """M = P * [r(1+r)^n] / [(1+r)^n - 1]"""
    inputs = _base_inputs(extra_monthly_amount=0)
    result = simulate_mortgage_vs_invest(inputs)

    r = 6.5 / 100 / 12
    n = 240
    p = 300_000
    expected = p * (r * (1 + r) ** n) / ((1 + r) ** n - 1)

    assert abs(result.summary.regular_monthly_payment - expected) < 0.01


def test_scenario_a_pays_off_early_with_large_extra():
    """Large extra amount should pay off mortgage before term end."""
    inputs = _base_inputs(
        remaining_principal=100_000,
        annual_interest_rate=5.0,
        remaining_months=120,
        extra_monthly_amount=5000,
        expected_annual_return=4.0,
    )
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.months_saved > 0
    # Scenario A should accumulate less interest than B
    assert result.summary.total_interest_a < result.summary.total_interest_b


def test_high_return_invest_wins():
    """With high investment return (8%) invest should beat overpaying (6.5% mortgage)."""
    inputs = _base_inputs(expected_annual_return=8.0)
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.winning_strategy == "inwestycja"
    assert result.summary.net_advantage > 0


def test_low_return_overpay_wins():
    """With low investment return (3%) overpaying should beat investing."""
    inputs = _base_inputs(expected_annual_return=3.0)
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.winning_strategy == "nadpłata"
    assert result.summary.net_advantage > 0


def test_zero_extra_no_difference():
    """With no extra amount both scenarios have same interest; no portfolio in B."""
    inputs = _base_inputs(extra_monthly_amount=0)
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.total_interest_a == result.summary.total_interest_b
    assert result.summary.months_saved == 0
    assert result.summary.final_investment_portfolio == 0.0
    assert result.summary.interest_saved == 0.0


def test_yearly_projections_count():
    """Yearly projections should have one row per year."""
    inputs = _base_inputs(remaining_months=120)
    result = simulate_mortgage_vs_invest(inputs)

    assert len(result.yearly_projections) == 10


def test_yearly_projections_balances_decrease():
    """Mortgage balance should decrease over time in both scenarios."""
    inputs = _base_inputs()
    result = simulate_mortgage_vs_invest(inputs)

    balances_a = [row.scenario_a_mortgage_balance for row in result.yearly_projections]
    balances_b = [row.scenario_b_mortgage_balance for row in result.yearly_projections]

    # Each year balance should be <= previous year
    for i in range(1, len(balances_a)):
        assert balances_a[i] <= balances_a[i - 1]
        assert balances_b[i] <= balances_b[i - 1]


def test_invalid_principal():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=-1000,
            annual_interest_rate=6.5,
            remaining_months=120,
            extra_monthly_amount=500,
            expected_annual_return=7.0,
        )


def test_invalid_interest_rate():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=0,
            remaining_months=120,
            extra_monthly_amount=500,
            expected_annual_return=7.0,
        )


def test_zero_expected_return_is_valid():
    """0% expected return is valid (money kept without growth)."""
    inputs = _base_inputs(expected_annual_return=0)
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.winning_strategy == "nadpłata"


def test_invalid_expected_return():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=6.5,
            remaining_months=120,
            extra_monthly_amount=500,
            expected_annual_return=-1.0,
        )
