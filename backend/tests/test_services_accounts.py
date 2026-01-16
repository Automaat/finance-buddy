from datetime import date
from decimal import Decimal

from app.models import Account, Snapshot, SnapshotValue
from app.services.accounts import get_all_accounts


def test_get_all_accounts_with_values(test_db_session):
    """Test getting all accounts with their latest snapshot values"""
    # Create accounts
    asset1 = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    asset2 = Account(name="IKE", type="asset", category="ike", owner="Marcin", currency="PLN")
    liability = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
    )
    inactive = Account(
        name="Old Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        is_active=False,
    )
    test_db_session.add_all([asset1, asset2, liability, inactive])
    test_db_session.commit()

    # Create snapshot with values
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    test_db_session.add_all(
        [
            SnapshotValue(snapshot_id=snapshot.id, account_id=asset1.id, value=Decimal("50000")),
            SnapshotValue(snapshot_id=snapshot.id, account_id=asset2.id, value=Decimal("20000")),
            SnapshotValue(
                snapshot_id=snapshot.id, account_id=liability.id, value=Decimal("100000")
            ),
            SnapshotValue(snapshot_id=snapshot.id, account_id=inactive.id, value=Decimal("1000")),
        ]
    )
    test_db_session.commit()

    # Get all accounts
    result = get_all_accounts(test_db_session)

    # Should have 2 assets and 1 liability (inactive excluded)
    assert len(result.assets) == 2
    assert len(result.liabilities) == 1

    # Check asset values
    bank_account = next(a for a in result.assets if a.name == "Bank Account")
    assert bank_account.current_value == 50000.0
    assert bank_account.owner == "Marcin"
    assert bank_account.category == "bank"

    ike_account = next(a for a in result.assets if a.name == "IKE")
    assert ike_account.current_value == 20000.0

    # Check liability
    assert result.liabilities[0].name == "Mortgage"
    assert result.liabilities[0].current_value == 100000.0
    assert result.liabilities[0].owner == "Shared"


def test_get_all_accounts_no_snapshots(test_db_session):
    """Test getting accounts when no snapshots exist"""
    # Create accounts without snapshots
    asset = Account(
        name="Bank Account", type="asset", category="bank", owner="Test", currency="PLN"
    )
    liability = Account(
        name="Loan", type="liability", category="mortgage", owner="Test", currency="PLN"
    )
    test_db_session.add_all([asset, liability])
    test_db_session.commit()

    # Get all accounts
    result = get_all_accounts(test_db_session)

    # Should return accounts with 0 values
    assert len(result.assets) == 1
    assert len(result.liabilities) == 1
    assert result.assets[0].current_value == 0.0
    assert result.liabilities[0].current_value == 0.0


def test_get_all_accounts_partial_values(test_db_session):
    """Test getting accounts when only some have values in latest snapshot"""
    # Create accounts
    account1 = Account(
        name="Account 1", type="asset", category="bank", owner="Test", currency="PLN"
    )
    account2 = Account(
        name="Account 2", type="asset", category="bank", owner="Test", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create snapshot with only one account value
    snapshot = Snapshot(date=date(2024, 1, 31))
    test_db_session.add(snapshot)
    test_db_session.commit()

    test_db_session.add(
        SnapshotValue(snapshot_id=snapshot.id, account_id=account1.id, value=Decimal("1000"))
    )
    test_db_session.commit()

    # Get all accounts
    result = get_all_accounts(test_db_session)

    # Both should be returned, one with value, one with 0
    assert len(result.assets) == 2
    acc1 = next(a for a in result.assets if a.name == "Account 1")
    acc2 = next(a for a in result.assets if a.name == "Account 2")
    assert acc1.current_value == 1000.0
    assert acc2.current_value == 0.0


def test_get_all_accounts_empty_db(test_db_session):
    """Test getting accounts from empty database"""
    result = get_all_accounts(test_db_session)

    assert len(result.assets) == 0
    assert len(result.liabilities) == 0
