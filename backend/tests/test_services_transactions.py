from datetime import date

import pytest
from fastapi import HTTPException

from app.models import Transaction
from app.schemas.transactions import TransactionCreate
from app.services.transactions import (
    create_transaction,
    delete_transaction,
    get_account_transactions,
    get_all_transactions,
    get_transaction_counts,
)
from tests.factories import create_test_account, create_test_transaction


def test_get_account_transactions_success(test_db_session):
    """Test getting transactions for a specific investment account"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=2000.0,
        transaction_date=date(2024, 3, 5),
        owner="Marcin",
        is_active=False,
    )

    # Get transactions
    result = get_account_transactions(test_db_session, account.id)

    assert result.transaction_count == 2
    assert result.total_invested == 8000.0
    assert len(result.transactions) == 2
    # Should be sorted by date desc
    assert result.transactions[0].date == date(2024, 2, 10)
    assert result.transactions[0].amount == 3000.0
    assert result.transactions[1].date == date(2024, 1, 15)
    assert result.transactions[1].amount == 5000.0


def test_get_account_transactions_non_investment_account(test_db_session):
    """Test getting transactions for non-investment account fails"""
    account = create_test_account(
        test_db_session,
        name="Bank Account",
        account_type="asset",
        category="bank",
        owner="Marcin",
    )

    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, account.id)

    assert exc_info.value.status_code == 400
    assert "cannot have transactions" in exc_info.value.detail


def test_get_account_transactions_not_found(test_db_session):
    """Test getting transactions for non-existent account fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_get_account_transactions_inactive_account(test_db_session):
    """Test getting transactions for inactive account fails"""
    account = create_test_account(
        test_db_session,
        name="Old Stock Account",
        account_type="asset",
        category="stock",
        owner="Marcin",
        is_active=False,
    )

    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, account.id)

    assert exc_info.value.status_code == 404


def test_get_all_transactions_no_filters(test_db_session):
    """Test getting all transactions without filters"""
    account1 = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="IKZE Bonds",
        account_type="asset",
        category="bond",
        owner="Ewa",
    )

    create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account2.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Ewa",
    )

    result = get_all_transactions(test_db_session)

    assert result.transaction_count == 2
    assert result.total_invested == 8000.0
    assert len(result.transactions) == 2


def test_get_all_transactions_filter_by_account(test_db_session):
    """Test filtering transactions by account"""
    account1 = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="IKZE Bonds",
        account_type="asset",
        category="bond",
        owner="Ewa",
    )

    create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account2.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Ewa",
    )

    result = get_all_transactions(test_db_session, account_id=account1.id)

    assert result.transaction_count == 1
    assert result.total_invested == 5000.0
    assert result.transactions[0].account_name == "IKE Stocks"


def test_get_all_transactions_filter_by_owner(test_db_session):
    """Test filtering transactions by owner"""
    account = create_test_account(
        test_db_session,
        name="Shared Stocks",
        account_type="asset",
        category="stock",
        owner="Shared",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Ewa",
    )

    result = get_all_transactions(test_db_session, owner="Marcin")

    assert result.transaction_count == 1
    assert result.total_invested == 5000.0
    assert result.transactions[0].owner == "Marcin"


def test_get_all_transactions_filter_by_date_range(test_db_session):
    """Test filtering transactions by date range"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=2000.0,
        transaction_date=date(2024, 3, 5),
        owner="Marcin",
    )

    result = get_all_transactions(
        test_db_session, date_from=date(2024, 2, 1), date_to=date(2024, 2, 28)
    )

    assert result.transaction_count == 1
    assert result.transactions[0].date == date(2024, 2, 10)


def test_create_transaction_success(test_db_session):
    """Test creating a transaction successfully"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    result = create_transaction(test_db_session, account.id, data)

    assert result.id is not None
    assert result.account_id == account.id
    assert result.account_name == "IKE Stocks"
    assert result.amount == 5000.0
    assert result.date == date(2024, 1, 15)
    assert result.owner == "Marcin"


def test_create_transaction_duplicate(test_db_session):
    """Test creating duplicate transaction fails"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )

    # Try to create duplicate (same account_id, date)
    data = TransactionCreate(amount=3000.0, date=date(2024, 1, 15), owner="Ewa")

    with pytest.raises(HTTPException) as exc_info:
        create_transaction(test_db_session, account.id, data)

    assert exc_info.value.status_code == 409
    assert "already exists" in exc_info.value.detail


def test_create_transaction_non_investment_account(test_db_session):
    """Test creating transaction for non-investment account fails"""
    account = create_test_account(
        test_db_session,
        name="Bank Account",
        account_type="asset",
        category="bank",
        owner="Marcin",
    )

    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_transaction(test_db_session, account.id, data)

    assert exc_info.value.status_code == 400
    assert "cannot have transactions" in exc_info.value.detail


def test_create_transaction_account_not_found(test_db_session):
    """Test creating transaction for non-existent account fails"""
    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_transaction(test_db_session, 999, data)

    assert exc_info.value.status_code == 404


def test_delete_transaction_success(test_db_session):
    """Test soft deleting a transaction"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    transaction = create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    transaction_id = transaction.id

    delete_transaction(test_db_session, account.id, transaction_id)

    deleted = test_db_session.query(Transaction).filter_by(id=transaction_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_transaction_not_found(test_db_session):
    """Test deleting non-existent transaction fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_transaction(test_db_session, 1, 999)

    assert exc_info.value.status_code == 404


def test_delete_transaction_idempotent(test_db_session):
    """Test deleting already deleted transaction is idempotent"""
    account = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )

    transaction = create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    transaction_id = transaction.id

    delete_transaction(test_db_session, account.id, transaction_id)
    delete_transaction(test_db_session, account.id, transaction_id)

    deleted = test_db_session.query(Transaction).filter_by(id=transaction_id).first()
    assert deleted.is_active is False


def test_delete_transaction_wrong_account(test_db_session):
    """Test deleting transaction with wrong account_id fails"""
    account1 = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="IKE Bonds",
        account_type="asset",
        category="bond",
        owner="Marcin",
    )

    transaction = create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )

    with pytest.raises(HTTPException) as exc_info:
        delete_transaction(test_db_session, account2.id, transaction.id)

    assert exc_info.value.status_code == 403
    assert "does not belong to account" in exc_info.value.detail


def test_get_transaction_counts(test_db_session):
    """Test getting transaction counts per account"""
    account1 = create_test_account(
        test_db_session,
        name="IKE Stocks",
        account_type="asset",
        category="stock",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="IKZE Bonds",
        account_type="asset",
        category="bond",
        owner="Ewa",
    )

    create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 15),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=3000.0,
        transaction_date=date(2024, 2, 10),
        owner="Marcin",
    )
    create_test_transaction(
        test_db_session,
        account_id=account2.id,
        amount=2000.0,
        transaction_date=date(2024, 3, 5),
        owner="Ewa",
    )
    create_test_transaction(
        test_db_session,
        account_id=account1.id,
        amount=1000.0,
        transaction_date=date(2024, 4, 1),
        owner="Marcin",
        is_active=False,
    )

    result = get_transaction_counts(test_db_session)

    assert result[account1.id] == 2
    assert result[account2.id] == 1
