from datetime import date

import pytest
from fastapi import HTTPException

from app.models import Account, Transaction
from app.schemas.transactions import TransactionCreate
from app.services.transactions import (
    create_transaction,
    delete_transaction,
    get_account_transactions,
    get_all_transactions,
    get_transaction_counts,
)


def test_get_account_transactions_success(test_db_session):
    """Test getting transactions for a specific investment account"""
    # Create investment account
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create transactions
    t1 = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Marcin", is_active=True
    )
    t3 = Transaction(
        account_id=account.id, amount=2000.0, date=date(2024, 3, 5), owner="Marcin", is_active=False
    )
    test_db_session.add_all([t1, t2, t3])
    test_db_session.commit()

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
    # Create bank account (not investment)
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, account.id)

    assert exc_info.value.status_code == 400
    assert "not an investment account" in exc_info.value.detail


def test_get_account_transactions_not_found(test_db_session):
    """Test getting transactions for non-existent account fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_get_account_transactions_inactive_account(test_db_session):
    """Test getting transactions for inactive account fails"""
    account = Account(
        name="Old Stock Account",
        type="asset",
        category="stock",
        owner="Marcin",
        currency="PLN",
        is_active=False,
    )
    test_db_session.add(account)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        get_account_transactions(test_db_session, account.id)

    assert exc_info.value.status_code == 404


def test_get_all_transactions_no_filters(test_db_session):
    """Test getting all transactions without filters"""
    # Create accounts
    account1 = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="IKZE Bonds", type="asset", category="bond", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create transactions
    t1 = Transaction(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    t2 = Transaction(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    result = get_all_transactions(test_db_session)

    assert result.transaction_count == 2
    assert result.total_invested == 8000.0
    assert len(result.transactions) == 2


def test_get_all_transactions_filter_by_account(test_db_session):
    """Test filtering transactions by account"""
    # Create accounts
    account1 = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="IKZE Bonds", type="asset", category="bond", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create transactions
    t1 = Transaction(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    t2 = Transaction(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    result = get_all_transactions(test_db_session, account_id=account1.id)

    assert result.transaction_count == 1
    assert result.total_invested == 5000.0
    assert result.transactions[0].account_name == "IKE Stocks"


def test_get_all_transactions_filter_by_owner(test_db_session):
    """Test filtering transactions by owner"""
    account = Account(
        name="Shared Stocks", type="asset", category="stock", owner="Shared", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create transactions with different owners
    t1 = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    result = get_all_transactions(test_db_session, owner="Marcin")

    assert result.transaction_count == 1
    assert result.total_invested == 5000.0
    assert result.transactions[0].owner == "Marcin"


def test_get_all_transactions_filter_by_date_range(test_db_session):
    """Test filtering transactions by date range"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create transactions
    t1 = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Marcin", is_active=True
    )
    t3 = Transaction(
        account_id=account.id, amount=2000.0, date=date(2024, 3, 5), owner="Marcin", is_active=True
    )
    test_db_session.add_all([t1, t2, t3])
    test_db_session.commit()

    result = get_all_transactions(
        test_db_session, date_from=date(2024, 2, 1), date_to=date(2024, 2, 28)
    )

    assert result.transaction_count == 1
    assert result.transactions[0].date == date(2024, 2, 10)


def test_create_transaction_success(test_db_session):
    """Test creating a transaction successfully"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    result = create_transaction(test_db_session, account.id, data)

    assert result.id is not None
    assert result.account_id == account.id
    assert result.account_name == "IKE Stocks"
    assert result.amount == 5000.0
    assert result.date == date(2024, 1, 15)
    assert result.owner == "Marcin"


def test_create_transaction_non_investment_account(test_db_session):
    """Test creating transaction for non-investment account fails"""
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_transaction(test_db_session, account.id, data)

    assert exc_info.value.status_code == 400
    assert "not an investment account" in exc_info.value.detail


def test_create_transaction_account_not_found(test_db_session):
    """Test creating transaction for non-existent account fails"""
    data = TransactionCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_transaction(test_db_session, 999, data)

    assert exc_info.value.status_code == 404


def test_delete_transaction_success(test_db_session):
    """Test soft deleting a transaction"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    transaction = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    test_db_session.add(transaction)
    test_db_session.commit()
    transaction_id = transaction.id

    delete_transaction(test_db_session, transaction_id)

    deleted = test_db_session.query(Transaction).filter_by(id=transaction_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_transaction_not_found(test_db_session):
    """Test deleting non-existent transaction fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_transaction(test_db_session, 999)

    assert exc_info.value.status_code == 404


def test_delete_transaction_idempotent(test_db_session):
    """Test deleting already deleted transaction is idempotent"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    transaction = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    test_db_session.add(transaction)
    test_db_session.commit()
    transaction_id = transaction.id

    delete_transaction(test_db_session, transaction_id)
    delete_transaction(test_db_session, transaction_id)

    deleted = test_db_session.query(Transaction).filter_by(id=transaction_id).first()
    assert deleted.is_active is False


def test_get_transaction_counts(test_db_session):
    """Test getting transaction counts per account"""
    # Create accounts
    account1 = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="IKZE Bonds", type="asset", category="bond", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create transactions
    t1 = Transaction(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    t2 = Transaction(
        account_id=account1.id,
        amount=3000.0,
        date=date(2024, 2, 10),
        owner="Marcin",
        is_active=True,
    )
    t3 = Transaction(
        account_id=account2.id, amount=2000.0, date=date(2024, 3, 5), owner="Ewa", is_active=True
    )
    t4 = Transaction(
        account_id=account1.id,
        amount=1000.0,
        date=date(2024, 4, 1),
        owner="Marcin",
        is_active=False,
    )
    test_db_session.add_all([t1, t2, t3, t4])
    test_db_session.commit()

    result = get_transaction_counts(test_db_session)

    assert result[account1.id] == 2
    assert result[account2.id] == 1
