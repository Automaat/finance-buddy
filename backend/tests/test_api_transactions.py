from datetime import date

from app.models import Account, Transaction


def test_get_account_transactions_success(test_client, test_db_session):
    """Test GET /api/accounts/{account_id}/transactions"""
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
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    response = test_client.get(f"/api/accounts/{account.id}/transactions")

    assert response.status_code == 200
    data = response.json()
    assert data["transaction_count"] == 2
    assert data["total_invested"] == 8000.0
    assert len(data["transactions"]) == 2


def test_get_account_transactions_non_investment_account(test_client, test_db_session):
    """Test GET transactions for non-investment account fails"""
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    response = test_client.get(f"/api/accounts/{account.id}/transactions")

    assert response.status_code == 400
    assert "not an investment account" in response.json()["detail"]


def test_get_account_transactions_not_found(test_client, test_db_session):
    """Test GET transactions for non-existent account fails"""
    response = test_client.get("/api/accounts/999/transactions")

    assert response.status_code == 404


def test_create_transaction_success(test_client, test_db_session):
    """Test POST /api/accounts/{account_id}/transactions"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/transactions", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["account_id"] == account.id
    assert data["amount"] == 5000.0
    assert data["date"] == "2024-01-15"
    assert data["owner"] == "Marcin"


def test_create_transaction_non_investment_account(test_client, test_db_session):
    """Test POST transaction for non-investment account fails"""
    account = Account(
        name="Bank Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/transactions", json=payload)

    assert response.status_code == 400


def test_create_transaction_invalid_amount(test_client, test_db_session):
    """Test POST transaction with invalid amount fails"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": -1000.0, "date": "2024-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/transactions", json=payload)

    assert response.status_code == 422


def test_create_transaction_future_date(test_client, test_db_session):
    """Test POST transaction with future date fails"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2099-01-15", "owner": "Marcin"}

    response = test_client.post(f"/api/accounts/{account.id}/transactions", json=payload)

    assert response.status_code == 422


def test_create_transaction_invalid_owner(test_client, test_db_session):
    """Test POST transaction with invalid owner fails"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    payload = {"amount": 5000.0, "date": "2024-01-15", "owner": "InvalidOwner"}

    response = test_client.post(f"/api/accounts/{account.id}/transactions", json=payload)

    assert response.status_code == 422


def test_delete_transaction_success(test_client, test_db_session):
    """Test DELETE /api/accounts/{account_id}/transactions/{transaction_id}"""
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

    response = test_client.delete(f"/api/accounts/{account.id}/transactions/{transaction.id}")

    assert response.status_code == 204

    # Verify soft delete
    deleted = test_db_session.query(Transaction).filter_by(id=transaction.id).first()
    assert deleted.is_active is False


def test_delete_transaction_not_found(test_client, test_db_session):
    """Test DELETE non-existent transaction fails"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    response = test_client.delete(f"/api/accounts/{account.id}/transactions/999")

    assert response.status_code == 404


def test_get_all_transactions_no_filters(test_client, test_db_session):
    """Test GET /api/transactions without filters"""
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
        account_id=account1.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    response = test_client.get("/api/transactions")

    assert response.status_code == 200
    data = response.json()
    assert data["transaction_count"] == 2
    assert data["total_invested"] == 8000.0


def test_get_all_transactions_filter_by_account(test_client, test_db_session):
    """Test GET /api/transactions with account_id filter"""
    account1 = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="IKZE Bonds", type="asset", category="bond", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    t1 = Transaction(
        account_id=account1.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account2.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    response = test_client.get(f"/api/transactions?account_id={account1.id}")

    assert response.status_code == 200
    data = response.json()
    assert data["transaction_count"] == 1
    assert data["transactions"][0]["account_name"] == "IKE Stocks"


def test_get_all_transactions_filter_by_owner(test_client, test_db_session):
    """Test GET /api/transactions with owner filter"""
    account = Account(
        name="Shared Stocks", type="asset", category="stock", owner="Shared", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    t1 = Transaction(
        account_id=account.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account.id, amount=3000.0, date=date(2024, 2, 10), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2])
    test_db_session.commit()

    response = test_client.get("/api/transactions?owner=Marcin")

    assert response.status_code == 200
    data = response.json()
    assert data["transaction_count"] == 1
    assert data["transactions"][0]["owner"] == "Marcin"


def test_get_all_transactions_filter_by_date_range(test_client, test_db_session):
    """Test GET /api/transactions with date range filters"""
    account = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

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

    response = test_client.get("/api/transactions?date_from=2024-02-01&date_to=2024-02-28")

    assert response.status_code == 200
    data = response.json()
    assert data["transaction_count"] == 1
    assert data["transactions"][0]["date"] == "2024-02-10"


def test_get_transaction_counts(test_client, test_db_session):
    """Test GET /api/transactions/counts"""
    account1 = Account(
        name="IKE Stocks", type="asset", category="stock", owner="Marcin", currency="PLN"
    )
    account2 = Account(
        name="IKZE Bonds", type="asset", category="bond", owner="Ewa", currency="PLN"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    t1 = Transaction(
        account_id=account1.id, amount=5000.0, date=date(2024, 1, 15), owner="Marcin", is_active=True
    )
    t2 = Transaction(
        account_id=account1.id, amount=3000.0, date=date(2024, 2, 10), owner="Marcin", is_active=True
    )
    t3 = Transaction(
        account_id=account2.id, amount=2000.0, date=date(2024, 3, 5), owner="Ewa", is_active=True
    )
    test_db_session.add_all([t1, t2, t3])
    test_db_session.commit()

    response = test_client.get("/api/transactions/counts")

    assert response.status_code == 200
    data = response.json()
    assert data[str(account1.id)] == 2
    assert data[str(account2.id)] == 1
