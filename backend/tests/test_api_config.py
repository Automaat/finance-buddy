from datetime import UTC


def test_get_config_not_found(test_client):
    """Test GET /api/config returns 404 when config doesn't exist"""
    response = test_client.get("/api/config")

    assert response.status_code == 404
    assert response.json()["detail"] == "Configuration not initialized"


def test_create_config_success(test_client):
    """Test PUT /api/config creates new configuration"""
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 60,
        "allocation_bonds": 30,
        "allocation_gold": 8,
        "allocation_commodities": 2,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == 1
    assert data["birth_date"] == "1990-01-01"
    assert data["retirement_age"] == 67
    assert data["retirement_monthly_salary"] == "8000.00"
    assert data["allocation_real_estate"] == 20
    assert data["allocation_stocks"] == 60
    assert data["allocation_bonds"] == 30
    assert data["allocation_gold"] == 8
    assert data["allocation_commodities"] == 2
    assert data["monthly_expenses"] == "5000.00"
    assert data["monthly_mortgage_payment"] == "3000.00"


def test_get_config_success(test_client):
    """Test GET /api/config returns config after creation"""
    # Create config first
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 60,
        "allocation_bonds": 30,
        "allocation_gold": 8,
        "allocation_commodities": 2,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }
    test_client.put("/api/config", json=payload)

    # Get config
    response = test_client.get("/api/config")

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == 1
    assert data["birth_date"] == "1990-01-01"
    assert data["retirement_age"] == 67


def test_update_config_success(test_client):
    """Test PUT /api/config updates existing configuration"""
    # Create initial config
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 60,
        "allocation_bonds": 30,
        "allocation_gold": 8,
        "allocation_commodities": 2,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }
    test_client.put("/api/config", json=payload)

    # Update config
    updated_payload = {
        "birth_date": "1985-06-15",
        "retirement_age": 70,
        "retirement_monthly_salary": 10000,
        "allocation_real_estate": 30,
        "allocation_stocks": 50,
        "allocation_bonds": 35,
        "allocation_gold": 10,
        "allocation_commodities": 5,
        "monthly_expenses": 6000,
        "monthly_mortgage_payment": 4000,
    }
    response = test_client.put("/api/config", json=updated_payload)

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == 1  # Same ID
    assert data["birth_date"] == "1985-06-15"
    assert data["retirement_age"] == 70
    assert data["retirement_monthly_salary"] == "10000.00"
    assert data["allocation_real_estate"] == 30
    assert data["allocation_stocks"] == 50


def test_create_config_invalid_allocation_sum(test_client):
    """Test PUT /api/config fails when market allocation sum != 100%"""
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 60,
        "allocation_bonds": 30,
        "allocation_gold": 5,
        "allocation_commodities": 3,  # Market sum = 98%
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 422
    assert "Market allocations must sum to 100%" in str(response.json())


def test_create_config_invalid_retirement_age(test_client):
    """Test PUT /api/config fails with invalid retirement age"""
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 150,  # Invalid
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 48,
        "allocation_bonds": 24,
        "allocation_gold": 5,
        "allocation_commodities": 3,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 422


def test_create_config_invalid_allocation_range(test_client):
    """Test PUT /api/config fails with allocation outside 0-100 range"""
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 120,  # Invalid
        "allocation_bonds": -10,  # Invalid
        "allocation_gold": 5,
        "allocation_commodities": 3,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 422


def test_create_config_negative_retirement_salary(test_client):
    """Test PUT /api/config fails with negative retirement monthly salary"""
    payload = {
        "birth_date": "1990-01-01",
        "retirement_age": 67,
        "retirement_monthly_salary": -1000,  # Invalid
        "allocation_real_estate": 20,
        "allocation_stocks": 48,
        "allocation_bonds": 24,
        "allocation_gold": 5,
        "allocation_commodities": 3,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 422


def test_create_config_future_birth_date(test_client):
    """Test PUT /api/config fails with future birth date"""
    from datetime import datetime, timedelta

    future_date = (datetime.now(tz=UTC).date() + timedelta(days=1)).isoformat()
    payload = {
        "birth_date": future_date,
        "retirement_age": 67,
        "retirement_monthly_salary": 8000,
        "allocation_real_estate": 20,
        "allocation_stocks": 48,
        "allocation_bonds": 24,
        "allocation_gold": 5,
        "allocation_commodities": 3,
        "monthly_expenses": 5000,
        "monthly_mortgage_payment": 3000,
    }

    response = test_client.put("/api/config", json=payload)

    assert response.status_code == 422
