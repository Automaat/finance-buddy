from datetime import date
from decimal import Decimal

import pytest
from fastapi import HTTPException

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.snapshots import SnapshotCreate, SnapshotValueInput
from app.services.snapshots import (
    create_snapshot,
    get_all_snapshots,
    get_latest_snapshot_values,
    get_snapshot_by_id,
)


def test_create_snapshot_success(test_db_session):
    """Test creating a snapshot with account values"""
    # Create test accounts
    account1 = Account(
        name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    account2 = Account(
        name="Test IKE", type="asset", category="ike", owner="Test", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create snapshot
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test snapshot",
        values=[
            SnapshotValueInput(account_id=account1.id, value=10000.50),
            SnapshotValueInput(account_id=account2.id, value=5000.25),
        ],
    )

    result = create_snapshot(test_db_session, data)

    assert result.date == date(2024, 1, 31)
    assert result.notes == "Test snapshot"
    assert len(result.values) == 2
    assert result.values[0].account_name == "Test Bank"
    assert result.values[0].value == 10000.50


def test_create_snapshot_duplicate_date(test_db_session):
    """Test creating snapshot with duplicate date fails"""
    # Create account
    account = Account(
        name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create first snapshot
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="First",
        values=[SnapshotValueInput(account_id=account.id, value=1000)],
    )
    create_snapshot(test_db_session, data)

    # Try to create duplicate
    data2 = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Duplicate",
        values=[SnapshotValueInput(account_id=account.id, value=2000)],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data2)

    assert exc_info.value.status_code == 400
    assert "already exists" in exc_info.value.detail


def test_create_snapshot_invalid_account(test_db_session):
    """Test creating snapshot with invalid account ID fails"""
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[SnapshotValueInput(account_id=999, value=1000)],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_get_all_snapshots(test_db_session):
    """Test getting all snapshots with net worth calculation"""
    # Create accounts
    asset = Account(
        name="Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    liability = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Test", currency="PLN"
    )
    test_db_session.add_all([asset, liability])
    test_db_session.commit()

    # Create snapshots
    snapshot1 = Snapshot(date=date(2024, 1, 31), notes="Jan")
    snapshot2 = Snapshot(date=date(2024, 2, 29), notes="Feb")
    test_db_session.add_all([snapshot1, snapshot2])
    test_db_session.commit()

    # Add values
    test_db_session.add_all(
        [
            SnapshotValue(snapshot_id=snapshot1.id, account_id=asset.id, value=Decimal("10000")),
            SnapshotValue(
                snapshot_id=snapshot1.id, account_id=liability.id, value=Decimal("2000")
            ),
            SnapshotValue(snapshot_id=snapshot2.id, account_id=asset.id, value=Decimal("11000")),
            SnapshotValue(
                snapshot_id=snapshot2.id, account_id=liability.id, value=Decimal("1800")
            ),
        ]
    )
    test_db_session.commit()

    # Get all snapshots
    result = get_all_snapshots(test_db_session)

    assert len(result) == 2
    # Should be ordered by date desc
    assert result[0].date == date(2024, 2, 29)
    assert result[0].total_net_worth == 9200.0  # 11000 - 1800
    assert result[1].date == date(2024, 1, 31)
    assert result[1].total_net_worth == 8000.0  # 10000 - 2000


def test_get_snapshot_by_id(test_db_session):
    """Test getting single snapshot by ID"""
    # Create account and snapshot
    account = Account(
        name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("5000.50")
    )
    test_db_session.add(value)
    test_db_session.commit()

    # Get snapshot
    result = get_snapshot_by_id(test_db_session, snapshot.id)

    assert result.id == snapshot.id
    assert result.date == date(2024, 1, 31)
    assert result.notes == "Test"
    assert len(result.values) == 1
    assert result.values[0].account_name == "Test Bank"
    assert result.values[0].value == 5000.50


def test_get_snapshot_by_id_not_found(test_db_session):
    """Test getting non-existent snapshot fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_snapshot_by_id(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_get_latest_snapshot_values(test_db_session):
    """Test getting latest snapshot values for pre-fill"""
    # Create accounts
    account1 = Account(
        name="Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    account2 = Account(
        name="IKE", type="asset", category="ike", owner="Test", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create snapshots
    old_snapshot = Snapshot(date=date(2024, 1, 31))
    latest_snapshot = Snapshot(date=date(2024, 2, 29))
    test_db_session.add_all([old_snapshot, latest_snapshot])
    test_db_session.commit()

    # Add values (latest should be returned)
    test_db_session.add_all(
        [
            SnapshotValue(
                snapshot_id=old_snapshot.id, account_id=account1.id, value=Decimal("1000")
            ),
            SnapshotValue(
                snapshot_id=latest_snapshot.id, account_id=account1.id, value=Decimal("2000")
            ),
            SnapshotValue(
                snapshot_id=latest_snapshot.id, account_id=account2.id, value=Decimal("3000")
            ),
        ]
    )
    test_db_session.commit()

    # Get latest values
    result = get_latest_snapshot_values(test_db_session)

    assert len(result) == 2
    assert result[account1.id] == 2000.0
    assert result[account2.id] == 3000.0


def test_get_latest_snapshot_values_empty(test_db_session):
    """Test getting latest values when no snapshots exist"""
    result = get_latest_snapshot_values(test_db_session)
    assert result == {}
