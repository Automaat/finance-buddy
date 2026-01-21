from datetime import date

import pytest
from fastapi import HTTPException

from app.models import Account, DebtPayment
from app.schemas.debt_payments import DebtPaymentCreate
from app.services.debt_payments import (
    create_payment,
    delete_payment,
    get_account_payments,
    get_all_payments,
    get_payment_counts,
)


def test_get_account_payments_success(test_db_session):
    """Test getting payments for a specific liability account"""
    # Create liability account
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create payments
    p1 = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    p2 = DebtPayment(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Marcin", is_active=True
    )
    p3 = DebtPayment(
        account_id=account.id, amount=2000.0, date=date(2024, 3, 5), owner="Marcin", is_active=False
    )
    test_db_session.add_all([p1, p2, p3])
    test_db_session.commit()

    # Get payments
    result = get_account_payments(test_db_session, account.id)

    assert result.payment_count == 2
    assert result.total_paid == 8000.0
    assert len(result.payments) == 2
    # Should be sorted by date desc
    assert result.payments[0].date == date(2024, 2, 10)
    assert result.payments[0].amount == 3000.0
    assert result.payments[1].date == date(2024, 1, 15)
    assert result.payments[1].amount == 5000.0


def test_get_account_payments_non_liability_account(test_db_session):
    """Test getting payments for non-liability account fails"""
    # Create asset account (not liability)
    account = Account(
        name="Bank Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        get_account_payments(test_db_session, account.id)

    assert exc_info.value.status_code == 400
    assert "not a liability account" in exc_info.value.detail


def test_get_account_payments_not_found(test_db_session):
    """Test getting payments for non-existent account fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_account_payments(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_get_account_payments_inactive_account(test_db_session):
    """Test getting payments for inactive account fails"""
    account = Account(
        name="Old Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        is_active=False,
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        get_account_payments(test_db_session, account.id)

    assert exc_info.value.status_code == 404


def test_get_all_payments_no_filters(test_db_session):
    """Test getting all payments without filters"""
    # Create accounts
    account1 = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Installment",
        type="liability",
        category="installment",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create payments
    p1 = DebtPayment(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    p2 = DebtPayment(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([p1, p2])
    test_db_session.commit()

    result = get_all_payments(test_db_session)

    assert result.payment_count == 2
    assert result.total_paid == 8000.0
    assert len(result.payments) == 2


def test_get_all_payments_filter_by_account(test_db_session):
    """Test filtering payments by account"""
    # Create accounts
    account1 = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Installment",
        type="liability",
        category="installment",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create payments
    p1 = DebtPayment(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    p2 = DebtPayment(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([p1, p2])
    test_db_session.commit()

    result = get_all_payments(test_db_session, account_id=account1.id)

    assert result.payment_count == 1
    assert result.total_paid == 5000.0
    assert result.payments[0].account_name == "Mortgage"


def test_get_all_payments_filter_by_owner(test_db_session):
    """Test filtering payments by owner"""
    account = Account(
        name="Shared Mortgage",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create payments with different owners
    p1 = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    p2 = DebtPayment(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([p1, p2])
    test_db_session.commit()

    result = get_all_payments(test_db_session, owner="Marcin")

    assert result.payment_count == 1
    assert result.total_paid == 5000.0
    assert result.payments[0].owner == "Marcin"


def test_get_all_payments_filter_by_date_range(test_db_session):
    """Test filtering payments by date range"""
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create payments
    p1 = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    p2 = DebtPayment(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Marcin", is_active=True
    )
    p3 = DebtPayment(
        account_id=account.id, amount=2000.0, date=date(2024, 3, 5), owner="Marcin", is_active=True
    )
    test_db_session.add_all([p1, p2, p3])
    test_db_session.commit()

    result = get_all_payments(
        test_db_session, date_from=date(2024, 2, 1), date_to=date(2024, 2, 28)
    )

    assert result.payment_count == 1
    assert result.payments[0].date == date(2024, 2, 10)


def test_create_payment_success(test_db_session):
    """Test creating a payment successfully"""
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    data = DebtPaymentCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    result = create_payment(test_db_session, account.id, data)

    assert result.id is not None
    assert result.account_id == account.id
    assert result.account_name == "Mortgage"
    assert result.amount == 5000.0
    assert result.date == date(2024, 1, 15)
    assert result.owner == "Marcin"


def test_create_payment_duplicate(test_db_session):
    """Test creating duplicate payment fails"""
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create first payment
    payment = DebtPayment(
        account_id=account.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(payment)
    test_db_session.commit()

    # Try to create duplicate (same account_id, date)
    data = DebtPaymentCreate(amount=3000.0, date=date(2024, 1, 15), owner="Ewa")

    with pytest.raises(HTTPException) as exc_info:
        create_payment(test_db_session, account.id, data)

    assert exc_info.value.status_code == 409
    assert "already exists" in exc_info.value.detail


def test_create_payment_non_liability_account(test_db_session):
    """Test creating payment for non-liability account fails"""
    account = Account(
        name="Bank Account",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    data = DebtPaymentCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_payment(test_db_session, account.id, data)

    assert exc_info.value.status_code == 400
    assert "not a liability account" in exc_info.value.detail


def test_create_payment_account_not_found(test_db_session):
    """Test creating payment for non-existent account fails"""
    data = DebtPaymentCreate(amount=5000.0, date=date(2024, 1, 15), owner="Marcin")

    with pytest.raises(HTTPException) as exc_info:
        create_payment(test_db_session, 999, data)

    assert exc_info.value.status_code == 404


def test_delete_payment_success(test_db_session):
    """Test soft deleting a payment"""
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    test_db_session.add(payment)
    test_db_session.commit()
    payment_id = payment.id

    delete_payment(test_db_session, account.id, payment_id)

    deleted = test_db_session.query(DebtPayment).filter_by(id=payment_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_payment_not_found(test_db_session):
    """Test deleting non-existent payment fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_payment(test_db_session, 1, 999)

    assert exc_info.value.status_code == 404


def test_delete_payment_idempotent(test_db_session):
    """Test deleting already deleted payment is idempotent"""
    account = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    test_db_session.add(payment)
    test_db_session.commit()
    payment_id = payment.id

    delete_payment(test_db_session, account.id, payment_id)
    delete_payment(test_db_session, account.id, payment_id)

    deleted = test_db_session.query(DebtPayment).filter_by(id=payment_id).first()
    assert deleted.is_active is False


def test_delete_payment_wrong_account(test_db_session):
    """Test deleting payment with wrong account_id fails"""
    account1 = Account(
        name="Mortgage 1",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Mortgage 2",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(payment)
    test_db_session.commit()

    with pytest.raises(HTTPException) as exc_info:
        delete_payment(test_db_session, account2.id, payment.id)

    assert exc_info.value.status_code == 403
    assert "does not belong to account" in exc_info.value.detail


def test_get_payment_counts(test_db_session):
    """Test getting payment counts per account"""
    # Create accounts
    account1 = Account(
        name="Mortgage",
        type="liability",
        category="mortgage",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Installment",
        type="liability",
        category="installment",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    # Create payments
    p1 = DebtPayment(
        account_id=account1.id,
        amount=5000.0,
        date=date(2024, 1, 15),
        owner="Marcin",
        is_active=True,
    )
    p2 = DebtPayment(
        account_id=account1.id,
        amount=3000.0,
        date=date(2024, 2, 10),
        owner="Marcin",
        is_active=True,
    )
    p3 = DebtPayment(
        account_id=account2.id, amount=2000.0, date=date(2024, 3, 5), owner="Ewa", is_active=True
    )
    p4 = DebtPayment(
        account_id=account1.id,
        amount=1000.0,
        date=date(2024, 4, 1),
        owner="Marcin",
        is_active=False,
    )
    test_db_session.add_all([p1, p2, p3, p4])
    test_db_session.commit()

    result = get_payment_counts(test_db_session)

    assert result[account1.id] == 2
    assert result[account2.id] == 1
