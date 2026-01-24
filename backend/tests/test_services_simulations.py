from datetime import date

from sqlalchemy.orm import Session

from app.models.app_config import AppConfig
from app.models.retirement_limit import RetirementLimit
from app.schemas.simulations import PPKSimulationConfig, SimulationInputs
from app.services.simulations import (
    fetch_current_balances,
    fetch_ppk_balances,
    get_age_from_config,
    get_limit_for_year,
    get_ppk_return_for_age,
    run_simulation,
    simulate_account,
    simulate_brokerage_account,
    simulate_ppk_account,
)
from tests.factories import create_test_account, create_test_snapshot, create_test_snapshot_value


def test_get_limit_for_year_from_db(test_db_session: Session):
    """Test fetching limit from database"""
    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = get_limit_for_year(test_db_session, 2026, "IKE", "Marcin")

    assert result == 28260.0


def test_get_limit_for_year_fallback(test_db_session: Session):
    """Test fallback to default when limit not in DB"""
    result = get_limit_for_year(test_db_session, 2026, "IKE", "Marcin")

    assert result == 28260


def test_get_limit_for_year_ikze(test_db_session: Session):
    """Test IKZE limit fallback"""
    result = get_limit_for_year(test_db_session, 2026, "IKZE", "Marcin")

    assert result == 11304


def test_fetch_current_balances_no_snapshot(test_db_session: Session):
    """Test fetching balances when no snapshot exists"""
    balances = fetch_current_balances(test_db_session)

    assert balances == {"ike_marcin": 0, "ike_ewa": 0, "ikze_marcin": 0, "ikze_ewa": 0}


def test_fetch_current_balances_with_snapshot(test_db_session: Session):
    """Test fetching balances from latest snapshot"""
    # Create accounts
    ike_marcin = create_test_account(
        test_db_session,
        name="IKE Marcin",
        category="retirement",
        owner="Marcin",
        account_wrapper="IKE",
        purpose="retirement",
    )
    ikze_ewa = create_test_account(
        test_db_session,
        name="IKZE Ewa",
        category="retirement",
        owner="Ewa",
        account_wrapper="IKZE",
        purpose="retirement",
    )

    # Create snapshot
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2026, 1, 31), notes="Test")

    # Create snapshot values
    create_test_snapshot_value(test_db_session, snapshot.id, ike_marcin.id, value=50000.0)
    create_test_snapshot_value(test_db_session, snapshot.id, ikze_ewa.id, value=30000.0)

    balances = fetch_current_balances(test_db_session)

    assert balances["ike_marcin"] == 50000.0
    assert balances["ikze_ewa"] == 30000.0
    assert balances["ike_ewa"] == 0
    assert balances["ikze_marcin"] == 0


def test_get_age_from_config(test_db_session: Session):
    """Test age calculation from config"""
    from decimal import Decimal

    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )
    test_db_session.add(config)
    test_db_session.commit()

    age = get_age_from_config(test_db_session)

    assert age >= 35  # Born 1990, current year 2026+


def test_simulate_account_auto_fill(test_db_session: Session):
    """Test simulation with auto-fill limit"""
    # Add limit
    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = simulate_account(
        account_wrapper="IKE",
        owner="Marcin",
        starting_balance=10000.0,
        auto_fill_limit=True,
        monthly_contribution=0,
        years_to_retirement=5,
        current_age=30,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        tax_rate=0,
        db=test_db_session,
    )

    assert result.account_name == "IKE (Marcin)"
    assert result.starting_balance == 10000.0
    assert len(result.yearly_projections) == 5
    assert result.final_balance > 10000.0
    # First year contribution should be at grown limit (2026 base * 1.05)
    assert result.yearly_projections[0].annual_contribution == 29673.0  # 28260 * 1.05
    # Limit should grow each year
    assert result.yearly_projections[1].annual_limit > result.yearly_projections[0].annual_limit


def test_simulate_account_fixed_monthly(test_db_session: Session):
    """Test simulation with fixed monthly contributions"""
    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = simulate_account(
        account_wrapper="IKE",
        owner="Marcin",
        starting_balance=0,
        auto_fill_limit=False,
        monthly_contribution=1000.0,
        years_to_retirement=3,
        current_age=30,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        tax_rate=0,
        db=test_db_session,
    )

    # Annual contribution should be 12 * 1000 = 12000
    assert result.yearly_projections[0].annual_contribution == 12000.0
    assert result.total_contributions == 12000.0 * 3


def test_simulate_account_limit_capping(test_db_session: Session):
    """Test that contributions are capped at limit"""
    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = simulate_account(
        account_wrapper="IKE",
        owner="Marcin",
        starting_balance=0,
        auto_fill_limit=False,
        monthly_contribution=5000.0,  # 60000/year exceeds limit
        years_to_retirement=2,
        current_age=30,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        tax_rate=0,
        db=test_db_session,
    )

    # Should be capped at grown limit (28260 * 1.05 = 29673)
    assert result.yearly_projections[0].annual_contribution == 29673.0


def test_simulate_account_ikze_tax_savings(test_db_session: Session):
    """Test IKZE tax savings calculation"""
    limit = RetirementLimit(year=2026, account_wrapper="IKZE", owner="Marcin", limit_amount=11304.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = simulate_account(
        account_wrapper="IKZE",
        owner="Marcin",
        starting_balance=0,
        auto_fill_limit=True,
        monthly_contribution=0,
        years_to_retirement=2,
        current_age=30,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        tax_rate=17.0,
        db=test_db_session,
    )

    # Tax savings should be 17% of contribution (grown limit: 11304 * 1.05 = 11869.2)
    expected_tax_savings = 11304.0 * 1.05 * 0.17
    assert result.yearly_projections[0].tax_savings == expected_tax_savings
    assert result.total_tax_savings > 0


def test_simulate_account_compound_interest(test_db_session: Session):
    """Test compound interest calculation"""
    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    result = simulate_account(
        account_wrapper="IKE",
        owner="Marcin",
        starting_balance=10000.0,
        auto_fill_limit=False,
        monthly_contribution=0,  # No contributions to test pure compound interest
        years_to_retirement=2,
        current_age=30,
        annual_return_rate=10.0,
        limit_growth_rate=5.0,
        tax_rate=0,
        db=test_db_session,
    )

    # After year 1: 10000 * 1.10 = 11000
    assert abs(result.yearly_projections[0].balance_end_of_year - 11000.0) < 0.01
    # After year 2: 11000 * 1.10 = 12100
    assert abs(result.yearly_projections[1].balance_end_of_year - 12100.0) < 0.01


def test_run_simulation_all_accounts(test_db_session: Session):
    """Test full simulation with all accounts"""
    from decimal import Decimal

    # Setup
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )
    test_db_session.add(config)

    for wrapper in ["IKE", "IKZE"]:
        for owner in ["Marcin", "Ewa"]:
            limit_amount = 28260.0 if wrapper == "IKE" else 11304.0
            limit = RetirementLimit(
                year=2026,
                account_wrapper=wrapper,
                owner=owner,
                limit_amount=limit_amount,
            )
            test_db_session.add(limit)

    test_db_session.commit()

    inputs = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=True,
        simulate_ikze_marcin=True,
        simulate_ikze_ewa=True,
        ike_marcin_balance=10000.0,
        ike_ewa_balance=5000.0,
        ikze_marcin_balance=3000.0,
        ikze_ewa_balance=2000.0,
        ike_marcin_auto_fill=True,
        ike_ewa_auto_fill=True,
        ikze_marcin_auto_fill=True,
        ikze_ewa_auto_fill=True,
        marcin_tax_rate=17.0,
        ewa_tax_rate=17.0,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
    )

    result = run_simulation(test_db_session, inputs)

    assert len(result.simulations) == 4
    assert result.summary.years_until_retirement == 5
    assert result.summary.total_final_balance > 0
    assert result.summary.total_contributions > 0
    assert result.summary.total_returns > 0
    assert result.summary.total_tax_savings > 0  # IKZE accounts
    assert result.summary.estimated_monthly_income > 0
    assert result.summary.estimated_monthly_income_today > 0
    # Inflation-adjusted should be less than nominal
    assert result.summary.estimated_monthly_income_today < result.summary.estimated_monthly_income


def test_run_simulation_selective_accounts(test_db_session: Session):
    """Test simulation with only some accounts selected"""
    from decimal import Decimal

    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    inputs = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=False,
        simulate_ikze_marcin=False,
        simulate_ikze_ewa=False,
        ike_marcin_balance=10000.0,
        ike_marcin_auto_fill=True,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
    )

    result = run_simulation(test_db_session, inputs)

    assert len(result.simulations) == 1
    assert result.simulations[0].account_name == "IKE (Marcin)"
    assert result.summary.total_tax_savings == 0  # No IKZE accounts


def test_get_ppk_return_for_age():
    """Test age-based return rate for PPK"""
    assert get_ppk_return_for_age(25) == 7.0  # Aggressive
    assert get_ppk_return_for_age(39) == 7.0
    assert get_ppk_return_for_age(40) == 6.0  # Moderate
    assert get_ppk_return_for_age(49) == 6.0
    assert get_ppk_return_for_age(50) == 5.0  # Conservative
    assert get_ppk_return_for_age(59) == 5.0
    assert get_ppk_return_for_age(60) == 4.0  # Defensive
    assert get_ppk_return_for_age(70) == 4.0


def test_simulate_ppk_account_basic():
    """Test basic PPK simulation with subsidies"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=10000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=False,
        include_welcome_bonus=True,
        include_annual_subsidy=True,
    )

    result = simulate_ppk_account(config, current_age=30, retirement_age=35, salary_growth=3.0)

    assert result.account_name == "PPK (Marcin)"
    assert result.starting_balance == 0
    assert len(result.yearly_projections) == 5
    assert result.final_balance > 0
    assert result.total_contributions > 0
    assert result.total_subsidies > 0
    # Welcome bonus (250) + annual subsidies (240 * 5 years if eligible)
    assert result.total_subsidies >= 250


def test_simulate_ppk_account_welcome_bonus_timing():
    """Test welcome bonus added after 3 months"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=5000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=True,
        include_welcome_bonus=True,
        include_annual_subsidy=False,
    )

    result = simulate_ppk_account(config, current_age=30, retirement_age=31, salary_growth=0)

    # Welcome bonus should be added in year 0 (first year, after 3 months)
    assert result.total_subsidies == 250
    assert result.yearly_projections[0].government_subsidies == 250


def test_simulate_ppk_account_annual_subsidy_threshold():
    """Test annual subsidy eligibility based on threshold"""
    # Below threshold - should get annual subsidy
    config_below = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=5000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=True,
        include_welcome_bonus=False,
        include_annual_subsidy=True,
    )

    result_below = simulate_ppk_account(
        config_below, current_age=30, retirement_age=33, salary_growth=0
    )

    # Annual contribution: 5000 * (2 + 1.5) / 100 * 12 = 2100 PLN > 1009.26
    # Should get 240 PLN annual subsidy per year
    assert result_below.total_subsidies == 240 * 3  # 3 years

    # Above threshold - should NOT get annual subsidy
    config_above = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=5000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=False,
        include_welcome_bonus=False,
        include_annual_subsidy=True,
    )

    result_above = simulate_ppk_account(
        config_above, current_age=30, retirement_age=33, salary_growth=0
    )

    assert result_above.total_subsidies == 0


def test_simulate_ppk_account_low_salary_no_annual_subsidy():
    """Test annual subsidy withheld when contributions below 1009.26 PLN threshold"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=2000,  # Low salary
        employee_rate=0.5,  # Minimum rate
        employer_rate=1.5,  # Minimum rate
        salary_below_threshold=True,  # Eligible for subsidy
        include_welcome_bonus=True,
        include_annual_subsidy=True,
    )

    result = simulate_ppk_account(config, current_age=30, retirement_age=33, salary_growth=0)

    # Annual contribution: 2000 * (0.5 + 1.5) / 100 * 12 = 480 PLN < 1009.26
    # Should get welcome bonus (250) but NO annual subsidies
    assert result.total_subsidies == 250  # Only welcome bonus


def test_simulate_ppk_account_salary_growth():
    """Test salary growth over years"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=0,
        monthly_gross_salary=10000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=False,
        include_welcome_bonus=False,
        include_annual_subsidy=False,
    )

    result = simulate_ppk_account(config, current_age=30, retirement_age=33, salary_growth=5.0)

    # Check contributions increase with salary growth
    year0_contrib = result.yearly_projections[0].annual_contribution
    year1_contrib = result.yearly_projections[1].annual_contribution
    year2_contrib = result.yearly_projections[2].annual_contribution

    # Each year should have ~5% higher contributions
    assert year1_contrib > year0_contrib
    assert year2_contrib > year1_contrib
    # Year 1 should be approximately year0 * 1.05
    assert abs(year1_contrib / year0_contrib - 1.05) < 0.01


def test_simulate_ppk_account_age_based_returns():
    """Test that return rates change based on age"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=10000,
        monthly_gross_salary=10000,
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=False,
        include_welcome_bonus=False,
        include_annual_subsidy=False,
    )

    # Simulate from age 38 to 42 (crossing 40 boundary)
    result = simulate_ppk_account(config, current_age=38, retirement_age=42, salary_growth=0)

    # Age 38: 7% return
    assert result.yearly_projections[0].return_rate == 7.0
    # Age 39: 7% return
    assert result.yearly_projections[1].return_rate == 7.0
    # Age 40: 6% return (lifecycle rebalancing)
    assert result.yearly_projections[2].return_rate == 6.0
    # Age 41: 6% return
    assert result.yearly_projections[3].return_rate == 6.0


def test_simulate_ppk_account_monthly_compounding():
    """Test monthly compounding of returns"""
    config = PPKSimulationConfig(
        owner="Marcin",
        enabled=True,
        starting_balance=10000,
        monthly_gross_salary=0,  # No contributions to isolate compounding
        employee_rate=2.0,
        employer_rate=1.5,
        salary_below_threshold=False,
        include_welcome_bonus=False,
        include_annual_subsidy=False,
    )

    result = simulate_ppk_account(config, current_age=30, retirement_age=31, salary_growth=0)

    # 7% annual - 0.6% fees = 6.4% net return
    # Monthly: 6.4% / 12 = 0.533% per month
    # Expected: 10000 * (1.00533)^12 ≈ 10658
    expected_balance = 10000 * ((1 + 6.4 / 12 / 100) ** 12)
    assert abs(result.final_balance - expected_balance) < 10  # Allow small rounding difference


def test_fetch_ppk_balances(test_db_session: Session):
    """Test fetching PPK balances from snapshot"""
    # Create PPK accounts
    ppk_marcin = create_test_account(
        test_db_session,
        name="PPK Marcin",
        category="retirement",
        owner="Marcin",
        account_wrapper="PPK",
        purpose="retirement",
    )
    ppk_ewa = create_test_account(
        test_db_session,
        name="PPK Ewa",
        category="retirement",
        owner="Ewa",
        account_wrapper="PPK",
        purpose="retirement",
    )

    # Create snapshot
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2026, 1, 31), notes="Test")

    # Create snapshot values
    create_test_snapshot_value(test_db_session, snapshot.id, ppk_marcin.id, value=15000.0)
    create_test_snapshot_value(test_db_session, snapshot.id, ppk_ewa.id, value=12000.0)

    balances = fetch_ppk_balances(test_db_session)

    assert balances["marcin"] == 15000.0
    assert balances["ewa"] == 12000.0


def test_run_simulation_with_ppk(test_db_session: Session):
    """Test full simulation including PPK accounts"""
    from decimal import Decimal

    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
        ppk_employee_rate_marcin=Decimal("2.0"),
        ppk_employer_rate_marcin=Decimal("1.5"),
        ppk_employee_rate_ewa=Decimal("2.0"),
        ppk_employer_rate_ewa=Decimal("1.5"),
    )
    test_db_session.add(config)

    for wrapper in ["IKE", "IKZE"]:
        for owner in ["Marcin", "Ewa"]:
            limit_amount = 28260.0 if wrapper == "IKE" else 11304.0
            limit = RetirementLimit(
                year=2026, account_wrapper=wrapper, owner=owner, limit_amount=limit_amount
            )
            test_db_session.add(limit)

    test_db_session.commit()

    inputs = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=False,
        simulate_ikze_marcin=False,
        simulate_ikze_ewa=False,
        ike_marcin_balance=10000.0,
        ike_marcin_auto_fill=True,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        expected_salary_growth=3.0,
        ppk_marcin=PPKSimulationConfig(
            owner="Marcin",
            enabled=True,
            starting_balance=5000,
            monthly_gross_salary=10000,
            employee_rate=2.0,
            employer_rate=1.5,
            salary_below_threshold=False,
            include_welcome_bonus=True,
            include_annual_subsidy=True,
        ),
        ppk_ewa=PPKSimulationConfig(
            owner="Ewa",
            enabled=True,
            starting_balance=3000,
            monthly_gross_salary=8000,
            employee_rate=2.0,
            employer_rate=1.5,
            salary_below_threshold=False,
            include_welcome_bonus=True,
            include_annual_subsidy=True,
        ),
    )

    result = run_simulation(test_db_session, inputs)

    # Should have IKE Marcin + 2 PPK accounts
    assert len(result.simulations) == 3
    ppk_sims = [s for s in result.simulations if s.account_name.startswith("PPK")]
    assert len(ppk_sims) == 2
    assert result.summary.total_subsidies > 0  # Should have PPK subsidies
    assert result.summary.total_final_balance > 0
    assert result.summary.estimated_monthly_income > 0


def test_simulate_brokerage_account_basic():
    """Test basic brokerage simulation with capital gains tax"""
    result = simulate_brokerage_account(
        account_name="Rachunek maklerski (Marcin)",
        starting_balance=10000.0,
        monthly_contribution=1000.0,
        current_age=30,
        retirement_age=35,
        annual_return_rate=7.0,
        capital_gains_tax_rate=19.0,
    )

    assert result.account_name == "Rachunek maklerski (Marcin)"
    assert result.starting_balance == 10000.0
    assert len(result.yearly_projections) == 5
    assert result.final_balance > 10000.0
    assert result.total_contributions == 1000.0 * 12 * 5
    assert result.total_returns > 0
    assert result.total_tax_savings == 0  # No tax savings for brokerage


def test_simulate_brokerage_account_tax_calculation():
    """Test capital gains tax is correctly applied"""
    result = simulate_brokerage_account(
        account_name="Rachunek maklerski (Marcin)",
        starting_balance=10000.0,
        monthly_contribution=0,
        current_age=30,
        retirement_age=31,
        annual_return_rate=10.0,
        capital_gains_tax_rate=19.0,
    )

    # Year 1: 10000 + (10000 * 12 * 0) = 10000
    # Gross returns: 10000 * 0.10 = 1000
    # Tax: 1000 * 0.19 = 190
    # Net returns: 1000 * 0.81 = 810
    # Final balance: 10000 + 810 = 10810
    expected_balance = 10000 + (10000 * 0.10 * 0.81)
    assert abs(result.final_balance - expected_balance) < 0.01
    assert abs(result.total_returns - 810.0) < 0.01


def test_simulate_brokerage_account_zero_values():
    """Test brokerage simulation with zero balance and contributions"""
    result = simulate_brokerage_account(
        account_name="Rachunek maklerski (Ewa)",
        starting_balance=0,
        monthly_contribution=0,
        current_age=30,
        retirement_age=35,
        annual_return_rate=7.0,
    )

    assert result.final_balance == 0
    assert result.total_contributions == 0
    assert result.total_returns == 0


def test_simulate_brokerage_account_no_limits():
    """Test brokerage has no contribution limits"""
    result = simulate_brokerage_account(
        account_name="Rachunek maklerski (Marcin)",
        starting_balance=0,
        monthly_contribution=10000.0,  # Large contribution
        current_age=30,
        retirement_age=32,
        annual_return_rate=7.0,
    )

    # Should accept full contribution (10000 * 12 = 120000 per year)
    assert result.yearly_projections[0].annual_contribution == 120000.0
    assert result.yearly_projections[0].annual_limit == 0
    assert result.yearly_projections[0].limit_utilized_pct == 0


def test_run_simulation_with_brokerage(test_db_session: Session):
    """Test full simulation including brokerage accounts"""
    from decimal import Decimal

    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    inputs = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=False,
        simulate_ikze_marcin=False,
        simulate_ikze_ewa=False,
        ike_marcin_balance=10000.0,
        ike_marcin_auto_fill=True,
        simulate_brokerage_marcin=True,
        simulate_brokerage_ewa=False,
        brokerage_marcin_balance=5000.0,
        brokerage_marcin_monthly=1000.0,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        inflation_rate=3.0,
    )

    result = run_simulation(test_db_session, inputs)

    # Should have IKE Marcin + Brokerage Marcin
    assert len(result.simulations) == 2
    brokerage_sims = [s for s in result.simulations if "maklerski" in s.account_name]
    assert len(brokerage_sims) == 1
    assert brokerage_sims[0].account_name == "Rachunek maklerski (Marcin)"
    assert result.summary.total_final_balance > 0
    assert result.summary.estimated_monthly_income > 0


def test_run_simulation_custom_inflation_rate(test_db_session: Session):
    """Test inflation rate parameter affects monthly income calculation"""
    from decimal import Decimal

    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    # Run with 3% inflation
    inputs_3pct = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=False,
        simulate_ikze_marcin=False,
        simulate_ikze_ewa=False,
        ike_marcin_balance=10000.0,
        ike_marcin_auto_fill=True,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        inflation_rate=3.0,
    )
    result_3pct = run_simulation(test_db_session, inputs_3pct)

    # Run with 5% inflation
    inputs_5pct = SimulationInputs(
        current_age=30,
        retirement_age=35,
        simulate_ike_marcin=True,
        simulate_ike_ewa=False,
        simulate_ikze_marcin=False,
        simulate_ikze_ewa=False,
        ike_marcin_balance=10000.0,
        ike_marcin_auto_fill=True,
        annual_return_rate=7.0,
        limit_growth_rate=5.0,
        inflation_rate=5.0,
    )
    result_5pct = run_simulation(test_db_session, inputs_5pct)

    # Higher inflation = lower purchasing power today
    assert (
        result_5pct.summary.estimated_monthly_income_today
        < result_3pct.summary.estimated_monthly_income_today
    )
    # Nominal income should be same
    assert (
        result_5pct.summary.estimated_monthly_income
        == result_3pct.summary.estimated_monthly_income
    )
