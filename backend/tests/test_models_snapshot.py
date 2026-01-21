from datetime import date
from decimal import Decimal

import pytest
from sqlalchemy.exc import IntegrityError

from app.models.snapshot import Snapshot, SnapshotValue
from tests.factories import create_test_account, create_test_snapshot, create_test_snapshot_value


def test_snapshot_creation(test_db_session):
    """Test creating a snapshot with required fields."""
    snapshot = create_test_snapshot(
        test_db_session, snapshot_date=date(2024, 1, 1), notes="January snapshot"
    )

    assert snapshot.id is not None
    assert snapshot.date == date(2024, 1, 1)
    assert snapshot.notes == "January snapshot"
    assert snapshot.created_at is not None


def test_snapshot_unique_date_constraint(test_db_session):
    """Test that snapshot dates must be unique."""
    create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    snapshot2 = Snapshot(date=date(2024, 1, 1))
    test_db_session.add(snapshot2)

    with pytest.raises(IntegrityError):
        test_db_session.commit()


def test_snapshot_optional_notes(test_db_session):
    """Test creating a snapshot without notes."""
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1), notes="")

    assert snapshot.notes == ""


def test_snapshot_value_creation(test_db_session):
    """Test creating a snapshot value with all required fields."""
    account = create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    snapshot_value = create_test_snapshot_value(
        test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.50")
    )

    assert snapshot_value.id is not None
    assert snapshot_value.snapshot_id == snapshot.id
    assert snapshot_value.account_id == account.id
    assert snapshot_value.value == Decimal("1000.50")


def test_snapshot_value_unique_constraint(test_db_session):
    """Test that snapshot-account combination must be unique."""
    account = create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    create_test_snapshot_value(
        test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00")
    )

    value2 = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("2000.00"))
    test_db_session.add(value2)

    with pytest.raises(IntegrityError):
        test_db_session.commit()


def test_snapshot_cascade_delete(test_db_session):
    """Test that deleting a snapshot cascades to snapshot values."""
    account = create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    snapshot_value = create_test_snapshot_value(
        test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00")
    )

    snapshot_value_id = snapshot_value.id
    test_db_session.delete(snapshot)
    test_db_session.commit()

    deleted_value = test_db_session.get(SnapshotValue, snapshot_value_id)
    assert deleted_value is None


def test_account_cascade_delete(test_db_session):
    """Test that deleting an account cascades to snapshot values."""
    account = create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    snapshot_value = create_test_snapshot_value(
        test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1000.00")
    )

    snapshot_value_id = snapshot_value.id
    test_db_session.delete(account)
    test_db_session.commit()

    deleted_value = test_db_session.get(SnapshotValue, snapshot_value_id)
    assert deleted_value is None


def test_snapshot_value_decimal_precision(test_db_session):
    """Test that snapshot values maintain decimal precision."""
    account = create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 1, 1))

    snapshot_value = create_test_snapshot_value(
        test_db_session,
        snapshot_id=snapshot.id,
        account_id=account.id,
        value=Decimal("12345678901234.56"),
    )

    retrieved_value = test_db_session.get(SnapshotValue, snapshot_value.id)
    assert retrieved_value.value == Decimal("12345678901234.56")
