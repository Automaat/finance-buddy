from datetime import date
from decimal import Decimal

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.salary_record import SalaryRecord
from app.models.snapshot import Snapshot, SnapshotValue
from app.services.dashboard import (
    _calculate_debt_to_income,
    _calculate_savings_rate,
    get_dashboard_data,
)
from tests.factories import (
    create_test_account,
    create_test_asset,
    create_test_snapshot,
    create_test_snapshot_value,
)


def test_empty_database(test_db_session):
    """Test dashboard with no data returns zeros and empty lists."""
    result = get_dashboard_data(test_db_session)

    assert result.current_net_worth == 0
    assert result.change_vs_last_month == 0
    assert result.total_assets == 0
    assert result.total_liabilities == 0
    assert result.net_worth_history == []
    assert result.allocation == []


def test_single_snapshot(test_db_session):
    """Test dashboard with single snapshot returns correct totals."""
    asset_account = create_test_account(
        test_db_session, name="Savings", category="banking", owner="John"
    )
    liability_account = create_test_account(
        test_db_session, name="Mortgage", account_type="liability", category="housing", owner="John"
    )

    snapshot = create_test_snapshot(
        test_db_session, snapshot_date=date(2024, 1, 1), notes="January"
    )

    create_test_snapshot_value(test_db_session, snapshot.id, asset_account.id, Decimal("10000.00"))
    create_test_snapshot_value(
        test_db_session, snapshot.id, liability_account.id, Decimal("5000.00")
    )

    result = get_dashboard_data(test_db_session)

    assert result.current_net_worth == 5000.0
    assert result.change_vs_last_month == 5000.0  # First month, vs 0
    assert result.total_assets == 10000.0
    assert result.total_liabilities == 5000.0
    assert len(result.net_worth_history) == 1
    assert result.net_worth_history[0].date == date(2024, 1, 1)
    assert result.net_worth_history[0].value == 5000.0
    assert len(result.allocation) == 1
    assert result.allocation[0].category == "banking"
    assert result.allocation[0].owner == "John"
    assert result.allocation[0].value == 10000.0


def test_multiple_snapshots(test_db_session):
    """Test dashboard with multiple snapshots shows net worth history."""
    account = create_test_account(test_db_session, name="Savings", category="banking", owner="John")

    snapshot1 = create_test_snapshot(
        test_db_session, snapshot_date=date(2024, 1, 1), notes="January"
    )
    snapshot2 = create_test_snapshot(
        test_db_session, snapshot_date=date(2024, 2, 1), notes="February"
    )
    snapshot3 = create_test_snapshot(test_db_session, snapshot_date=date(2024, 3, 1), notes="March")

    create_test_snapshot_value(test_db_session, snapshot1.id, account.id, Decimal("1000.00"))
    create_test_snapshot_value(test_db_session, snapshot2.id, account.id, Decimal("1500.00"))
    create_test_snapshot_value(test_db_session, snapshot3.id, account.id, Decimal("2000.00"))

    result = get_dashboard_data(test_db_session)

    assert result.current_net_worth == 2000.0
    assert result.change_vs_last_month == 500.0  # 2000 - 1500
    assert len(result.net_worth_history) == 3
    assert result.net_worth_history[0].value == 1000.0
    assert result.net_worth_history[1].value == 1500.0
    assert result.net_worth_history[2].value == 2000.0


def test_asset_liability_aggregation(test_db_session):
    """Test aggregation of assets and liabilities by type."""
    savings = create_test_account(test_db_session, name="Savings", category="banking", owner="John")
    stocks = create_test_account(
        test_db_session, name="Stocks", category="investments", owner="John"
    )
    credit_card = create_test_account(
        test_db_session,
        name="Credit Card",
        account_type="liability",
        category="other",
        owner="John",
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(test_db_session, snapshot.id, savings.id, Decimal("5000.00"))
    create_test_snapshot_value(test_db_session, snapshot.id, stocks.id, Decimal("3000.00"))
    create_test_snapshot_value(test_db_session, snapshot.id, credit_card.id, Decimal("1000.00"))

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 8000.0  # 5000 + 3000
    assert result.total_liabilities == 1000.0
    assert result.current_net_worth == 7000.0


def test_allocation_by_category_and_owner(test_db_session):
    """Test asset allocation aggregated by category and owner."""
    john_savings = create_test_account(
        test_db_session, name="John Savings", category="banking", owner="John"
    )
    jane_savings = create_test_account(
        test_db_session, name="Jane Savings", category="banking", owner="Jane"
    )
    john_stocks = create_test_account(
        test_db_session, name="John Stocks", category="investments", owner="John"
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(test_db_session, snapshot.id, john_savings.id, Decimal("2000.00"))
    create_test_snapshot_value(test_db_session, snapshot.id, jane_savings.id, Decimal("3000.00"))
    create_test_snapshot_value(test_db_session, snapshot.id, john_stocks.id, Decimal("5000.00"))

    result = get_dashboard_data(test_db_session)

    assert len(result.allocation) == 3
    # Find each allocation item
    john_banking = next(
        a for a in result.allocation if a.category == "banking" and a.owner == "John"
    )
    jane_banking = next(
        a for a in result.allocation if a.category == "banking" and a.owner == "Jane"
    )
    john_investments = next(
        a for a in result.allocation if a.category == "investments" and a.owner == "John"
    )

    assert john_banking.value == 2000.0
    assert jane_banking.value == 3000.0
    assert john_investments.value == 5000.0


def test_snapshots_exist_but_no_values(test_db_session):
    """Test edge case: snapshots exist but merge filters out all (no accounts/values)."""
    create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1), notes="Empty snapshot")

    result = get_dashboard_data(test_db_session)

    # Should handle gracefully with zeros
    assert result.current_net_worth == 0
    assert result.change_vs_last_month == 0
    assert result.total_assets == 0
    assert result.total_liabilities == 0
    assert result.net_worth_history == []
    assert result.allocation == []


def test_inactive_accounts_excluded(test_db_session):
    """Test that inactive accounts are excluded from dashboard."""
    active_account = create_test_account(
        test_db_session, name="Active", category="banking", owner="John", is_active=True
    )
    inactive_account = create_test_account(
        test_db_session, name="Inactive", category="banking", owner="John", is_active=False
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(test_db_session, snapshot.id, active_account.id, Decimal("1000.00"))
    create_test_snapshot_value(
        test_db_session, snapshot.id, inactive_account.id, Decimal("5000.00")
    )

    result = get_dashboard_data(test_db_session)

    # Only active account should be counted
    assert result.total_assets == 1000.0
    assert result.current_net_worth == 1000.0


def test_only_assets_no_liabilities(test_db_session):
    """Test dashboard with only assets (no liabilities)."""
    account = create_test_account(test_db_session, name="Savings", category="banking", owner="John")

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(test_db_session, snapshot.id, account.id, Decimal("1000.00"))

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 1000.0
    assert result.total_liabilities == 0.0
    assert result.current_net_worth == 1000.0


def test_only_liabilities_no_assets(test_db_session):
    """Test dashboard with only liabilities (no assets)."""
    account = create_test_account(
        test_db_session, name="Loan", account_type="liability", category="debt", owner="John"
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(test_db_session, snapshot.id, account.id, Decimal("1000.00"))

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 0.0
    assert result.total_liabilities == 1000.0
    assert result.current_net_worth == -1000.0
    assert result.allocation == []  # No assets to allocate


def test_assets_and_accounts_both_included(test_db_session):
    """Test that both Asset table and Account table values are included in net worth."""
    asset1 = create_test_asset(test_db_session, name="Car")
    asset2 = create_test_asset(test_db_session, name="Electronics")

    account_asset = create_test_account(
        test_db_session, name="Bank Account", category="banking", owner="John"
    )
    account_liability = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="housing",
        owner="John",
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1), notes="Test")

    # Create values for both Assets and Accounts
    create_test_snapshot_value(test_db_session, snapshot.id, account_asset.id, Decimal("50000.00"))
    create_test_snapshot_value(
        test_db_session, snapshot.id, account_liability.id, Decimal("30000.00")
    )
    # Asset-only snapshot values (no account_id)
    asset_sv1 = SnapshotValue(
        snapshot_id=snapshot.id, asset_id=asset1.id, value=Decimal("15000.00")
    )
    asset_sv2 = SnapshotValue(snapshot_id=snapshot.id, asset_id=asset2.id, value=Decimal("5000.00"))
    test_db_session.add_all([asset_sv1, asset_sv2])
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Total assets should include both Asset table (15000 + 5000) and Account assets (50000)
    assert result.total_assets == 70000.0
    assert result.total_liabilities == 30000.0
    # Net worth should include Asset table values: 15000 + 5000 + 50000 - 30000 = 40000
    assert result.current_net_worth == 40000.0
    assert len(result.net_worth_history) == 1
    assert result.net_worth_history[0].value == 40000.0


def test_retirement_account_value_calculation(test_db_session):
    """Test that retirement account value is calculated correctly."""
    # Create retirement and non-retirement accounts
    retirement_account = Account(
        name="IKE Account",
        type="asset",
        category="fund",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
    )
    general_account = Account(
        name="Bank Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    emergency_account = Account(
        name="Emergency Fund",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="emergency_fund",
    )
    test_db_session.add_all([retirement_account, general_account, emergency_account])
    test_db_session.commit()

    # Create snapshot with values
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=retirement_account.id, value=Decimal("50000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=general_account.id, value=Decimal("10000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=emergency_account.id, value=Decimal("5000")
            ),
        ]
    )
    test_db_session.commit()

    # Get dashboard data
    result = get_dashboard_data(test_db_session)

    # Should only include retirement account in retirement_account_value
    assert result.retirement_account_value == 50000.0
    # Total assets should include all accounts
    assert result.total_assets == 65000.0
    assert result.current_net_worth == 65000.0


def test_metric_cards_calculation(test_db_session):
    """Test metric cards calculation with AppConfig."""
    from app.models.app_config import AppConfig
    from app.models.transaction import Transaction

    # Create AppConfig
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=8,
        allocation_commodities=2,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create accounts
    real_estate_account = Account(
        name="Apartment",
        type="asset",
        category="real_estate",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        square_meters=Decimal("65.50"),
    )
    emergency_fund_account = Account(
        name="Emergency Fund",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="emergency_fund",
    )
    retirement_account = Account(
        name="IKE",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="IKE",
    )
    mortgage_account = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    investment_account = Account(
        name="Stocks",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all(
        [
            real_estate_account,
            emergency_fund_account,
            retirement_account,
            mortgage_account,
            investment_account,
        ]
    )
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=real_estate_account.id, value=Decimal("500000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id,
                account_id=emergency_fund_account.id,
                value=Decimal("15000"),
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=retirement_account.id, value=Decimal("100000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=mortgage_account.id, value=Decimal("300000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=investment_account.id, value=Decimal("50000")
            ),
        ]
    )
    test_db_session.commit()

    # Add transactions for investment contributions
    test_db_session.add_all(
        [
            Transaction(
                account_id=investment_account.id,
                amount=Decimal("10000"),
                date=date(2024, 1, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=retirement_account.id,
                amount=Decimal("20000"),
                date=date(2024, 1, 10),
                owner="Marcin",
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Check metric cards
    # Property sqm adjusted for mortgage equity: 65.50 * ((500000 - 300000) / 500000) = 26.2
    assert round(result.metric_cards.property_sqm, 2) == 26.2
    assert result.metric_cards.emergency_fund_months == 3.0  # 15000 / 5000
    assert result.metric_cards.retirement_income_monthly == (100000 * 0.04) / 12
    assert result.metric_cards.mortgage_remaining == 300000.0
    assert result.metric_cards.mortgage_months_left == 100  # 300000 / 3000
    assert result.metric_cards.mortgage_years_left == 100 / 12
    assert result.metric_cards.retirement_total == 100000.0
    assert result.metric_cards.investment_contributions == 30000.0  # 10000 + 20000
    assert result.metric_cards.investment_returns == 120000.0  # (50000 + 100000) - 30000


def test_metric_cards_null_square_meters(test_db_session):
    """Test metric cards with null square_meters in real estate account."""
    from app.models.app_config import AppConfig

    # Create AppConfig
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=8,
        allocation_commodities=2,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create real estate account with null square_meters
    real_estate_account = Account(
        name="Apartment",
        type="asset",
        category="real_estate",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        square_meters=None,  # Null square meters
    )
    mortgage_account = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([real_estate_account, mortgage_account])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=real_estate_account.id, value=Decimal("500000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=mortgage_account.id, value=Decimal("300000")
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Property sqm should be 0.0 when square_meters is null
    assert result.metric_cards.property_sqm == 0.0
    assert result.metric_cards.mortgage_remaining == 300000.0


def test_metric_cards_mixed_liabilities(test_db_session):
    """Test metric cards with mixed liability types - only housing affects property equity."""
    from app.models.app_config import AppConfig

    # Create AppConfig
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=8,
        allocation_commodities=2,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create accounts with multiple liability types
    real_estate_account = Account(
        name="Apartment",
        type="asset",
        category="real_estate",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        square_meters=Decimal("80.0"),
    )
    mortgage_account = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    car_loan_account = Account(
        name="Car Loan",
        type="liability",
        category="installment",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    credit_card_account = Account(
        name="Credit Card",
        type="liability",
        category="other",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all(
        [real_estate_account, mortgage_account, car_loan_account, credit_card_account]
    )
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=real_estate_account.id, value=Decimal("500000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=mortgage_account.id, value=Decimal("300000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=car_loan_account.id, value=Decimal("50000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=credit_card_account.id, value=Decimal("10000")
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Property sqm should only be affected by housing mortgage, not car loan or credit card
    # Equity: (500000 - 300000) / 500000 = 0.4
    # Owned sqm: 80.0 * 0.4 = 32.0
    assert result.metric_cards.property_sqm == 32.0
    assert result.metric_cards.mortgage_remaining == 300000.0


def test_metric_cards_underwater_mortgage(test_db_session):
    """Test metric cards with underwater mortgage - liability exceeds property value."""
    from app.models.app_config import AppConfig

    # Create AppConfig
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=8,
        allocation_commodities=2,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create real estate account with underwater mortgage
    real_estate_account = Account(
        name="Apartment",
        type="asset",
        category="real_estate",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        square_meters=Decimal("60.0"),
    )
    mortgage_account = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([real_estate_account, mortgage_account])
    test_db_session.commit()

    # Create snapshot with underwater mortgage (liability > asset)
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add snapshot values - mortgage exceeds property value
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=real_estate_account.id, value=Decimal("300000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=mortgage_account.id, value=Decimal("400000")
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Property sqm should be 0.0 (not negative) when underwater
    # Equity: (300000 - 400000) / 300000 = -0.33, clamped to 0
    assert result.metric_cards.property_sqm == 0.0
    assert result.metric_cards.mortgage_remaining == 400000.0


def test_metric_cards_no_mortgage(test_db_session):
    """Test metric cards with no mortgage - 100% ownership."""
    from app.models.app_config import AppConfig

    # Create AppConfig
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=8,
        allocation_commodities=2,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create real estate account with no mortgage
    real_estate_account = Account(
        name="Apartment",
        type="asset",
        category="real_estate",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        square_meters=Decimal("75.5"),
    )
    test_db_session.add(real_estate_account)
    test_db_session.commit()

    # Create snapshot with no mortgage
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add snapshot values - no mortgage
    test_db_session.add(
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=real_estate_account.id, value=Decimal("600000")
        )
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Property sqm should be 100% owned (full square meters)
    # Equity: (600000 - 0) / 600000 = 1.0
    # Owned sqm: 75.5 * 1.0 = 75.5
    assert result.metric_cards.property_sqm == 75.5
    assert result.metric_cards.mortgage_remaining == 0.0


def test_allocation_analysis(test_db_session):
    """Test allocation analysis with investment accounts."""
    from app.models.app_config import AppConfig

    # Create AppConfig with allocation targets
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=67,
        retirement_monthly_salary=Decimal("8000"),
        allocation_real_estate=20,
        allocation_stocks=60,
        allocation_bonds=30,
        allocation_gold=10,
        allocation_commodities=0,
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("3000"),
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create investment accounts
    stock_account = Account(
        name="Stocks",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    bond_account = Account(
        name="Bonds",
        type="asset",
        category="bond",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    gold_account = Account(
        name="Gold",
        type="asset",
        category="gold",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    ppk_account = Account(
        name="PPK",
        type="asset",
        category="ppk",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="PPK",
    )
    ike_account = Account(
        name="IKE",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="IKE",
    )
    test_db_session.add_all([stock_account, bond_account, gold_account, ppk_account, ike_account])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add values: stocks 40%, bonds 40%, gold 20% (stocks under-allocated)
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=stock_account.id, value=Decimal("20000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=bond_account.id, value=Decimal("20000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=gold_account.id, value=Decimal("10000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=ppk_account.id, value=Decimal("5000")
            ),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=ike_account.id, value=Decimal("15000")
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Check allocation analysis by category (excludes PPK)
    # Total: stock (20000 + 15000 IKE) + bonds (20000) + gold (10000) = 65000
    assert result.allocation_analysis.total_investment_value == 65000.0
    assert len(result.allocation_analysis.by_category) == 3  # stocks, bonds, gold

    # Find allocations
    stocks_alloc = next(a for a in result.allocation_analysis.by_category if a.category == "stocks")
    bonds_alloc = next(a for a in result.allocation_analysis.by_category if a.category == "bonds")
    gold_alloc = next(a for a in result.allocation_analysis.by_category if a.category == "gold")

    # Stocks: 35000 / 65000 = 53.85% (target 60%, -6.15% under-allocated)
    assert stocks_alloc.current_value == 35000.0
    assert abs(stocks_alloc.current_percentage - 53.85) < 0.1
    assert stocks_alloc.target_percentage == 60.0
    assert abs(stocks_alloc.difference - (-6.15)) < 0.1

    # Bonds: 20000 / 65000 = 30.77% (target 30%, +0.77%)
    assert bonds_alloc.current_value == 20000.0
    assert abs(bonds_alloc.current_percentage - 30.77) < 0.1
    assert bonds_alloc.target_percentage == 30.0
    assert abs(bonds_alloc.difference - 0.77) < 0.1

    # Gold: 10000 / 65000 = 15.38% (target 10%, +5.38%)
    assert gold_alloc.current_value == 10000.0
    assert abs(gold_alloc.current_percentage - 15.38) < 0.1
    assert gold_alloc.target_percentage == 10.0
    assert abs(gold_alloc.difference - 5.38) < 0.1

    # Check wrapper breakdown (includes PPK: 70000 total)
    assert len(result.allocation_analysis.by_wrapper) > 0
    wrapper_total = sum(w.value for w in result.allocation_analysis.by_wrapper)
    assert wrapper_total == 70000.0  # Including PPK

    # Check wrapper percentages sum to 100%
    wrapper_pct_sum = sum(w.percentage for w in result.allocation_analysis.by_wrapper)
    assert abs(wrapper_pct_sum - 100.0) < 0.01

    # Rebalancing: Stocks under-allocated, should have buy suggestion
    assert len(result.allocation_analysis.rebalancing) > 0
    stocks_rebal = next(
        (r for r in result.allocation_analysis.rebalancing if r.category == "stocks"), None
    )
    assert stocks_rebal is not None
    assert stocks_rebal.action == "buy"


def test_investment_time_series(test_db_session):
    """Test investment time series with transactions."""
    from app.models.transaction import Transaction

    # Create investment account
    investment_account = Account(
        name="Stocks",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(investment_account)
    test_db_session.commit()

    # Create snapshots
    snapshot1 = Snapshot(date=date(2024, 1, 31))
    snapshot2 = Snapshot(date=date(2024, 2, 29))
    test_db_session.add_all([snapshot1, snapshot2])
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=investment_account.id, value=Decimal("10500")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=investment_account.id, value=Decimal("21200")
            ),
        ]
    )
    test_db_session.commit()

    # Add transactions
    test_db_session.add_all(
        [
            Transaction(
                account_id=investment_account.id,
                amount=Decimal("10000"),
                date=date(2024, 1, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=investment_account.id,
                amount=Decimal("10000"),
                date=date(2024, 2, 15),
                owner="Marcin",
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Check time series
    assert len(result.investment_time_series) == 2

    # Jan: value=10500, contributions=10000, returns=500
    assert result.investment_time_series[0].date == date(2024, 1, 31)
    assert result.investment_time_series[0].value == 10500.0
    assert result.investment_time_series[0].contributions == 10000.0
    assert result.investment_time_series[0].returns == 500.0

    # Feb: value=21200, contributions=20000, returns=1200
    assert result.investment_time_series[1].date == date(2024, 2, 29)
    assert result.investment_time_series[1].value == 21200.0
    assert result.investment_time_series[1].contributions == 20000.0
    assert result.investment_time_series[1].returns == 1200.0


def test_wrapper_time_series(test_db_session):
    """Test wrapper-specific time series (IKE, IKZE, PPK)."""
    from app.models.transaction import Transaction

    # Create wrapper accounts
    ike_account = Account(
        name="IKE",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="IKE",
    )
    ikze_account = Account(
        name="IKZE",
        type="asset",
        category="bond",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="IKZE",
    )
    ppk_account = Account(
        name="PPK",
        type="asset",
        category="ppk",
        owner="Marcin",
        currency="PLN",
        purpose="retirement",
        account_wrapper="PPK",
    )
    test_db_session.add_all([ike_account, ikze_account, ppk_account])
    test_db_session.commit()

    # Create snapshots
    snapshot1 = Snapshot(date=date(2024, 1, 31))
    snapshot2 = Snapshot(date=date(2024, 2, 29))
    test_db_session.add_all([snapshot1, snapshot2])
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            # Jan
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=ike_account.id, value=Decimal("10500")
            ),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=ikze_account.id, value=Decimal("5200")
            ),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=ppk_account.id, value=Decimal("3100")
            ),
            # Feb
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=ike_account.id, value=Decimal("21200")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=ikze_account.id, value=Decimal("10500")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=ppk_account.id, value=Decimal("6300")
            ),
        ]
    )
    test_db_session.commit()

    # Add transactions
    test_db_session.add_all(
        [
            Transaction(
                account_id=ike_account.id,
                amount=Decimal("10000"),
                date=date(2024, 1, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=ike_account.id,
                amount=Decimal("10000"),
                date=date(2024, 2, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=ikze_account.id,
                amount=Decimal("5000"),
                date=date(2024, 1, 10),
                owner="Marcin",
            ),
            Transaction(
                account_id=ikze_account.id,
                amount=Decimal("5000"),
                date=date(2024, 2, 10),
                owner="Marcin",
            ),
            Transaction(
                account_id=ppk_account.id,
                amount=Decimal("3000"),
                date=date(2024, 1, 5),
                owner="Marcin",
            ),
            Transaction(
                account_id=ppk_account.id,
                amount=Decimal("3000"),
                date=date(2024, 2, 5),
                owner="Marcin",
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Check IKE time series
    assert len(result.wrapper_time_series.ike) == 2
    assert result.wrapper_time_series.ike[0].value == 10500.0
    assert result.wrapper_time_series.ike[0].contributions == 10000.0
    assert result.wrapper_time_series.ike[0].returns == 500.0
    assert result.wrapper_time_series.ike[1].value == 21200.0
    assert result.wrapper_time_series.ike[1].contributions == 20000.0
    assert result.wrapper_time_series.ike[1].returns == 1200.0

    # Check IKZE time series
    assert len(result.wrapper_time_series.ikze) == 2
    assert result.wrapper_time_series.ikze[0].value == 5200.0
    assert result.wrapper_time_series.ikze[0].contributions == 5000.0
    assert result.wrapper_time_series.ikze[0].returns == 200.0

    # Check PPK time series
    assert len(result.wrapper_time_series.ppk) == 2
    assert result.wrapper_time_series.ppk[0].value == 3100.0
    assert result.wrapper_time_series.ppk[0].contributions == 3000.0
    assert result.wrapper_time_series.ppk[0].returns == 100.0


def test_category_time_series_with_grouping(test_db_session):
    """Test category time series with fund/etf grouped as stock."""
    from app.models.transaction import Transaction

    # Create accounts with different categories
    stock_account = Account(
        name="Stocks",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    fund_account = Account(
        name="Fund",
        type="asset",
        category="fund",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    etf_account = Account(
        name="ETF",
        type="asset",
        category="etf",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    bond_account = Account(
        name="Bonds",
        type="asset",
        category="bond",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([stock_account, fund_account, etf_account, bond_account])
    test_db_session.commit()

    # Create snapshots
    snapshot1 = Snapshot(date=date(2024, 1, 31))
    snapshot2 = Snapshot(date=date(2024, 2, 29))
    test_db_session.add_all([snapshot1, snapshot2])
    test_db_session.commit()

    # Add snapshot values
    test_db_session.add_all(
        [
            # Jan
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=stock_account.id, value=Decimal("10000")
            ),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=fund_account.id, value=Decimal("5000")
            ),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=etf_account.id, value=Decimal("3000")
            ),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=bond_account.id, value=Decimal("8000")
            ),
            # Feb
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=stock_account.id, value=Decimal("20500")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=fund_account.id, value=Decimal("10300")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=etf_account.id, value=Decimal("6200")
            ),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=bond_account.id, value=Decimal("16500")
            ),
        ]
    )
    test_db_session.commit()

    # Add transactions
    test_db_session.add_all(
        [
            Transaction(
                account_id=stock_account.id,
                amount=Decimal("10000"),
                date=date(2024, 1, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=stock_account.id,
                amount=Decimal("10000"),
                date=date(2024, 2, 15),
                owner="Marcin",
            ),
            Transaction(
                account_id=fund_account.id,
                amount=Decimal("5000"),
                date=date(2024, 1, 10),
                owner="Marcin",
            ),
            Transaction(
                account_id=fund_account.id,
                amount=Decimal("5000"),
                date=date(2024, 2, 10),
                owner="Marcin",
            ),
            Transaction(
                account_id=etf_account.id,
                amount=Decimal("3000"),
                date=date(2024, 1, 5),
                owner="Marcin",
            ),
            Transaction(
                account_id=etf_account.id,
                amount=Decimal("3000"),
                date=date(2024, 2, 5),
                owner="Marcin",
            ),
            Transaction(
                account_id=bond_account.id,
                amount=Decimal("8000"),
                date=date(2024, 1, 20),
                owner="Marcin",
            ),
            Transaction(
                account_id=bond_account.id,
                amount=Decimal("8000"),
                date=date(2024, 2, 20),
                owner="Marcin",
            ),
        ]
    )
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Check stock time series (includes stock + fund + etf)
    assert len(result.category_time_series.stock) == 2
    # Jan: 10000 + 5000 + 3000 = 18000
    assert result.category_time_series.stock[0].value == 18000.0
    assert result.category_time_series.stock[0].contributions == 18000.0  # 10000 + 5000 + 3000
    assert result.category_time_series.stock[0].returns == 0.0
    # Feb: 20500 + 10300 + 6200 = 37000
    assert result.category_time_series.stock[1].value == 37000.0
    assert (
        result.category_time_series.stock[1].contributions == 36000.0
    )  # 10000*2 + 5000*2 + 3000*2
    assert result.category_time_series.stock[1].returns == 1000.0  # 37000 - 36000

    # Check bond time series
    assert len(result.category_time_series.bond) == 2
    assert result.category_time_series.bond[0].value == 8000.0
    assert result.category_time_series.bond[0].contributions == 8000.0
    assert result.category_time_series.bond[0].returns == 0.0
    assert result.category_time_series.bond[1].value == 16500.0
    assert result.category_time_series.bond[1].contributions == 16000.0
    assert result.category_time_series.bond[1].returns == 500.0


def test_calculate_savings_rate_happy_path(test_db_session):
    """Test savings rate calculation with sufficient data."""
    # Create account
    account = Account(
        name="Savings",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create 4 snapshots (need 4 to calculate 3 deltas)
    snapshots = [
        Snapshot(date=date(2024, 1, 1), notes="January"),
        Snapshot(date=date(2024, 2, 1), notes="February"),
        Snapshot(date=date(2024, 3, 1), notes="March"),
        Snapshot(date=date(2024, 4, 1), notes="April"),
    ]
    test_db_session.add_all(snapshots)
    test_db_session.commit()

    # Create values with increasing net worth (5000, 10000, 15000, 20000)
    # Deltas: 5000, 5000, 5000 - avg delta = 5000
    values = []
    for i, snapshot in enumerate(snapshots):
        values.append(
            SnapshotValue(
                snapshot_id=snapshot.id,
                account_id=account.id,
                value=Decimal((i + 1) * 5000),
            )
        )
    test_db_session.add_all(values)
    test_db_session.commit()

    # Create 3 salary records with gross amount 10000 each
    salaries = [
        SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
        SalaryRecord(
            date=date(2024, 2, 1),
            gross_amount=Decimal("10000"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
        SalaryRecord(
            date=date(2024, 3, 1),
            gross_amount=Decimal("10000"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
    ]
    test_db_session.add_all(salaries)
    test_db_session.commit()

    # Prepare data for function call
    import pandas as pd

    snapshots_query = test_db_session.query(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query.statement, test_db_session.get_bind())

    values_query = test_db_session.query(SnapshotValue)
    values_df = pd.read_sql(values_query.statement, test_db_session.get_bind())

    accounts_query = test_db_session.query(Account)
    accounts_df = pd.read_sql(accounts_query.statement, test_db_session.get_bind())

    # Merge data
    df = values_df.merge(
        accounts_df, left_on="account_id", right_on="id", how="left", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    result = _calculate_savings_rate(snapshots_df, df, test_db_session)

    # avg delta = 5000, avg salary = 10000
    # savings rate = (5000 / 10000) * 100 = 50%
    assert result is not None
    assert result == 50.0


def test_calculate_savings_rate_insufficient_snapshots(test_db_session):
    """Test savings rate returns None with < 4 snapshots."""
    # Create account
    account = Account(
        name="Savings",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create only 3 snapshots (not enough)
    snapshots = [
        Snapshot(date=date(2024, 1, 1), notes="January"),
        Snapshot(date=date(2024, 2, 1), notes="February"),
        Snapshot(date=date(2024, 3, 1), notes="March"),
    ]
    test_db_session.add_all(snapshots)
    test_db_session.commit()

    import pandas as pd

    snapshots_query = test_db_session.query(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query.statement, test_db_session.get_bind())

    df = pd.DataFrame()  # Empty df, doesn't matter for this test

    result = _calculate_savings_rate(snapshots_df, df, test_db_session)

    assert result is None


def test_calculate_savings_rate_no_salaries(test_db_session):
    """Test savings rate returns None with no salary records."""
    # Create account and 4 snapshots
    account = Account(
        name="Savings",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    snapshots = [
        Snapshot(date=date(2024, 1, 1), notes="January"),
        Snapshot(date=date(2024, 2, 1), notes="February"),
        Snapshot(date=date(2024, 3, 1), notes="March"),
        Snapshot(date=date(2024, 4, 1), notes="April"),
    ]
    test_db_session.add_all(snapshots)
    test_db_session.commit()

    values = []
    for i, snapshot in enumerate(snapshots):
        values.append(
            SnapshotValue(
                snapshot_id=snapshot.id,
                account_id=account.id,
                value=Decimal((i + 1) * 5000),
            )
        )
    test_db_session.add_all(values)
    test_db_session.commit()

    # No salaries created

    import pandas as pd

    snapshots_query = test_db_session.query(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query.statement, test_db_session.get_bind())

    values_query = test_db_session.query(SnapshotValue)
    values_df = pd.read_sql(values_query.statement, test_db_session.get_bind())

    accounts_query = test_db_session.query(Account)
    accounts_df = pd.read_sql(accounts_query.statement, test_db_session.get_bind())

    df = values_df.merge(
        accounts_df, left_on="account_id", right_on="id", how="left", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    result = _calculate_savings_rate(snapshots_df, df, test_db_session)

    assert result is None


def test_calculate_savings_rate_zero_salary(test_db_session):
    """Test savings rate returns None with zero salary."""
    # Create account and 4 snapshots
    account = Account(
        name="Savings",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    snapshots = [
        Snapshot(date=date(2024, 1, 1), notes="January"),
        Snapshot(date=date(2024, 2, 1), notes="February"),
        Snapshot(date=date(2024, 3, 1), notes="March"),
        Snapshot(date=date(2024, 4, 1), notes="April"),
    ]
    test_db_session.add_all(snapshots)
    test_db_session.commit()

    values = []
    for i, snapshot in enumerate(snapshots):
        values.append(
            SnapshotValue(
                snapshot_id=snapshot.id,
                account_id=account.id,
                value=Decimal((i + 1) * 5000),
            )
        )
    test_db_session.add_all(values)
    test_db_session.commit()

    # Create salaries with zero gross amount
    salaries = [
        SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("0"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
        SalaryRecord(
            date=date(2024, 2, 1),
            gross_amount=Decimal("0"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
        SalaryRecord(
            date=date(2024, 3, 1),
            gross_amount=Decimal("0"),
            contract_type="B2B",
            company="Company A",
            owner="John",
            is_active=True,
        ),
    ]
    test_db_session.add_all(salaries)
    test_db_session.commit()

    import pandas as pd

    snapshots_query = test_db_session.query(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query.statement, test_db_session.get_bind())

    values_query = test_db_session.query(SnapshotValue)
    values_df = pd.read_sql(values_query.statement, test_db_session.get_bind())

    accounts_query = test_db_session.query(Account)
    accounts_df = pd.read_sql(accounts_query.statement, test_db_session.get_bind())

    df = values_df.merge(
        accounts_df, left_on="account_id", right_on="id", how="left", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    result = _calculate_savings_rate(snapshots_df, df, test_db_session)

    assert result is None


def test_calculate_debt_to_income_happy_path(test_db_session):
    """Test debt-to-income ratio calculation with valid data."""
    # Create app config with mortgage payment
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("2850"),
        allocation_real_estate=0,
        allocation_stocks=70,
        allocation_bonds=25,
        allocation_gold=5,
        allocation_commodities=0,
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create salary record
    salary = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=Decimal("10000"),
        contract_type="B2B",
        company="Company A",
        owner="John",
        is_active=True,
    )
    test_db_session.add(salary)
    test_db_session.commit()

    result = _calculate_debt_to_income(test_db_session)

    # DTI = (2850 / 10000) * 100 = 28.5%
    assert result is not None
    assert round(result, 1) == 28.5


def test_calculate_debt_to_income_no_config(test_db_session):
    """Test debt-to-income returns None with no config."""
    # Create salary but no config
    salary = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=Decimal("10000"),
        contract_type="B2B",
        company="Company A",
        owner="John",
        is_active=True,
    )
    test_db_session.add(salary)
    test_db_session.commit()

    result = _calculate_debt_to_income(test_db_session)

    assert result is None


def test_calculate_debt_to_income_no_mortgage_payment(test_db_session):
    """Test debt-to-income returns None with zero mortgage payment."""
    # Create config with zero mortgage payment
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("0"),
        allocation_real_estate=0,
        allocation_stocks=70,
        allocation_bonds=25,
        allocation_gold=5,
        allocation_commodities=0,
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create salary
    salary = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=Decimal("10000"),
        contract_type="B2B",
        company="Company A",
        owner="John",
        is_active=True,
    )
    test_db_session.add(salary)
    test_db_session.commit()

    result = _calculate_debt_to_income(test_db_session)

    assert result is None


def test_calculate_debt_to_income_no_salary(test_db_session):
    """Test debt-to-income returns None with no salary records."""
    # Create config
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("2850"),
        allocation_real_estate=0,
        allocation_stocks=70,
        allocation_bonds=25,
        allocation_gold=5,
        allocation_commodities=0,
    )
    test_db_session.add(config)
    test_db_session.commit()

    # No salary created

    result = _calculate_debt_to_income(test_db_session)

    assert result is None


def test_calculate_debt_to_income_zero_salary(test_db_session):
    """Test debt-to-income returns None with zero salary."""
    # Create config
    config = AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        monthly_expenses=Decimal("5000"),
        monthly_mortgage_payment=Decimal("2850"),
        allocation_real_estate=0,
        allocation_stocks=70,
        allocation_bonds=25,
        allocation_gold=5,
        allocation_commodities=0,
    )
    test_db_session.add(config)
    test_db_session.commit()

    # Create salary with zero gross amount
    salary = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=Decimal("0"),
        contract_type="B2B",
        company="Company A",
        owner="John",
        is_active=True,
    )
    test_db_session.add(salary)
    test_db_session.commit()

    result = _calculate_debt_to_income(test_db_session)

    assert result is None
