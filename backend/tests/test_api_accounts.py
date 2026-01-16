def test_create_account_missing_required_fields(test_client):
    """Test POST /api/accounts with missing required fields"""
    payload = {
        "name": "Incomplete Account",
        # Missing type, category, owner
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422  # Unprocessable Entity


def test_create_account_invalid_type(test_client):
    """Test POST /api/accounts with invalid type"""
    payload = {
        "name": "Invalid Account",
        "type": "invalid_type",
        "category": "bank",
        "owner": "Marcin",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_create_account_invalid_category(test_client):
    """Test POST /api/accounts with invalid category"""
    payload = {
        "name": "Invalid Category Account",
        "type": "asset",
        "category": "invalid_category",
        "owner": "Marcin",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_create_account_invalid_owner(test_client):
    """Test POST /api/accounts with invalid owner"""
    payload = {
        "name": "Invalid Owner Account",
        "type": "asset",
        "category": "bank",
        "owner": "InvalidOwner",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_update_account_valid(test_client, test_db_session):
    """Test PUT /api/accounts/{id} with valid data"""
    from app.models import Account

    # Create account
    account = Account(
        name="Original Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()
    account_id = account.id

    # Update account
    payload = {"name": "Updated Account", "category": "ike", "owner": "Ewa"}

    response = test_client.put(f"/api/accounts/{account_id}", json=payload)

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == account_id
    assert data["name"] == "Updated Account"
    assert data["category"] == "ike"
    assert data["owner"] == "Ewa"
    assert data["type"] == "asset"  # unchanged


def test_update_account_invalid_category(test_client, test_db_session):
    """Test PUT /api/accounts/{id} with invalid category"""
    from app.models import Account

    # Create account
    account = Account(
        name="Test Account", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()
    account_id = account.id

    # Try to update with invalid category
    payload = {"category": "invalid_category"}

    response = test_client.put(f"/api/accounts/{account_id}", json=payload)

    assert response.status_code == 422


def test_update_account_not_found(test_client):
    """Test PUT /api/accounts/{id} with non-existent id"""
    payload = {"name": "Updated Name"}

    response = test_client.put("/api/accounts/999", json=payload)

    assert response.status_code == 404
    assert "not found" in response.json()["detail"]


def test_delete_account_success(test_client, test_db_session):
    """Test DELETE /api/accounts/{id} successfully"""
    from app.models import Account

    # Create account
    account = Account(
        name="To Delete", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()
    account_id = account.id

    # Delete account
    response = test_client.delete(f"/api/accounts/{account_id}")

    assert response.status_code == 204

    # Verify account is soft deleted
    deleted = test_db_session.query(Account).filter_by(id=account_id).first()
    assert deleted is not None
    assert deleted.is_active is False


def test_delete_account_not_found(test_client):
    """Test DELETE /api/accounts/{id} with non-existent id"""
    response = test_client.delete("/api/accounts/999")

    assert response.status_code == 404
    assert "not found" in response.json()["detail"]


def test_delete_account_removed_from_list(test_client, test_db_session):
    """Test deleted account doesn't appear in GET /api/accounts"""
    from app.models import Account

    # Create account
    account = Account(
        name="To Remove", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()
    account_id = account.id

    # Verify it appears in list
    response = test_client.get("/api/accounts")
    assert response.status_code == 200
    data = response.json()
    assert any(acc["id"] == account_id for acc in data["assets"])

    # Delete account
    delete_response = test_client.delete(f"/api/accounts/{account_id}")
    assert delete_response.status_code == 204

    # Verify it doesn't appear in list
    response = test_client.get("/api/accounts")
    assert response.status_code == 200
    data = response.json()
    assert not any(acc["id"] == account_id for acc in data["assets"])
