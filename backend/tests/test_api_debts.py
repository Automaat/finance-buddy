from datetime import date

from tests.factories import create_test_account, create_test_debt


def test_get_all_debts_empty(test_client):
    """Test GET /api/debts returns empty list"""
    response = test_client.get("/api/debts")

    assert response.status_code == 200
    data = response.json()
    assert data["debts"] == []
    assert data["total_count"] == 0
    assert data["active_debts_count"] == 0


def test_get_all_debts_success(test_client, test_db_session):
    """Test GET /api/debts returns all active debts"""
    # Create liability accounts
    account1 = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="Car Loan",
        account_type="liability",
        category="installment",
        owner="Ewa",
    )

    # Create debts
    create_test_debt(
        test_db_session,
        account_id=account1.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )
    create_test_debt(
        test_db_session,
        account_id=account2.id,
        name="Car Financing",
        debt_type="installment_0percent",
        start_date=date(2023, 6, 1),
        initial_amount=80000.0,
        interest_rate=5.0,
    )

    response = test_client.get("/api/debts")

    assert response.status_code == 200
    data = response.json()
    assert len(data["debts"]) == 2
    assert data["total_count"] == 2
    assert data["active_debts_count"] == 2
    assert data["total_initial_amount"] == 580000.0


def test_get_all_debts_filter_by_account(test_client, test_db_session):
    """Test GET /api/debts?account_id=X filters by account"""
    account1 = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="Car Loan",
        account_type="liability",
        category="installment",
        owner="Ewa",
    )

    create_test_debt(
        test_db_session,
        account_id=account1.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )
    create_test_debt(
        test_db_session,
        account_id=account2.id,
        name="Car Financing",
        debt_type="installment_0percent",
        start_date=date(2023, 6, 1),
        initial_amount=80000.0,
        interest_rate=5.0,
    )

    response = test_client.get(f"/api/debts?account_id={account1.id}")

    assert response.status_code == 200
    data = response.json()
    assert len(data["debts"]) == 1
    assert data["debts"][0]["account_id"] == account1.id
    assert data["debts"][0]["name"] == "Home Mortgage"


def test_get_all_debts_filter_by_type(test_client, test_db_session):
    """Test GET /api/debts?debt_type=X filters by type"""
    account1 = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )
    account2 = create_test_account(
        test_db_session,
        name="Car Loan",
        account_type="liability",
        category="installment",
        owner="Ewa",
    )

    create_test_debt(
        test_db_session,
        account_id=account1.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )
    create_test_debt(
        test_db_session,
        account_id=account2.id,
        name="Car Financing",
        debt_type="installment_0percent",
        start_date=date(2023, 6, 1),
        initial_amount=80000.0,
        interest_rate=5.0,
    )

    response = test_client.get("/api/debts?debt_type=mortgage")

    assert response.status_code == 200
    data = response.json()
    assert len(data["debts"]) == 1
    assert data["debts"][0]["debt_type"] == "mortgage"


def test_create_debt_success(test_client, test_db_session):
    """Test POST /api/accounts/{account_id}/debts"""
    account = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )

    payload = {
        "name": "Home Mortgage",
        "debt_type": "mortgage",
        "start_date": "2020-01-01",
        "initial_amount": 500000.0,
        "interest_rate": 3.5,
        "currency": "PLN",
    }

    response = test_client.post(f"/api/accounts/{account.id}/debts", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["account_id"] == account.id
    assert data["name"] == "Home Mortgage"
    assert data["debt_type"] == "mortgage"
    assert data["initial_amount"] == 500000.0
    assert data["interest_rate"] == 3.5
    assert data["is_active"] is True


def test_create_debt_account_not_found(test_client):
    """Test POST debt for non-existent account fails"""
    payload = {
        "name": "Home Mortgage",
        "debt_type": "mortgage",
        "start_date": "2020-01-01",
        "initial_amount": 500000.0,
        "interest_rate": 3.5,
        "currency": "PLN",
    }

    response = test_client.post("/api/accounts/999/debts", json=payload)

    assert response.status_code == 404


def test_create_debt_not_liability_account(test_client, test_db_session):
    """Test POST debt for non-liability account fails"""
    account = create_test_account(
        test_db_session,
        name="Bank Account",
        account_type="asset",
        category="bank",
        owner="Marcin",
    )

    payload = {
        "name": "Invalid Debt",
        "debt_type": "mortgage",
        "start_date": "2020-01-01",
        "initial_amount": 100000.0,
        "interest_rate": 3.5,
        "currency": "PLN",
    }

    response = test_client.post(f"/api/accounts/{account.id}/debts", json=payload)

    assert response.status_code == 400
    assert "not a liability account" in response.json()["detail"]


def test_get_debt_success(test_client, test_db_session):
    """Test GET /api/debts/{debt_id}"""
    account = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )

    debt = create_test_debt(
        test_db_session,
        account_id=account.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )

    response = test_client.get(f"/api/debts/{debt.id}")

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == debt.id
    assert data["name"] == "Home Mortgage"
    assert data["account_id"] == account.id


def test_get_debt_not_found(test_client):
    """Test GET debt with invalid ID fails"""
    response = test_client.get("/api/debts/999")

    assert response.status_code == 404


def test_update_debt_success(test_client, test_db_session):
    """Test PUT /api/debts/{debt_id}"""
    account = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )

    debt = create_test_debt(
        test_db_session,
        account_id=account.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )

    payload = {"name": "Updated Mortgage Name", "interest_rate": 4.0}

    response = test_client.put(f"/api/debts/{debt.id}", json=payload)

    assert response.status_code == 200
    data = response.json()
    assert data["name"] == "Updated Mortgage Name"
    assert data["interest_rate"] == 4.0
    assert data["initial_amount"] == 500000.0  # unchanged


def test_update_debt_not_found(test_client):
    """Test PUT debt with invalid ID fails"""
    payload = {"name": "Updated Name"}

    response = test_client.put("/api/debts/999", json=payload)

    assert response.status_code == 404


def test_delete_debt_success(test_client, test_db_session):
    """Test DELETE /api/debts/{debt_id}"""
    account = create_test_account(
        test_db_session,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Marcin",
    )

    debt = create_test_debt(
        test_db_session,
        account_id=account.id,
        name="Home Mortgage",
        debt_type="mortgage",
        start_date=date(2020, 1, 1),
        initial_amount=500000.0,
        interest_rate=3.5,
    )

    response = test_client.delete(f"/api/debts/{debt.id}")

    assert response.status_code == 204

    # Verify debt is soft deleted
    get_response = test_client.get(f"/api/debts/{debt.id}")
    assert get_response.status_code == 404


def test_delete_debt_not_found(test_client):
    """Test DELETE debt with invalid ID fails"""
    response = test_client.delete("/api/debts/999")

    assert response.status_code == 404
