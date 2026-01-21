from decimal import Decimal
from unittest.mock import patch

import pytest
from fastapi import HTTPException
from sqlalchemy.exc import IntegrityError

from app.models import Account
from app.schemas.accounts import AccountCreate, AccountUpdate
from app.services.accounts import create_account, delete_account, get_all_accounts, update_account
from tests.factories import create_test_account, create_test_snapshot, create_test_snapshot_value


def test_get_all_accounts_with_values(test_db_session):
    """Test getting all accounts with their latest snapshot values"""
    # Create accounts
    asset1 = create_test_account(test_db_session, name="Bank Account")
    asset2 = create_test_account(
        test_db_session,
        name="IKE",
        category="fund",
        account_wrapper="IKE",
        purpose="retirement",
    )
    liability = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Shared",
    )
    inactive = create_test_account(test_db_session, name="Old Account", is_active=False)

    # Create snapshot with values
    snapshot = create_test_snapshot(test_db_session)
    create_test_snapshot_value(test_db_session, snapshot.id, asset1.id, Decimal("50000"))
    create_test_snapshot_value(test_db_session, snapshot.id, asset2.id, Decimal("20000"))
    create_test_snapshot_value(test_db_session, snapshot.id, liability.id, Decimal("100000"))
    create_test_snapshot_value(test_db_session, snapshot.id, inactive.id, Decimal("1000"))

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
    create_test_account(test_db_session, name="Bank Account")
    create_test_account(
        test_db_session,
        name="Loan",
        account_type="liability",
        category="mortgage",
    )

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
    account1 = create_test_account(test_db_session, name="Account 1")
    create_test_account(test_db_session, name="Account 2")

    # Create snapshot with only one account value
    snapshot = create_test_snapshot(test_db_session)
    create_test_snapshot_value(test_db_session, snapshot.id, account1.id, Decimal("1000"))

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


def test_create_account_asset(test_db_session):
    """Test creating a new asset account"""
    data = AccountCreate(
        name="New Savings Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )

    result = create_account(test_db_session, data)

    assert result.id is not None
    assert result.name == "New Savings Account"
    assert result.type == "asset"
    assert result.category == "bank"
    assert result.owner == "Marcin"
    assert result.currency == "PLN"
    assert result.is_active is True
    assert result.current_value == 0.0

    # Verify account was saved to database
    saved_account = test_db_session.query(Account).filter_by(id=result.id).first()
    assert saved_account is not None
    assert saved_account.name == "New Savings Account"


def test_create_account_liability(test_db_session):
    """Test creating a new liability account"""
    data = AccountCreate(
        name="Car Loan",
        type="liability",
        category="installment",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )

    result = create_account(test_db_session, data)

    assert result.id is not None
    assert result.name == "Car Loan"
    assert result.type == "liability"
    assert result.category == "installment"
    assert result.owner == "Ewa"
    assert result.currency == "PLN"
    assert result.is_active is True
    assert result.current_value == 0.0


def test_create_account_investment(test_db_session):
    """Test creating an investment account"""
    data = AccountCreate(
        name="IKE - Test",
        type="asset",
        category="fund",
        owner="Shared",
        currency="PLN",
        account_wrapper="IKE",
        purpose="retirement",
    )

    result = create_account(test_db_session, data)

    assert result.category == "fund"
    assert result.account_wrapper == "IKE"
    assert result.owner == "Shared"

    # Verify in database
    saved_account = test_db_session.query(Account).filter_by(id=result.id).first()
    assert saved_account.category == "fund"
    assert saved_account.account_wrapper == "IKE"


def test_create_account_with_default_currency(test_db_session):
    """Test creating account uses PLN as default currency"""
    data = AccountCreate(
        name="Test Account", type="asset", category="other", owner="Marcin", purpose="general"
    )

    result = create_account(test_db_session, data)

    assert result.currency == "PLN"


def test_create_account_duplicate_name(test_db_session):
    """Test creating account with duplicate name fails"""
    import pytest
    from fastapi import HTTPException

    # Create first account
    data1 = AccountCreate(
        name="Duplicate Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    create_account(test_db_session, data1)

    # Try to create account with same name
    data2 = AccountCreate(
        name="Duplicate Test Account",
        type="asset",
        category="fund",
        owner="Ewa",
        currency="PLN",
        account_wrapper="IKE",
        purpose="retirement",
    )

    with pytest.raises(HTTPException) as exc_info:
        create_account(test_db_session, data2)

    assert exc_info.value.status_code == 400
    assert "already exists" in exc_info.value.detail


def test_update_account_success(test_db_session):
    """Test updating an account successfully"""
    # Create account
    account = create_test_account(test_db_session, name="Original Name")
    account_id = account.id

    # Update account
    data = AccountUpdate(
        name="Updated Name", category="fund", owner="Ewa", currency="EUR", account_wrapper="IKE"
    )
    result = update_account(test_db_session, account_id, data)

    assert result.id == account_id
    assert result.name == "Updated Name"
    assert result.type == "asset"  # type unchanged
    assert result.category == "fund"
    assert result.account_wrapper == "IKE"
    assert result.owner == "Ewa"
    assert result.currency == "EUR"
    assert result.is_active is True


def test_update_account_duplicate_name(test_db_session):
    """Test updating account with duplicate name fails"""
    # Create two accounts
    create_test_account(test_db_session, name="Account 1")
    account2 = create_test_account(test_db_session, name="Account 2")

    # Try to update account2 with account1's name
    data = AccountUpdate(name="Account 1")

    with pytest.raises(HTTPException) as exc_info:
        update_account(test_db_session, account2.id, data)

    assert exc_info.value.status_code == 400
    assert "already exists" in exc_info.value.detail


def test_update_account_not_found(test_db_session):
    """Test updating non-existent account fails"""
    data = AccountUpdate(name="New Name")

    with pytest.raises(HTTPException) as exc_info:
        update_account(test_db_session, 999, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_update_account_partial(test_db_session):
    """Test partial update of account"""
    # Create account
    account = create_test_account(test_db_session, name="Original")
    account_id = account.id

    # Update only name
    data = AccountUpdate(name="Updated Only Name")
    result = update_account(test_db_session, account_id, data)

    assert result.name == "Updated Only Name"
    assert result.type == "asset"
    assert result.category == "bank"
    assert result.owner == "Marcin"
    assert result.currency == "PLN"


def test_update_account_purpose(test_db_session):
    """Test updating account purpose"""
    # Create account
    account = create_test_account(test_db_session, name="Test Account")
    account_id = account.id

    # Update purpose
    data = AccountUpdate(purpose="retirement")
    result = update_account(test_db_session, account_id, data)

    assert result.purpose == "retirement"
    assert result.name == "Test Account"

    # Verify in database
    saved_account = test_db_session.query(Account).filter_by(id=account_id).first()
    assert saved_account.purpose == "retirement"


def test_delete_account_success(test_db_session):
    """Test soft deleting an account"""
    # Create account
    account = create_test_account(test_db_session, name="To Delete")
    account_id = account.id

    # Delete account
    delete_account(test_db_session, account_id)

    # Verify soft delete
    deleted = test_db_session.query(Account).filter_by(id=account_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_account_not_found(test_db_session):
    """Test deleting non-existent account fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_account(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_delete_account_idempotent(test_db_session):
    """Test deleting already deleted account is idempotent"""
    # Create account
    account = create_test_account(test_db_session, name="Already Deleted")
    account_id = account.id

    # Delete once
    delete_account(test_db_session, account_id)

    # Delete again - should not raise error
    delete_account(test_db_session, account_id)

    # Verify still deleted
    deleted = test_db_session.query(Account).filter_by(id=account_id).first()
    assert deleted.is_active is False


def test_create_account_integrity_error(test_db_session):
    """Test creating account with IntegrityError returns 500"""
    data = AccountCreate(
        name="Test Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )

    # Mock db.commit to raise IntegrityError
    with patch.object(test_db_session, "commit", side_effect=IntegrityError("", "", Exception())):
        with pytest.raises(HTTPException) as exc_info:
            create_account(test_db_session, data)

        assert exc_info.value.status_code == 500
        assert "integrity error" in exc_info.value.detail.lower()


def test_update_account_integrity_error(test_db_session):
    """Test updating account with IntegrityError returns 500"""
    account = create_test_account(test_db_session, name="Test Account")

    data = AccountUpdate(name="Updated Name")

    # Mock db.commit to raise IntegrityError
    with patch.object(test_db_session, "commit", side_effect=IntegrityError("", "", Exception())):
        with pytest.raises(HTTPException) as exc_info:
            update_account(test_db_session, account.id, data)

        assert exc_info.value.status_code == 500
        assert "integrity error" in exc_info.value.detail.lower()
