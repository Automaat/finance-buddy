"""Tests for utility functions (db_helpers, validators)."""

from datetime import UTC, date, datetime, timedelta

import pytest
from fastapi import HTTPException

from app.core.enums import Owner
from app.models import Account, Snapshot
from app.models.snapshot import SnapshotValue
from app.utils.db_helpers import (
    check_duplicate_name,
    get_latest_snapshot_value,
    get_latest_snapshot_values_batch,
    get_or_404,
    soft_delete,
)
from app.utils.validators import (
    validate_non_negative_amount,
    validate_not_empty_string,
    validate_not_future_date,
    validate_owner_marcin_or_ewa,
    validate_positive_amount,
)

# ==========================
# db_helpers.py tests
# ==========================


def test_get_or_404_success(test_db_session):
    """Test successful entity retrieval"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    result = get_or_404(test_db_session, Account, account.id)

    assert result.id == account.id
    assert result.name == "Test Account"


def test_get_or_404_not_found(test_db_session):
    """Test 404 error for non-existent entity"""
    with pytest.raises(HTTPException) as exc_info:
        get_or_404(test_db_session, Account, 999)

    assert exc_info.value.status_code == 404
    assert "Account with id 999 not found" in exc_info.value.detail


def test_get_latest_snapshot_value_success(test_db_session):
    """Test fetching latest snapshot value"""
    # Create account
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create snapshots
    snapshot1 = Snapshot(date=date(2024, 1, 31), notes="Jan")
    snapshot2 = Snapshot(date=date(2024, 2, 29), notes="Feb")
    test_db_session.add_all([snapshot1, snapshot2])
    test_db_session.commit()

    # Create snapshot values
    value1 = SnapshotValue(snapshot_id=snapshot1.id, account_id=account.id, value=1000.0)
    value2 = SnapshotValue(snapshot_id=snapshot2.id, account_id=account.id, value=1500.0)
    test_db_session.add_all([value1, value2])
    test_db_session.commit()

    result = get_latest_snapshot_value(test_db_session, account.id)

    assert result == 1500.0


def test_get_latest_snapshot_value_no_snapshots(test_db_session):
    """Test returns 0.0 when no snapshots exist"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    result = get_latest_snapshot_value(test_db_session, account.id)

    assert result == 0.0


def test_get_latest_snapshot_values_batch_success(test_db_session):
    """Test batch fetching latest snapshot values"""
    # Create accounts
    account1 = Account(
        name="Account 1",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Account 2",
        type="asset",
        category="bank",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create snapshot values
    value1 = SnapshotValue(snapshot_id=snapshot.id, account_id=account1.id, value=1000.0)
    value2 = SnapshotValue(snapshot_id=snapshot.id, account_id=account2.id, value=2000.0)
    test_db_session.add_all([value1, value2])
    test_db_session.commit()

    result = get_latest_snapshot_values_batch(test_db_session, [account1.id, account2.id])

    assert result == {account1.id: 1000.0, account2.id: 2000.0}


def test_get_latest_snapshot_values_batch_missing_accounts(test_db_session):
    """Test batch fetch fills missing accounts with 0.0"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    result = get_latest_snapshot_values_batch(test_db_session, [account.id, 999])

    assert result == {account.id: 0.0, 999: 0.0}


def test_get_latest_snapshot_values_batch_empty_list(test_db_session):
    """Test batch fetch with empty list returns empty dict"""
    result = get_latest_snapshot_values_batch(test_db_session, [])
    assert result == {}


def test_get_latest_snapshot_values_batch_for_assets_success(test_db_session):
    """Test batch fetching latest snapshot values for assets"""
    from app.models import Asset
    from app.utils.db_helpers import get_latest_snapshot_values_batch_for_assets

    # Create assets
    asset1 = Asset(name="Asset 1", is_active=True)
    asset2 = Asset(name="Asset 2", is_active=True)
    test_db_session.add_all([asset1, asset2])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create snapshot values
    value1 = SnapshotValue(snapshot_id=snapshot.id, asset_id=asset1.id, value=5000.0)
    value2 = SnapshotValue(snapshot_id=snapshot.id, asset_id=asset2.id, value=10000.0)
    test_db_session.add_all([value1, value2])
    test_db_session.commit()

    result = get_latest_snapshot_values_batch_for_assets(test_db_session, [asset1.id, asset2.id])

    assert result == {asset1.id: 5000.0, asset2.id: 10000.0}


def test_get_latest_snapshot_values_batch_for_assets_missing_assets(test_db_session):
    """Test batch fetch for assets fills missing with 0.0"""
    from app.models import Asset
    from app.utils.db_helpers import get_latest_snapshot_values_batch_for_assets

    asset = Asset(name="Test Asset", is_active=True)
    test_db_session.add(asset)
    test_db_session.commit()

    # No snapshots created
    result = get_latest_snapshot_values_batch_for_assets(test_db_session, [asset.id, 999])

    assert result == {asset.id: 0.0, 999: 0.0}


def test_get_latest_snapshot_values_batch_for_assets_empty_list(test_db_session):
    """Test batch fetch for assets with empty list returns empty dict"""
    from app.utils.db_helpers import get_latest_snapshot_values_batch_for_assets

    result = get_latest_snapshot_values_batch_for_assets(test_db_session, [])
    assert result == {}


def test_get_latest_snapshot_values_batch_for_assets_out_of_order(test_db_session):
    """Test batch fetch with out-of-order snapshots returns latest by date"""
    from datetime import date

    from app.models import Asset, Snapshot
    from app.models.snapshot import SnapshotValue
    from app.utils.db_helpers import get_latest_snapshot_values_batch_for_assets

    # Create asset
    asset = Asset(name="Test Asset", is_active=True)
    test_db_session.add(asset)
    test_db_session.commit()

    # Create snapshots OUT OF ORDER
    # 1. Create Feb snapshot (id=1, date=2024-02-28)
    snapshot_feb = Snapshot(date=date(2024, 2, 28), notes="February")
    test_db_session.add(snapshot_feb)
    test_db_session.commit()

    value_feb = SnapshotValue(snapshot_id=snapshot_feb.id, asset_id=asset.id, value=2000.0)
    test_db_session.add(value_feb)
    test_db_session.commit()

    # 2. Backfill Jan snapshot (id=2, date=2024-01-31)
    snapshot_jan = Snapshot(date=date(2024, 1, 31), notes="January")
    test_db_session.add(snapshot_jan)
    test_db_session.commit()

    value_jan = SnapshotValue(snapshot_id=snapshot_jan.id, asset_id=asset.id, value=1000.0)
    test_db_session.add(value_jan)
    test_db_session.commit()

    # Verify IDs: snapshot_feb.id < snapshot_jan.id (out of date order)
    assert snapshot_feb.id < snapshot_jan.id

    # Should return Feb value (2000.0) not Jan value (1000.0)
    result = get_latest_snapshot_values_batch_for_assets(test_db_session, [asset.id])
    assert result == {asset.id: 2000.0}


def test_soft_delete_success(test_db_session):
    """Test soft deleting entity sets is_active=False"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    test_db_session.add(account)
    test_db_session.commit()

    soft_delete(test_db_session, Account, account.id)

    test_db_session.refresh(account)
    assert account.is_active is False


def test_soft_delete_idempotent(test_db_session):
    """Test soft delete is idempotent (no error when already inactive)"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=False,
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Should not raise error
    soft_delete(test_db_session, Account, account.id)

    test_db_session.refresh(account)
    assert account.is_active is False


def test_soft_delete_not_found(test_db_session):
    """Test soft delete raises 404 for non-existent entity"""
    with pytest.raises(HTTPException) as exc_info:
        soft_delete(test_db_session, Account, 999)

    assert exc_info.value.status_code == 404


def test_check_duplicate_name_no_duplicate(test_db_session):
    """Test no error when name is unique"""
    account = Account(
        name="Existing Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Should not raise error
    check_duplicate_name(test_db_session, Account, "New Account")


def test_check_duplicate_name_raises_400(test_db_session):
    """Test raises 400 when duplicate name exists"""
    account = Account(
        name="Duplicate Name",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    test_db_session.add(account)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        check_duplicate_name(test_db_session, Account, "Duplicate Name")

    assert exc_info.value.status_code == 400
    assert "Active account 'Duplicate Name' already exists" in exc_info.value.detail


def test_check_duplicate_name_exclude_id(test_db_session):
    """Test no error when duplicate name is same entity (update case)"""
    account = Account(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Should not raise error when excluding same ID
    check_duplicate_name(test_db_session, Account, "Test Account", exclude_id=account.id)


# ==========================
# validators.py tests
# ==========================


def test_validate_positive_amount_success():
    """Test positive amount passes validation"""
    assert validate_positive_amount(100.0) == 100.0
    assert validate_positive_amount(0.01) == 0.01


def test_validate_positive_amount_zero_fails():
    """Test zero amount fails validation"""
    with pytest.raises(ValueError, match="Amount must be greater than 0"):
        validate_positive_amount(0.0)


def test_validate_positive_amount_negative_fails():
    """Test negative amount fails validation"""
    with pytest.raises(ValueError, match="Amount must be greater than 0"):
        validate_positive_amount(-50.0)


def test_validate_non_negative_amount_success():
    """Test non-negative amount passes validation"""
    assert validate_non_negative_amount(100.0) == 100.0
    assert validate_non_negative_amount(0.0) == 0.0


def test_validate_non_negative_amount_negative_fails():
    """Test negative amount fails validation"""
    with pytest.raises(ValueError, match="Amount must be non-negative"):
        validate_non_negative_amount(-1.0)


def test_validate_not_future_date_success():
    """Test past and today dates pass validation"""
    today = datetime.now(UTC).date()
    yesterday = today - timedelta(days=1)

    assert validate_not_future_date(today) == today
    assert validate_not_future_date(yesterday) == yesterday


def test_validate_not_future_date_fails():
    """Test future date fails validation"""
    tomorrow = datetime.now(UTC).date() + timedelta(days=1)

    with pytest.raises(ValueError, match="Date cannot be in the future"):
        validate_not_future_date(tomorrow)


def test_validate_not_empty_string_success():
    """Test non-empty string passes validation"""
    assert validate_not_empty_string("Test Name") == "Test Name"
    assert validate_not_empty_string("  Trimmed  ") == "Trimmed"
    assert validate_not_empty_string("Company", "Company") == "Company"


def test_validate_not_empty_string_empty_fails():
    """Test empty string fails validation"""
    with pytest.raises(ValueError, match="Name cannot be empty"):
        validate_not_empty_string("")

    with pytest.raises(ValueError, match="Name cannot be empty"):
        validate_not_empty_string("   ")

    with pytest.raises(ValueError, match="Company cannot be empty"):
        validate_not_empty_string("", "Company")


def test_validate_not_empty_string_nullable():
    """Test None is allowed (for nullable fields)"""
    assert validate_not_empty_string(None) is None


def test_validate_owner_marcin_or_ewa_success():
    """Test MARCIN and EWA pass validation"""
    assert validate_owner_marcin_or_ewa(Owner.MARCIN) == Owner.MARCIN
    assert validate_owner_marcin_or_ewa(Owner.EWA) == Owner.EWA


def test_validate_owner_marcin_or_ewa_shared_fails():
    """Test SHARED fails validation"""
    with pytest.raises(
        ValueError, match="Owner must be MARCIN or EWA for individual retirement accounts"
    ):
        validate_owner_marcin_or_ewa(Owner.SHARED)
