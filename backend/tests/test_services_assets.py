from decimal import Decimal
from unittest.mock import patch

import pytest
from fastapi import HTTPException
from sqlalchemy.exc import IntegrityError

from app.models import Asset
from app.schemas.assets import AssetCreate, AssetUpdate
from app.services.assets import create_asset, delete_asset, get_all_assets, update_asset
from tests.factories import create_test_asset, create_test_snapshot, create_test_snapshot_value


def test_get_all_assets_with_values(test_db_session):
    """Test getting all assets with their latest snapshot values"""
    # Create assets
    asset1 = create_test_asset(test_db_session, name="Car")
    asset2 = create_test_asset(test_db_session, name="Electronics")
    inactive = create_test_asset(test_db_session, name="Old Phone", is_active=False)

    # Create snapshot with values
    snapshot = create_test_snapshot(test_db_session)
    create_test_snapshot_value(
        test_db_session, snapshot.id, asset_id=asset1.id, value=Decimal("50000")
    )
    create_test_snapshot_value(
        test_db_session, snapshot.id, asset_id=asset2.id, value=Decimal("3000")
    )
    create_test_snapshot_value(
        test_db_session, snapshot.id, asset_id=inactive.id, value=Decimal("100")
    )

    # Get all assets
    result = get_all_assets(test_db_session)

    # Should have 2 active assets (inactive excluded)
    assert len(result.assets) == 2

    # Check asset values
    car = next(a for a in result.assets if a.name == "Car")
    assert car.current_value == 50000.0
    assert car.is_active is True

    electronics = next(a for a in result.assets if a.name == "Electronics")
    assert electronics.current_value == 3000.0


def test_get_all_assets_no_snapshots(test_db_session):
    """Test getting assets when no snapshots exist"""
    # Create assets without snapshots
    create_test_asset(test_db_session, name="Car")

    # Get all assets
    result = get_all_assets(test_db_session)

    # Should have 1 asset with 0 value
    assert len(result.assets) == 1
    assert result.assets[0].name == "Car"
    assert result.assets[0].current_value == 0.0


def test_get_all_assets_empty(test_db_session):
    """Test getting assets when none exist"""
    result = get_all_assets(test_db_session)

    assert len(result.assets) == 0


def test_create_asset_success(test_db_session):
    """Test creating a new asset"""
    data = AssetCreate(name="Car")

    result = create_asset(test_db_session, data)

    assert result.name == "Car"
    assert result.is_active is True
    assert result.current_value == 0.0
    assert result.id > 0

    # Verify in database
    asset = test_db_session.get(Asset, result.id)
    assert asset is not None
    assert asset.name == "Car"


def test_create_asset_duplicate_name(test_db_session):
    """Test creating asset with duplicate name fails"""
    # Create first asset
    create_test_asset(test_db_session, name="Car")

    # Try to create duplicate
    data = AssetCreate(name="Car")

    with pytest.raises(HTTPException) as exc_info:
        create_asset(test_db_session, data)

    assert exc_info.value.status_code == 400
    assert "already exists" in exc_info.value.detail


def test_create_asset_allows_duplicate_inactive(test_db_session):
    """Test creating asset with same name as inactive asset is allowed"""
    # Create inactive asset
    create_test_asset(test_db_session, name="Car", is_active=False)

    # Create new active asset with same name (allowed)
    data = AssetCreate(name="Car")
    result = create_asset(test_db_session, data)

    assert result.name == "Car"
    assert result.is_active is True


def test_update_asset_success(test_db_session):
    """Test updating an asset"""
    asset = create_test_asset(test_db_session, name="Car")

    data = AssetUpdate(name="Tesla")

    result = update_asset(test_db_session, asset.id, data)

    assert result.name == "Tesla"
    assert result.id == asset.id

    # Verify in database
    updated = test_db_session.get(Asset, asset.id)
    assert updated.name == "Tesla"


def test_update_asset_not_found(test_db_session):
    """Test updating non-existent asset fails"""
    data = AssetUpdate(name="Car")

    with pytest.raises(HTTPException) as exc_info:
        update_asset(test_db_session, 999, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_update_asset_duplicate_name(test_db_session):
    """Test updating asset to duplicate name fails"""
    create_test_asset(test_db_session, name="Car")
    asset2 = create_test_asset(test_db_session, name="Electronics")

    # Try to rename asset2 to asset1's name
    data = AssetUpdate(name="Car")

    with pytest.raises(HTTPException) as exc_info:
        update_asset(test_db_session, asset2.id, data)

    assert exc_info.value.status_code == 400
    assert "already exists" in exc_info.value.detail


def test_update_asset_same_name(test_db_session):
    """Test updating asset keeping the same name works"""
    asset = create_test_asset(test_db_session, name="Car")

    # Update with same name
    data = AssetUpdate(name="Car")
    result = update_asset(test_db_session, asset.id, data)

    assert result.name == "Car"


def test_update_asset_with_snapshot_value(test_db_session):
    """Test updating asset returns current value from latest snapshot"""
    asset = create_test_asset(test_db_session, name="Car")

    # Create snapshot with value
    snapshot = create_test_snapshot(test_db_session)
    create_test_snapshot_value(
        test_db_session, snapshot.id, asset_id=asset.id, value=Decimal("50000")
    )

    # Update asset
    data = AssetUpdate(name="Tesla")
    result = update_asset(test_db_session, asset.id, data)

    assert result.name == "Tesla"
    assert result.current_value == 50000.0


def test_update_asset_no_snapshot(test_db_session):
    """Test updating asset without snapshot returns 0 value"""
    asset = create_test_asset(test_db_session, name="Car")

    data = AssetUpdate(name="Tesla")
    result = update_asset(test_db_session, asset.id, data)

    assert result.current_value == 0.0


def test_delete_asset_success(test_db_session):
    """Test soft deleting an asset"""
    asset = create_test_asset(test_db_session, name="Car")

    delete_asset(test_db_session, asset.id)

    # Verify soft delete
    deleted = test_db_session.get(Asset, asset.id)
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_asset_not_found(test_db_session):
    """Test deleting non-existent asset fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_asset(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_delete_asset_idempotent(test_db_session):
    """Test deleting already deleted asset is idempotent"""
    asset = create_test_asset(test_db_session, name="Car", is_active=False)

    # Delete already inactive asset (should succeed without error)
    delete_asset(test_db_session, asset.id)

    # Verify still inactive
    deleted = test_db_session.get(Asset, asset.id)
    assert deleted.is_active is False


def test_create_asset_integrity_error(test_db_session):
    """Test creating asset with IntegrityError returns 500"""
    data = AssetCreate(name="Test Asset")

    # Mock db.commit to raise IntegrityError
    with patch.object(test_db_session, "commit", side_effect=IntegrityError("", "", Exception())):
        with pytest.raises(HTTPException) as exc_info:
            create_asset(test_db_session, data)

        assert exc_info.value.status_code == 500
        assert "integrity error" in exc_info.value.detail.lower()


def test_update_asset_integrity_error(test_db_session):
    """Test updating asset with IntegrityError returns 500"""
    asset = create_test_asset(test_db_session, name="Test Asset")

    data = AssetUpdate(name="Updated Name")

    # Mock db.commit to raise IntegrityError
    with patch.object(test_db_session, "commit", side_effect=IntegrityError("", "", Exception())):
        with pytest.raises(HTTPException) as exc_info:
            update_asset(test_db_session, asset.id, data)

        assert exc_info.value.status_code == 500
        assert "integrity error" in exc_info.value.detail.lower()
