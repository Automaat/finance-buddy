from datetime import date

from sqlalchemy.orm import Session

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.retirement_limit import RetirementLimit
from app.models.snapshot import Snapshot, SnapshotValue
from app.schemas.simulations import SimulationInputs
from app.services.simulations import (
    fetch_current_balances,
    get_age_from_config,
    get_limit_for_year,
    run_simulation,
    simulate_account,
)


def test_get_limit_for_year_from_db(test_db_session: Session):
    """Test fetching limit from database"""
    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
    )
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
    ike_marcin = Account(
        name="IKE Marcin",
        type="asset",
        category="retirement",
        owner="Marcin",
        currency="PLN",
        account_wrapper="IKE",
        purpose="retirement",
    )
    ikze_ewa = Account(
        name="IKZE Ewa",
        type="asset",
        category="retirement",
        owner="Ewa",
        currency="PLN",
        account_wrapper="IKZE",
        purpose="retirement",
    )
    test_db_session.add_all([ike_marcin, ikze_ewa])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2026, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create snapshot values
    value1 = SnapshotValue(
        snapshot_id=snapshot.id, account_id=ike_marcin.id, value=50000.0
    )
    value2 = SnapshotValue(
        snapshot_id=snapshot.id, account_id=ikze_ewa.id, value=30000.0
    )
    test_db_session.add_all([value1, value2])
    test_db_session.commit()

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
    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
    )
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
    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
    )
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
    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
    )
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
    limit = RetirementLimit(
        year=2026, account_wrapper="IKZE", owner="Marcin", limit_amount=11304.0
    )
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
    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
    )
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

    limit = RetirementLimit(
        year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0
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
    )

    result = run_simulation(test_db_session, inputs)

    assert len(result.simulations) == 1
    assert result.simulations[0].account_name == "IKE (Marcin)"
    assert result.summary.total_tax_savings == 0  # No IKZE accounts
