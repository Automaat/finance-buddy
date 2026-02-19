import pytest
from pydantic import ValidationError

from app.schemas.simulations import MortgageVsInvestInputs
from app.services.simulations import simulate_mortgage_vs_invest


def _base_inputs(**overrides) -> MortgageVsInvestInputs:
    defaults = {
        "remaining_principal": 300_000,
        "annual_interest_rate": 6.5,
        "remaining_months": 240,
        "total_monthly_budget": 3_300,  # ~2237 regular payment + ~1000 extra
        "expected_annual_return": 8.0,
    }
    defaults.update(overrides)
    return MortgageVsInvestInputs(**defaults)


def test_regular_payment_formula():
    """M = P * [r(1+r)^n] / [(1+r)^n - 1]"""
    r = 6.5 / 100 / 12
    n = 240
    p = 300_000
    expected = p * (r * (1 + r) ** n) / ((1 + r) ** n - 1)
    # budget exactly equals regular payment: no extra
    inputs = _base_inputs(total_monthly_budget=expected)
    result = simulate_mortgage_vs_invest(inputs)

    assert abs(result.summary.regular_monthly_payment - expected) < 0.01


def test_scenario_a_pays_off_early_with_large_extra():
    """Large extra amount should pay off mortgage before term end."""
    inputs = _base_inputs(
        remaining_principal=100_000,
        annual_interest_rate=5.0,
        remaining_months=120,
        total_monthly_budget=6_100,  # ~1061 regular + 5000 extra
        expected_annual_return=4.0,
    )
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.months_saved > 0
    # Scenario A should accumulate less interest than B
    assert result.summary.total_interest_a < result.summary.total_interest_b


def test_high_return_invest_wins():
    """With high investment return (9%) net-of-Belka (7.29%) beats overpaying (6.5% mortgage)."""
    inputs = _base_inputs(expected_annual_return=9.0)
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
    r = 6.5 / 100 / 12
    n = 240
    p = 300_000
    regular = p * (r * (1 + r) ** n) / ((1 + r) ** n - 1)
    inputs = _base_inputs(total_monthly_budget=regular)  # no extra
    result = simulate_mortgage_vs_invest(inputs)

    assert result.summary.total_interest_a == result.summary.total_interest_b
    assert result.summary.months_saved == 0
    assert result.summary.final_investment_portfolio == 0.0
    assert result.summary.interest_saved == 0.0


def test_yearly_projections_count():
    """Yearly projections should have one row per year."""
    # 300k/6.5%/120m regular payment ~3406, use 4000 budget
    inputs = _base_inputs(remaining_months=120, total_monthly_budget=4_000)
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
            total_monthly_budget=2000,
            expected_annual_return=7.0,
        )


def test_invalid_interest_rate():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=0,
            remaining_months=120,
            total_monthly_budget=2000,
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
            total_monthly_budget=2000,
            expected_annual_return=-1.0,
        )


def test_invalid_remaining_months():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=6.5,
            remaining_months=0,
            total_monthly_budget=2000,
            expected_annual_return=7.0,
        )


def test_invalid_budget():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=6.5,
            remaining_months=120,
            total_monthly_budget=0,
            expected_annual_return=7.0,
        )


def test_invalid_inflation_rate():
    with pytest.raises(ValidationError):
        MortgageVsInvestInputs(
            remaining_principal=100_000,
            annual_interest_rate=6.5,
            remaining_months=120,
            total_monthly_budget=2000,
            expected_annual_return=7.0,
            inflation_rate=25.0,
        )


def test_budget_below_regular_payment_raises():
    """Budget less than regular payment raises HTTPException."""
    from fastapi import HTTPException

    r = 6.5 / 100 / 12
    n = 120
    p = 100_000
    regular = p * (r * (1 + r) ** n) / ((1 + r) ** n - 1)
    inputs = _base_inputs(
        remaining_principal=p,
        annual_interest_rate=6.5,
        remaining_months=n,
        total_monthly_budget=regular - 1,
    )
    with pytest.raises(HTTPException) as exc_info:
        simulate_mortgage_vs_invest(inputs)
    assert exc_info.value.status_code == 400


def test_variable_rate_enabled():
    """Simulation with variable rate should complete and return projections."""
    inputs = _base_inputs(remaining_months=120, total_monthly_budget=4_000)
    inputs = MortgageVsInvestInputs(**{**inputs.model_dump(), "enable_variable_rate": True})
    result = simulate_mortgage_vs_invest(inputs)
    assert len(result.yearly_projections) == 10
    # Variable rate changes from fixed rate
    rates = [r.annual_rate for r in result.yearly_projections]
    assert any(r != rates[0] for r in rates)


def test_scenario_b_pays_off_early_invest_continues():
    """With large budget scenario B mortgage pays off and extra goes to portfolio."""
    inputs = _base_inputs(
        remaining_principal=50_000,
        annual_interest_rate=5.0,
        remaining_months=60,
        total_monthly_budget=10_000,
        expected_annual_return=4.0,
    )
    result = simulate_mortgage_vs_invest(inputs)
    # With 10k budget and 50k principal at 5%, mortgage pays off quickly
    # After payoff extra_b = total_monthly_budget goes to portfolio
    assert result.summary.final_investment_portfolio > 0
