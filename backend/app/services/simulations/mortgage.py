import math

from fastapi import HTTPException

from app.schemas.simulations import (
    MortgageVsInvestInputs,
    MortgageVsInvestResponse,
    MortgageVsInvestSummary,
    MortgageVsInvestYearlyRow,
)


def simulate_mortgage_vs_invest(inputs: MortgageVsInvestInputs) -> MortgageVsInvestResponse:
    """Compare overpaying mortgage vs investing the extra amount month by month."""
    belka_rate = 0.19
    monthly_rate = inputs.annual_interest_rate / 100 / 12
    # Belka (19%) baked into monthly compounding: break-even at mortgage_rate / (1 - belka_rate)
    monthly_invest_rate = inputs.expected_annual_return * (1 - belka_rate) / 100 / 12
    n = inputs.remaining_months
    p = inputs.remaining_principal

    # Initial payment at starting rate (used for budget validation and summary)
    regular_payment = p * (monthly_rate * (1 + monthly_rate) ** n) / ((1 + monthly_rate) ** n - 1)

    if inputs.total_monthly_budget < regular_payment:
        raise HTTPException(
            status_code=400,
            detail=(
                f"Total monthly budget ({inputs.total_monthly_budget:.2f} PLN) is less than "
                f"the required mortgage payment ({regular_payment:.2f} PLN)"
            ),
        )

    # Scenario A: overpay mortgage, then invest freed cash after payoff
    balance_a = p
    cumulative_interest_a = 0.0
    investment_a = 0.0
    payoff_month_a = n  # default: pays off at term end

    # Scenario B: pay minimum each month, invest the rest
    balance_b = p
    investment_b = 0.0
    cumulative_interest_b = 0.0

    yearly_projections: list[MortgageVsInvestYearlyRow] = []
    current_annual_rate = inputs.annual_interest_rate  # updated each month if variable

    for month in range(1, n + 1):
        remaining_months = n - month + 1

        # Compute current mortgage rate
        if inputs.enable_variable_rate:
            # Pure cyclical model calibrated to Polish rate history (1% COVID low, 8% 2022 peak).
            # long_term_mean=4.5%, amplitude=3.5% → range 1%–8%, period=10yr.
            # Phase is derived from the starting rate so the trajectory always begins
            # going DOWN (reflecting current NBP rate-cutting cycle).
            long_term_mean = 4.5
            cycle_amplitude = 3.5
            cycle_period = 10.0
            sin_val = (inputs.annual_interest_rate - long_term_mean) / cycle_amplitude
            sin_val = max(-1.0, min(1.0, sin_val))
            # Second-quadrant phase → cosine negative → cycle decreasing at year 0
            phase = math.pi - math.asin(sin_val)
            year = (month - 1) / 12
            angle = 2 * math.pi * year / cycle_period + phase
            current_annual_rate = max(1.0, long_term_mean + cycle_amplitude * math.sin(angle))
        current_monthly_rate = current_annual_rate / 100 / 12

        # Scenario A: pay full budget towards mortgage, invest after payoff
        if balance_a > 0:
            interest_a = balance_a * current_monthly_rate
            cumulative_interest_a += interest_a
            amount_to_clear = balance_a + interest_a
            actual_payment_a = min(inputs.total_monthly_budget, amount_to_clear)
            surplus_a = inputs.total_monthly_budget - actual_payment_a
            balance_a = max(0.0, balance_a - (actual_payment_a - interest_a))
            if balance_a == 0 and payoff_month_a == n:
                payoff_month_a = month
            if surplus_a > 0:
                investment_a = (investment_a + surplus_a) * (1 + monthly_invest_rate)
        else:
            investment_a = (investment_a + inputs.total_monthly_budget) * (1 + monthly_invest_rate)

        # Scenario B: recalculate minimum payment at current rate, invest the rest
        interest_b = balance_b * current_monthly_rate
        cumulative_interest_b += interest_b
        min_payment_b = (
            balance_b
            * (current_monthly_rate * (1 + current_monthly_rate) ** remaining_months)
            / ((1 + current_monthly_rate) ** remaining_months - 1)
        )
        principal_payment_b = min_payment_b - interest_b
        balance_b = max(0.0, balance_b - principal_payment_b)
        extra_b = max(0.0, inputs.total_monthly_budget - min_payment_b)

        investment_b = (investment_b + extra_b) * (1 + monthly_invest_rate)

        # Collect yearly snapshot
        if month % 12 == 0:
            year = month // 12
            # Belka already baked into monthly_invest_rate — portfolio values are net
            after_tax_a = investment_a
            after_tax_b = investment_b
            # Inflation-adjusted (real) values in today's PLN
            inflation_factor = (1 + inputs.inflation_rate / 100) ** year
            real_a = after_tax_a / inflation_factor
            real_b = after_tax_b / inflation_factor
            # Inflation also erodes the real value of outstanding mortgage debt
            real_balance_a = balance_a / inflation_factor
            real_balance_b = balance_b / inflation_factor
            # Net worth comparison: real after-tax portfolio minus real mortgage balance
            net_advantage = (real_b - real_balance_b) - (real_a - real_balance_a)
            yearly_projections.append(
                MortgageVsInvestYearlyRow(
                    year=year,
                    annual_rate=round(current_annual_rate, 4),
                    scenario_a_mortgage_balance=round(balance_a, 2),
                    scenario_a_real_mortgage_balance=round(real_balance_a, 2),
                    scenario_a_cumulative_interest=round(cumulative_interest_a, 2),
                    scenario_a_investment_balance=round(investment_a, 2),
                    scenario_a_after_tax_portfolio=round(after_tax_a, 2),
                    scenario_a_real_portfolio=round(real_a, 2),
                    scenario_a_paid_off=balance_a == 0,
                    scenario_b_mortgage_balance=round(balance_b, 2),
                    scenario_b_real_mortgage_balance=round(real_balance_b, 2),
                    scenario_b_investment_balance=round(investment_b, 2),
                    scenario_b_after_tax_portfolio=round(after_tax_b, 2),
                    scenario_b_real_portfolio=round(real_b, 2),
                    scenario_b_cumulative_interest=round(cumulative_interest_b, 2),
                    net_advantage_invest=round(net_advantage, 2),
                )
            )

    interest_saved = cumulative_interest_b - cumulative_interest_a
    months_saved = n - payoff_month_a
    # Belka baked into monthly rate — no lump-sum tax at liquidation
    final_after_tax_a = investment_a
    final_after_tax_b = investment_b
    # Inflation-adjust final portfolios to today's PLN
    final_inflation_factor = (1 + inputs.inflation_rate / 100) ** (n / 12)
    final_real_a = final_after_tax_a / final_inflation_factor
    final_real_b = final_after_tax_b / final_inflation_factor

    net_advantage_final = final_real_b - final_real_a

    if net_advantage_final >= 0:
        winning_strategy = "inwestycja"
        net_advantage = net_advantage_final
    else:
        winning_strategy = "nadpłata"
        net_advantage = final_real_a - final_real_b

    summary = MortgageVsInvestSummary(
        regular_monthly_payment=round(regular_payment, 2),
        total_interest_a=round(cumulative_interest_a, 2),
        total_interest_b=round(cumulative_interest_b, 2),
        interest_saved=round(interest_saved, 2),
        final_investment_portfolio=round(investment_b, 2),
        belka_tax_a=0.0,
        belka_tax_b=0.0,
        final_portfolio_a_real=round(final_real_a, 2),
        final_portfolio_b_real=round(final_real_b, 2),
        months_saved=months_saved,
        winning_strategy=winning_strategy,
        net_advantage=round(net_advantage, 2),
        break_even_gross_return=round(inputs.annual_interest_rate / (1 - belka_rate), 4),
    )

    return MortgageVsInvestResponse(
        inputs=inputs,
        yearly_projections=yearly_projections,
        summary=summary,
    )
