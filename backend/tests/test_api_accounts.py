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


def test_create_account_invalid_purpose(test_client):
    """Test POST /api/accounts with invalid purpose"""
    payload = {
        "name": "Invalid Purpose Account",
        "type": "asset",
        "category": "bank",
        "owner": "Marcin",
        "purpose": "invalid_purpose",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_create_account_invalid_wrapper(test_client):
    """Test POST /api/accounts with invalid account wrapper"""
    payload = {
        "name": "Invalid Wrapper Account",
        "type": "asset",
        "category": "bank",
        "owner": "Marcin",
        "purpose": "general",
        "account_wrapper": "INVALID",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_update_account_invalid_purpose(test_client):
    """Test updating account with invalid purpose fails"""
    payload = {"purpose": "invalid_purpose"}

    response = test_client.put("/api/accounts/1", json=payload)

    assert response.status_code == 422


def test_create_account_empty_name(test_client):
    """Test creating account with empty name fails"""
    payload = {
        "name": "",
        "type": "asset",
        "category": "bank",
        "owner": "Marcin",
        "purpose": "general",
    }

    response = test_client.post("/api/accounts", json=payload)

    assert response.status_code == 422


def test_update_account_empty_name(test_client):
    """Test updating account with empty name fails"""
    payload = {"name": ""}

    response = test_client.put("/api/accounts/1", json=payload)

    assert response.status_code == 422


def test_update_account_invalid_category(test_client):
    """Test updating account with invalid category fails"""
    payload = {"category": "invalid_category"}

    response = test_client.put("/api/accounts/1", json=payload)

    assert response.status_code == 422
