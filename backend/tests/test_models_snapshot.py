from datetime import date
from decimal import Decimal

import pytest
from sqlalchemy.exc import IntegrityError

from app.models.account import Account
from app.models.snapshot import Snapshot, SnapshotValue


def test_snapshot_creation(test_db_session):
    """Test creating a snapshot with required fields."""
    snapshot = Snapshot(date=date(2024, 1, 1), notes="January snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

    assert snapshot.id is not None
    assert snapshot.date == date(2024, 1, 1)
    assert snapshot.notes == "January snapshot"
    assert snapshot.created_at is not None


def test_snapshot_unique_date_constraint(test_db_session):
    """Test that snapshot dates must be unique."""
    snapshot1 = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot1)
    test_db_session.commit()

    snapshot2 = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot2)

    with pytest.raises(IntegrityError):
        test_db_session.commit()


def test_snapshot_optional_notes(test_db_session):
    """Test creating a snapshot without notes."""
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot)
    test_db_session.commit()

    assert snapshot.notes is None


def test_snapshot_value_creation(test_db_session):
    """Test creating a snapshot value with all required fields."""
    account = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add_all([account, snapshot])
    test_db_session.commit()

    snapshot_value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.50")
    )
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    assert snapshot_value.id is not None
    assert snapshot_value.snapshot_id == snapshot.id
    assert snapshot_value.account_id == account.id
    assert snapshot_value.value == Decimal("1000.50")


def test_snapshot_value_unique_constraint(test_db_session):
    """Test that snapshot-account combination must be unique."""
    account = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add_all([account, snapshot])
    test_db_session.commit()

    value1 = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00"))
    test_db_session.add(value1)
    test_db_session.commit()

    value2 = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("2000.00"))
    test_db_session.add(value2)

    with pytest.raises(IntegrityError):
        test_db_session.commit()


def test_snapshot_cascade_delete(test_db_session):
    """Test that deleting a snapshot cascades to snapshot values."""
    account = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add_all([account, snapshot])
    test_db_session.commit()

    snapshot_value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00")
    )
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    snapshot_value_id = snapshot_value.id
    test_db_session.delete(snapshot)
    test_db_session.commit()

    deleted_value = test_db_session.get(SnapshotValue, snapshot_value_id)
    assert deleted_value is None


def test_account_cascade_delete(test_db_session):
    """Test that deleting an account cascades to snapshot values."""
    account = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add_all([account, snapshot])
    test_db_session.commit()

    snapshot_value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00")
    )
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    snapshot_value_id = snapshot_value.id
    test_db_session.delete(account)
    test_db_session.commit()

    deleted_value = test_db_session.get(SnapshotValue, snapshot_value_id)
    assert deleted_value is None


def test_snapshot_value_decimal_precision(test_db_session):
    """Test that snapshot values maintain decimal precision."""
    account = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    snapshot = Snapshot(date=date(2024, 1, 1))
    test_db_session.add_all([account, snapshot])
    test_db_session.commit()

    snapshot_value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("12345678901234.56")
    )
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    retrieved_value = test_db_session.get(SnapshotValue, snapshot_value.id)
    assert retrieved_value.value == Decimal("12345678901234.56")
