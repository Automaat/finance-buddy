from fastapi.testclient import TestClient

from app.main import app
from app.models import Account

client = TestClient(app)


def test_get_accounts(test_db_session):
    """Test GET /api/accounts endpoint"""
    # Create test accounts
    account = Account(
        name="Test Bank", type="asset", category="bank", owner="Marcin", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    response = client.get("/api/accounts")

    assert response.status_code == 200
    data = response.json()
    assert "assets" in data
    assert "liabilities" in data
    assert len(data["assets"]) >= 1


def test_create_account_success():
    """Test POST /api/accounts with valid data"""
    payload = {
        "name": "New Investment Account",
        "type": "asset",
        "category": "ike",
        "owner": "Marcin",
        "currency": "PLN",
    }

    response = client.post("/api/accounts", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "New Investment Account"
    assert data["type"] == "asset"
    assert data["category"] == "ike"
    assert data["owner"] == "Marcin"
    assert data["currency"] == "PLN"
    assert data["is_active"] is True
    assert data["current_value"] == 0.0
    assert "id" in data
    assert "created_at" in data


def test_create_account_default_currency():
    """Test POST /api/accounts with default PLN currency"""
    payload = {
        "name": "Test Account",
        "type": "asset",
        "category": "bank",
        "owner": "Ewa",
    }

    response = client.post("/api/accounts", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["currency"] == "PLN"


def test_create_account_liability():
    """Test creating a liability account via API"""
    payload = {
        "name": "New Mortgage",
        "type": "liability",
        "category": "mortgage",
        "owner": "Shared",
        "currency": "PLN",
    }

    response = client.post("/api/accounts", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["type"] == "liability"
    assert data["category"] == "mortgage"


def test_create_account_missing_required_fields():
    """Test POST /api/accounts with missing required fields"""
    payload = {
        "name": "Incomplete Account",
        # Missing type, category, owner
    }

    response = client.post("/api/accounts", json=payload)

    assert response.status_code == 422  # Unprocessable Entity
