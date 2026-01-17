from datetime import date
from decimal import Decimal

import pytest
from fastapi import HTTPException

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.snapshots import SnapshotCreate, SnapshotValueInput
from app.services.snapshots import (
    create_snapshot,
    get_all_snapshots,
    get_snapshot_by_id,
)


def test_create_snapshot_success(test_db_session):
    """Test creating a snapshot with account values"""
    # Create test accounts
    account1 = Account(
        name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN"
    )
    account2 = Account(name="Test IKE", type="asset", category="ike", owner="Test", currency="PLN")
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
    account = Account(name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN")
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
    asset = Account(name="Bank", type="asset", category="bank", owner="Test", currency="PLN")
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
            SnapshotValue(snapshot_id=snapshot1.id, account_id=liability.id, value=Decimal("2000")),
            SnapshotValue(snapshot_id=snapshot2.id, account_id=asset.id, value=Decimal("11000")),
            SnapshotValue(snapshot_id=snapshot2.id, account_id=liability.id, value=Decimal("1800")),
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
    account = Account(name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN")
    test_db_session.add(account)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("5000.50"))
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


def test_create_snapshot_duplicate_account_ids(test_db_session):
    """Test creating snapshot with duplicate account IDs fails"""
    # Create account
    account = Account(name="Test Bank", type="asset", category="bank", owner="Test", currency="PLN")
    test_db_session.add(account)
    test_db_session.commit()

    # Try to create snapshot with duplicate account IDs
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[
            SnapshotValueInput(account_id=account.id, value=1000),
            SnapshotValueInput(account_id=account.id, value=2000),
        ],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 400
    assert "Duplicate account IDs" in exc_info.value.detail


def test_create_snapshot_empty_values():
    """Test creating snapshot with empty values list fails"""
    from pydantic import ValidationError

    with pytest.raises(ValidationError) as exc_info:
        SnapshotCreate(date=date(2024, 1, 31), notes="Test", values=[])

    assert "at least one account value" in str(exc_info.value)


def test_create_snapshot_inactive_account(test_db_session):
    """Test creating snapshot with inactive account fails"""
    # Create active and inactive accounts
    active_account = Account(
        name="Active Bank",
        type="asset",
        category="bank",
        owner="Test",
        currency="PLN",
        is_active=True,
    )
    inactive_account = Account(
        name="Inactive Bank",
        type="asset",
        category="bank",
        owner="Test",
        currency="PLN",
        is_active=False,
    )
    test_db_session.add_all([active_account, inactive_account])
    test_db_session.commit()

    # Try to create snapshot with inactive account
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[
            SnapshotValueInput(account_id=active_account.id, value=1000),
            SnapshotValueInput(account_id=inactive_account.id, value=2000),
        ],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_create_snapshot_with_assets(test_db_session):
    """Test creating snapshot with asset values"""
    from app.models import Asset

    # Create test assets
    asset1 = Asset(name="Car", is_active=True)
    asset2 = Asset(name="Electronics", is_active=True)
    test_db_session.add_all([asset1, asset2])
    test_db_session.commit()

    # Create snapshot with assets
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test snapshot with assets",
        values=[
            SnapshotValueInput(asset_id=asset1.id, value=50000.0),
            SnapshotValueInput(asset_id=asset2.id, value=3000.0),
        ],
    )

    result = create_snapshot(test_db_session, data)

    assert result.date == date(2024, 1, 31)
    assert len(result.values) == 2
    assert result.values[0].asset_name == "Car"
    assert result.values[0].value == 50000.0
    assert result.values[1].asset_name == "Electronics"
    assert result.values[1].value == 3000.0


def test_create_snapshot_invalid_asset(test_db_session):
    """Test creating snapshot with invalid asset ID fails"""
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[SnapshotValueInput(asset_id=999, value=1000)],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 404
    assert "Assets not found" in exc_info.value.detail


def test_create_snapshot_duplicate_asset_ids(test_db_session):
    """Test creating snapshot with duplicate asset IDs fails"""
    from app.models import Asset

    # Create asset
    asset = Asset(name="Car", is_active=True)
    test_db_session.add(asset)
    test_db_session.commit()

    # Try to create snapshot with duplicate asset IDs
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[
            SnapshotValueInput(asset_id=asset.id, value=1000),
            SnapshotValueInput(asset_id=asset.id, value=2000),
        ],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 400
    assert "Duplicate asset IDs" in exc_info.value.detail


def test_create_snapshot_inactive_asset(test_db_session):
    """Test creating snapshot with inactive asset fails"""
    from app.models import Asset

    # Create active and inactive assets
    active_asset = Asset(name="Active Car", is_active=True)
    inactive_asset = Asset(name="Inactive Car", is_active=False)
    test_db_session.add_all([active_asset, inactive_asset])
    test_db_session.commit()

    # Try to create snapshot with inactive asset
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[
            SnapshotValueInput(asset_id=active_asset.id, value=1000),
            SnapshotValueInput(asset_id=inactive_asset.id, value=2000),
        ],
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 404
    assert "Assets not found" in exc_info.value.detail


def test_create_snapshot_mixed_assets_and_accounts(test_db_session):
    """Test creating snapshot with both assets and accounts"""
    from app.models import Asset

    # Create account and asset
    account = Account(name="Bank", type="asset", category="bank", owner="Test", currency="PLN")
    asset = Asset(name="Car", is_active=True)
    test_db_session.add_all([account, asset])
    test_db_session.commit()

    # Create snapshot with both
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Mixed snapshot",
        values=[
            SnapshotValueInput(account_id=account.id, value=10000.0),
            SnapshotValueInput(asset_id=asset.id, value=50000.0),
        ],
    )

    result = create_snapshot(test_db_session, data)

    assert len(result.values) == 2
    # Find account and asset values
    account_value = next(v for v in result.values if v.account_id is not None)
    asset_value = next(v for v in result.values if v.asset_id is not None)

    assert account_value.account_name == "Bank"
    assert account_value.value == 10000.0
    assert asset_value.asset_name == "Car"
    assert asset_value.value == 50000.0


def test_get_all_snapshots_with_assets(test_db_session):
    """Test getting snapshots with assets calculates net worth correctly"""
    from app.models import Asset

    # Create assets
    asset = Asset(name="Car", is_active=True)
    account = Account(name="Bank", type="asset", category="bank", owner="Test", currency="PLN")
    test_db_session.add_all([asset, account])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31), notes="Mixed")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add values
    test_db_session.add_all(
        [
            SnapshotValue(snapshot_id=snapshot.id, asset_id=asset.id, value=Decimal("50000")),
            SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("10000")),
        ]
    )
    test_db_session.commit()

    # Get snapshots
    result = get_all_snapshots(test_db_session)

    assert len(result) == 1
    # Assets contribute positively: 50000 + 10000
    assert result[0].total_net_worth == 60000.0


def test_get_snapshot_by_id_with_assets(test_db_session):
    """Test getting single snapshot with assets"""
    from app.models import Asset

    # Create asset and account
    asset = Asset(name="Car", is_active=True)
    account = Account(name="Bank", type="asset", category="bank", owner="Test", currency="PLN")
    test_db_session.add_all([asset, account])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31), notes="Mixed")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Add values
    asset_value = SnapshotValue(
        snapshot_id=snapshot.id, asset_id=asset.id, value=Decimal("50000")
    )
    account_value = SnapshotValue(
        snapshot_id=snapshot.id, account_id=account.id, value=Decimal("10000")
    )
    test_db_session.add_all([asset_value, account_value])
    test_db_session.commit()

    # Get snapshot
    result = get_snapshot_by_id(test_db_session, snapshot.id)

    assert len(result.values) == 2
    # Find values
    asset_resp = next(v for v in result.values if v.asset_id is not None)
    account_resp = next(v for v in result.values if v.account_id is not None)

    assert asset_resp.asset_name == "Car"
    assert asset_resp.value == 50000.0
    assert account_resp.account_name == "Bank"
    assert account_resp.value == 10000.0
