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


