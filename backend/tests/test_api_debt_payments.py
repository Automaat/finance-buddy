from datetime import date

from app.models import Account, DebtPayment


def test_get_account_payments_success(test_client, test_db_session):
    """Test GET /api/accounts/{account_id}/payments"""
    # Create liability account
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
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
    test_db_session.add_all([p1, p2])
    test_db_session.commit()

    response = test_client.get(f"/api/accounts/{account.id}/payments")

    assert response.status_code == 200
    data = response.json()
    assert data["payment_count"] == 2
    assert data["total_paid"] == 8000.0
    assert len(data["payments"]) == 2


def test_get_account_payments_non_liability_account(test_client, test_db_session):
    """Test GET payments for non-liability account fails"""
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    response = test_client.get(f"/api/accounts/{account.id}/payments")

    assert response.status_code == 400
    assert "not a liability account" in response.json()["detail"]


def test_get_account_payments_not_found(test_client):
    """Test GET payments for non-existent account fails"""
    response = test_client.get("/api/accounts/999/payments")

    assert response.status_code == 404


def test_create_payment_success(test_client, test_db_session):
    """Test POST /api/accounts/{account_id}/payments"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/payments", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["account_id"] == account.id
    assert data["amount"] == 5000.0
    assert data["date"] == "2024-01-15"
    assert data["owner"] == "Marcin"


def test_create_payment_non_liability_account(test_client, test_db_session):
    """Test POST payment for non-liability account fails"""
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/payments", json=payload)

    assert response.status_code == 400


def test_create_payment_invalid_amount(test_client, test_db_session):
    """Test POST payment with invalid amount fails"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": -1000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/payments", json=payload)

    assert response.status_code == 422


def test_create_payment_future_date(test_client, test_db_session):
    """Test POST payment with future date fails"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2099-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/payments", json=payload)

    assert response.status_code == 422


def test_create_payment_invalid_owner(test_client, test_db_session):
    """Test POST payment with invalid owner fails"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "InvalidOwner"}

    response = test_client.post(f"/api/accounts/{account.id}/payments", json=payload)

    assert response.status_code == 422


def test_delete_payment_success(test_client, test_db_session):
    """Test DELETE /api/accounts/{account_id}/payments/{payment_id}"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payment = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    test_db_session.add(payment)
    test_db_session.commit()
    payment_id = payment.id

    response = test_client.delete(f"/api/accounts/{account.id}/payments/{payment_id}")

    assert response.status_code == 204

    # Verify soft delete - expire session to see updates from test_client
    test_db_session.expire_all()
    deleted = test_db_session.query(DebtPayment).filter_by(id=payment_id).first()
    assert deleted.is_active is False


def test_delete_payment_not_found(test_client, test_db_session):
    """Test DELETE non-existent payment fails"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    response = test_client.delete(f"/api/accounts/{account.id}/payments/999")

    assert response.status_code == 404


def test_get_all_payments_no_filters(test_client, test_db_session):
    """Test GET /api/payments without filters"""
    # Create accounts
    account1 = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="Installment", type="liability", category="installment", owner="Ewa", currency="PLN"
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

    response = test_client.get("/api/payments")

    assert response.status_code == 200
    data = response.json()
    assert data["payment_count"] == 2
    assert data["total_paid"] == 8000.0


def test_get_all_payments_filter_by_account(test_client, test_db_session):
    """Test GET /api/payments with account_id filter"""
    account1 = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="Installment", type="liability", category="installment", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

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

    response = test_client.get(f"/api/payments?account_id={account1.id}")

    assert response.status_code == 200
    data = response.json()
    assert data["payment_count"] == 1
    assert data["payments"][0]["account_name"] == "Mortgage"


def test_get_all_payments_filter_by_owner(test_client, test_db_session):
    """Test GET /api/payments with owner filter"""
    account = Account(
        name="Shared Mortgage",
        type="liability",
        category="mortgage",
        owner="Shared",
        currency="PLN",
    )
    test_db_session.add(account)
    test_db_session.commit()

    p1 = DebtPayment(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    p2 = DebtPayment(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([p1, p2])
    test_db_session.commit()

    response = test_client.get("/api/payments?owner=Marcin")

    assert response.status_code == 200
    data = response.json()
    assert data["payment_count"] == 1
    assert data["payments"][0]["owner"] == "Marcin"


def test_get_all_payments_filter_by_date_range(test_client, test_db_session):
    """Test GET /api/payments with date range filters"""
    account = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

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

    response = test_client.get("/api/payments?date_from=2024-02-01&date_to=2024-02-28")

    assert response.status_code == 200
    data = response.json()
    assert data["payment_count"] == 1
    assert data["payments"][0]["date"] == "2024-02-10"


def test_get_payment_counts(test_client, test_db_session):
    """Test GET /api/payments/counts"""
    account1 = Account(
        name="Mortgage", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="Installment", type="liability", category="installment", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

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
    test_db_session.add_all([p1, p2, p3])
    test_db_session.commit()

    response = test_client.get("/api/payments/counts")

    assert response.status_code == 200
    data = response.json()
    assert data[str(account1.id)] == 2
    assert data[str(account2.id)] == 1


def test_delete_payment_wrong_account(test_client, test_db_session):
    """Test DELETE payment with wrong account_id fails"""
    account1 = Account(
        name="Mortgage 1", type="liability", category="mortgage", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="Mortgage 2", type="liability", category="mortgage", owner="Marcin", currency="PLN"
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

    response = test_client.delete(f"/api/accounts/{account2.id}/payments/{payment.id}")

    assert response.status_code == 403
    assert "does not belong to account" in response.json()["detail"]
