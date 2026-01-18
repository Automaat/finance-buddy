from datetime import date
from decimal import Decimal

from app.models.account import Account
from app.models.asset import Asset
from app.models.snapshot import Snapshot, SnapshotValue
from app.services.dashboard import get_dashboard_data


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
    # Create accounts
    asset_account = Account(
        name="Savings",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    liability_account = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add_all([asset_account, liability_account])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 1), notes="January")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create values
    values = [
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=asset_account.id, value=Decimal("10000.00")
        ),
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=liability_account.id, value=Decimal("5000.00")
        ),
    ]
    test_db_session.add_all(values)
    test_db_session.commit()

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

    # Create snapshots
    snapshots = [
        Snapshot(date=date(2024, 1, 1), notes="January"),
        Snapshot(date=date(2024, 2, 1), notes="February"),
        Snapshot(date=date(2024, 3, 1), notes="March"),
    ]
    test_db_session.add_all(snapshots)
    test_db_session.commit()

    # Create values with increasing amounts
    values = [
        SnapshotValue(snapshot_id=snapshots[0].id, account_id=account.id, value=Decimal("1000.00")),
        SnapshotValue(snapshot_id=snapshots[1].id, account_id=account.id, value=Decimal("1500.00")),
        SnapshotValue(snapshot_id=snapshots[2].id, account_id=account.id, value=Decimal("2000.00")),
    ]
    test_db_session.add_all(values)
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    assert result.current_net_worth == 2000.0
    assert result.change_vs_last_month == 500.0  # 2000 - 1500
    assert len(result.net_worth_history) == 3
    assert result.net_worth_history[0].value == 1000.0
    assert result.net_worth_history[1].value == 1500.0
    assert result.net_worth_history[2].value == 2000.0


def test_asset_liability_aggregation(test_db_session):
    """Test aggregation of assets and liabilities by type."""
    # Create multiple asset accounts
    accounts = [
        Account(
            name="Savings",
            type="asset",
            category="banking",
            owner="John",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
        Account(
            name="Stocks",
            type="asset",
            category="investments",
            owner="John",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
        Account(
            name="Credit Card",
            type="liability",
            category="credit",
            owner="John",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
    ]
    test_db_session.add_all(accounts)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    values = [
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[0].id, value=Decimal("5000.00")),
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[1].id, value=Decimal("3000.00")),
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[2].id, value=Decimal("1000.00")),
    ]
    test_db_session.add_all(values)
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 8000.0  # 5000 + 3000
    assert result.total_liabilities == 1000.0
    assert result.current_net_worth == 7000.0


def test_allocation_by_category_and_owner(test_db_session):
    """Test asset allocation aggregated by category and owner."""
    # Create accounts with different categories and owners
    accounts = [
        Account(
            name="John Savings",
            type="asset",
            category="banking",
            owner="John",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
        Account(
            name="Jane Savings",
            type="asset",
            category="banking",
            owner="Jane",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
        Account(
            name="John Stocks",
            type="asset",
            category="investments",
            owner="John",
            currency="PLN",
            is_active=True,
            purpose="general",
        ),
    ]
    test_db_session.add_all(accounts)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    values = [
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[0].id, value=Decimal("2000.00")),
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[1].id, value=Decimal("3000.00")),
        SnapshotValue(snapshot_id=snapshot.id, account_id=accounts[2].id, value=Decimal("5000.00")),
    ]
    test_db_session.add_all(values)
    test_db_session.commit()

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
    # Create snapshot with no values
    snapshot = Snapshot(date=date(2024, 1, 1), notes="Empty snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

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
    # Create active and inactive accounts
    active_account = Account(
        name="Active",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    inactive_account = Account(
        name="Inactive",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=False,
        purpose="general",
    )
    test_db_session.add_all([active_account, inactive_account])
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    values = [
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=active_account.id, value=Decimal("1000.00")
        ),
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=inactive_account.id, value=Decimal("5000.00")
        ),
    ]
    test_db_session.add_all(values)
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    # Only active account should be counted
    assert result.total_assets == 1000.0
    assert result.current_net_worth == 1000.0


def test_only_assets_no_liabilities(test_db_session):
    """Test dashboard with only assets (no liabilities)."""
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

    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00"))
    test_db_session.add(value)
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 1000.0
    assert result.total_liabilities == 0.0
    assert result.current_net_worth == 1000.0


def test_only_liabilities_no_assets(test_db_session):
    """Test dashboard with only liabilities (no assets)."""
    account = Account(
        name="Loan",
        type="liability",
        category="debt",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00"))
    test_db_session.add(value)
    test_db_session.commit()

    result = get_dashboard_data(test_db_session)

    assert result.total_assets == 0.0
    assert result.total_liabilities == 1000.0
    assert result.current_net_worth == -1000.0
    assert result.allocation == []  # No assets to allocate


def test_assets_and_accounts_both_included(test_db_session):
    """Test that both Asset table and Account table values are included in net worth."""
    # Create Asset table entries
    asset1 = Asset(name="Car", is_active=True)
    asset2 = Asset(name="Electronics", is_active=True)
    test_db_session.add_all([asset1, asset2])
    test_db_session.commit()

    # Create Account table entries
    account_asset = Account(
        name="Bank Account",
        type="asset",
        category="banking",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    account_liability = Account(
        name="Mortgage",
        type="liability",
        category="housing",
        owner="John",
        currency="PLN",
        is_active=True,
        purpose="general",
    )
    test_db_session.add_all([account_asset, account_liability])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 1), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create values for both Assets and Accounts
    values = [
        SnapshotValue(snapshot_id=snapshot.id, asset_id=asset1.id, value=Decimal("15000.00")),
        SnapshotValue(snapshot_id=snapshot.id, asset_id=asset2.id, value=Decimal("5000.00")),
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=account_asset.id, value=Decimal("50000.00")
        ),
        SnapshotValue(
            snapshot_id=snapshot.id, account_id=account_liability.id, value=Decimal("30000.00")
        ),
    ]
    test_db_session.add_all(values)
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
