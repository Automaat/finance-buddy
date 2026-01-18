from datetime import date

import pytest
from fastapi import HTTPException

from app.models import Account, Debt, DebtPayment, Snapshot, SnapshotValue
from app.schemas.debts import DebtCreate, DebtUpdate
from app.services.debts import create_debt, delete_debt, get_all_debts, get_debt, update_debt


def test_create_debt_success(test_db_session):
    """Test creating a debt successfully"""
    account = Account(
        name="Hipoteka Mieszkanie",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    data = DebtCreate(
        name="Mieszkanie Kraków",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
        notes="30-year fixed rate",
    )

    result = create_debt(test_db_session, account.id, data)

    assert result.id is not None
    assert result.account_id == account.id
    assert result.account_name == "Hipoteka Mieszkanie"
    assert result.account_owner == "Shared"
    assert result.name == "Mieszkanie Kraków"
    assert result.debt_type == "mortgage"
    assert result.start_date == date(2020, 1, 15)
    assert result.initial_amount == 500000.0
    assert result.interest_rate == 3.5
    assert result.currency == "PLN"
    assert result.notes == "30-year fixed rate"
    assert result.is_active is True


def test_create_debt_invalid_account(test_db_session):
    """Test creating debt for non-existent account fails"""
    data = DebtCreate(
        name="Test Debt",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=100000.0,
        interest_rate=3.5,
        currency="PLN",
    )

    with pytest.raises(HTTPException) as exc_info:
        create_debt(test_db_session, 999, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_create_debt_not_liability(test_db_session):
    """Test creating debt for asset account fails"""
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

    data = DebtCreate(
        name="Test Debt",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=100000.0,
        interest_rate=3.5,
        currency="PLN",
    )

    with pytest.raises(HTTPException) as exc_info:
        create_debt(test_db_session, account.id, data)

    assert exc_info.value.status_code == 400
    assert "not a liability account" in exc_info.value.detail


def test_create_debt_duplicate(test_db_session):
    """Test creating duplicate debt for same account fails"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    data1 = DebtCreate(
        name="First Debt",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    create_debt(test_db_session, account.id, data1)

    data2 = DebtCreate(
        name="Second Debt",
        debt_type="installment_0percent",
        start_date=date(2024, 1, 1),
        initial_amount=10000.0,
        interest_rate=0.0,
        currency="PLN",
    )

    with pytest.raises(HTTPException) as exc_info:
        create_debt(test_db_session, account.id, data2)

    assert exc_info.value.status_code == 409
    assert "already exists" in exc_info.value.detail


def test_get_all_debts_no_filters(test_db_session):
    """Test getting all debts without filters"""
    account1 = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Raty 0%",
        type="liability",
        category="installment",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    debt1 = Debt(
        account_id=account1.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    debt2 = Debt(
        account_id=account2.id,
        name="Laptop",
        debt_type="installment_0percent",
        start_date=date(2024, 1, 1),
        initial_amount=5000.0,
        interest_rate=0.0,
        currency="PLN",
    )
    test_db_session.add_all([debt1, debt2])
    test_db_session.commit()

    result = get_all_debts(test_db_session)

    assert result.total_count == 2
    assert result.active_debts_count == 2
    assert result.total_initial_amount == 505000.0
    assert len(result.debts) == 2


def test_get_all_debts_filter_by_account(test_db_session):
    """Test filtering debts by account"""
    account1 = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Raty 0%",
        type="liability",
        category="installment",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    debt1 = Debt(
        account_id=account1.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    debt2 = Debt(
        account_id=account2.id,
        name="Laptop",
        debt_type="installment_0percent",
        start_date=date(2024, 1, 1),
        initial_amount=5000.0,
        interest_rate=0.0,
        currency="PLN",
    )
    test_db_session.add_all([debt1, debt2])
    test_db_session.commit()

    result = get_all_debts(test_db_session, account_id=account1.id)

    assert result.total_count == 1
    assert result.debts[0].account_name == "Hipoteka"


def test_get_all_debts_filter_by_type(test_db_session):
    """Test filtering debts by debt_type"""
    account1 = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    account2 = Account(
        name="Raty 0%",
        type="liability",
        category="installment",
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    debt1 = Debt(
        account_id=account1.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    debt2 = Debt(
        account_id=account2.id,
        name="Laptop",
        debt_type="installment_0percent",
        start_date=date(2024, 1, 1),
        initial_amount=5000.0,
        interest_rate=0.0,
        currency="PLN",
    )
    test_db_session.add_all([debt1, debt2])
    test_db_session.commit()

    result = get_all_debts(test_db_session, debt_type="mortgage")

    assert result.total_count == 1
    assert result.debts[0].debt_type == "mortgage"


def test_get_debt_success(test_db_session):
    """Test getting a single debt by ID"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    result = get_debt(test_db_session, debt.id)

    assert result.id == debt.id
    assert result.account_name == "Hipoteka"
    assert result.name == "Mieszkanie"


def test_get_debt_not_found(test_db_session):
    """Test getting non-existent debt fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_debt(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_update_debt_success(test_db_session):
    """Test updating debt fields"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    update_data = DebtUpdate(interest_rate=4.0, notes="Rate increased")

    result = update_debt(test_db_session, debt.id, update_data)

    assert result.interest_rate == 4.0
    assert result.notes == "Rate increased"
    assert result.name == "Mieszkanie"


def test_update_debt_not_found(test_db_session):
    """Test updating non-existent debt fails"""
    update_data = DebtUpdate(interest_rate=4.0)

    with pytest.raises(HTTPException) as exc_info:
        update_debt(test_db_session, 999, update_data)

    assert exc_info.value.status_code == 404


def test_delete_debt_success(test_db_session):
    """Test soft deleting a debt"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()
    debt_id = debt.id

    delete_debt(test_db_session, debt_id)

    deleted = test_db_session.query(Debt).filter_by(id=debt_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_debt_idempotent(test_db_session):
    """Test deleting already deleted debt is idempotent"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()
    debt_id = debt.id

    delete_debt(test_db_session, debt_id)
    delete_debt(test_db_session, debt_id)

    deleted = test_db_session.query(Debt).filter_by(id=debt_id).first()
    assert deleted.is_active is False


def test_delete_debt_not_found(test_db_session):
    """Test deleting non-existent debt fails"""
    with pytest.raises(HTTPException) as exc_info:
        delete_debt(test_db_session, 999)

    assert exc_info.value.status_code == 404


def test_get_all_debts_with_snapshots_and_payments(test_db_session):
    """Test getting debts with snapshot balances and payments for interest calculation"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 12, 31), notes="Year-end 2024")
    test_db_session.add(snapshot)
    test_db_session.commit()

    snapshot_value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=400000.0)
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    payment1 = DebtPayment(
        account_id=account.id, amount=50000.0, date=date(2024, 6, 1), owner="Shared"
    )
    payment2 = DebtPayment(
        account_id=account.id, amount=60000.0, date=date(2024, 12, 1), owner="Shared"
    )
    test_db_session.add_all([payment1, payment2])
    test_db_session.commit()

    result = get_all_debts(test_db_session)

    assert result.total_count == 1
    debt_response = result.debts[0]
    assert debt_response.latest_balance == 400000.0
    assert debt_response.latest_balance_date == date(2024, 12, 31)
    assert debt_response.total_paid == 110000.0
    assert debt_response.interest_paid == 10000.0


def test_create_debt_with_snapshots(test_db_session):
    """Test creating debt when snapshot data exists"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 1, 31), notes="January snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

    snapshot_value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=450000.0)
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=55000.0, date=date(2024, 1, 15), owner="Shared"
    )
    test_db_session.add(payment)
    test_db_session.commit()

    data = DebtCreate(
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )

    result = create_debt(test_db_session, account.id, data)

    assert result.latest_balance == 450000.0
    assert result.latest_balance_date == date(2024, 1, 31)
    assert result.total_paid == 55000.0
    assert result.interest_paid == 5000.0


def test_get_debt_with_snapshots(test_db_session):
    """Test getting debt with snapshot balance"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 6, 30), notes="Mid-year snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

    snapshot_value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=425000.0)
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=80000.0, date=date(2024, 6, 15), owner="Shared"
    )
    test_db_session.add(payment)
    test_db_session.commit()

    result = get_debt(test_db_session, debt.id)

    assert result.latest_balance == 425000.0
    assert result.latest_balance_date == date(2024, 6, 30)
    assert result.total_paid == 80000.0
    assert result.interest_paid == 5000.0


def test_update_debt_all_fields(test_db_session):
    """Test updating all updatable debt fields"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    update_data = DebtUpdate(
        name="Updated Mieszkanie",
        debt_type="installment_0percent",
        start_date=date(2020, 2, 1),
        initial_amount=550000.0,
        interest_rate=0.0,
        currency="PLN",
        notes="Updated notes",
    )

    result = update_debt(test_db_session, debt.id, update_data)

    assert result.name == "Updated Mieszkanie"
    assert result.debt_type == "installment_0percent"
    assert result.start_date == date(2020, 2, 1)
    assert result.initial_amount == 550000.0
    assert result.interest_rate == 0.0
    assert result.currency == "PLN"
    assert result.notes == "Updated notes"


def test_update_debt_with_snapshots(test_db_session):
    """Test updating debt when snapshot data exists"""
    account = Account(
        name="Hipoteka",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    debt = Debt(
        account_id=account.id,
        name="Mieszkanie",
        debt_type="mortgage",
        start_date=date(2020, 1, 15),
        initial_amount=500000.0,
        interest_rate=3.5,
        currency="PLN",
    )
    test_db_session.add(debt)
    test_db_session.commit()

    snapshot = Snapshot(date=date(2024, 12, 31), notes="Year-end")
    test_db_session.add(snapshot)
    test_db_session.commit()

    snapshot_value = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=400000.0)
    test_db_session.add(snapshot_value)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=110000.0, date=date(2024, 12, 1), owner="Shared"
    )
    test_db_session.add(payment)
    test_db_session.commit()

    update_data = DebtUpdate(interest_rate=4.0)

    result = update_debt(test_db_session, debt.id, update_data)

    assert result.interest_rate == 4.0
    assert result.latest_balance == 400000.0
    assert result.total_paid == 110000.0
    assert result.interest_paid == 10000.0
