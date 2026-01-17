import pytest
from pydantic import ValidationError

from app.schemas.snapshots import SnapshotValueInput


def test_snapshot_value_asset_id_only():
    """Test SnapshotValueInput with only asset_id is valid"""
    value = SnapshotValueInput(asset_id=1, value=1000.0)
    assert value.asset_id == 1
    assert value.account_id is None
    assert value.value == 1000.0


def test_snapshot_value_account_id_only():
    """Test SnapshotValueInput with only account_id is valid"""
    value = SnapshotValueInput(account_id=1, value=1000.0)
    assert value.account_id == 1
    assert value.asset_id is None
    assert value.value == 1000.0


def test_snapshot_value_both_ids():
    """Test SnapshotValueInput rejects both asset_id and account_id"""
    with pytest.raises(ValidationError) as exc_info:
        SnapshotValueInput(asset_id=1, account_id=2, value=1000.0)

    assert "Only one of asset_id or account_id can be provided" in str(exc_info.value)


def test_snapshot_value_neither_id():
    """Test SnapshotValueInput rejects neither asset_id nor account_id"""
    with pytest.raises(ValidationError) as exc_info:
        SnapshotValueInput(value=1000.0)

    assert "Either asset_id or account_id must be provided" in str(exc_info.value)
