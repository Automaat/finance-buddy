"""Tests for investment API endpoints"""

from datetime import date
from decimal import Decimal

from app.models import Account, Snapshot, SnapshotValue, Transaction


def test_get_stock_stats_endpoint(test_client, test_db_session):
    """Test GET /api/investment/stock-stats endpoint"""
    # Create stock account
    account = Account(
        name="Stock Account",
        type="asset",
        category="stock",
        account_wrapper=None,
        owner="Marcin",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create transaction
    transaction = Transaction(
        account_id=account.id,
        amount=Decimal("5000.0"),
        date=date(2024, 1, 1),
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(transaction)
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

    sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("6000.0"))
    test_db_session.add(sv)
    test_db_session.commit()

    # Call API
    response = test_client.get("/api/investment/stock-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["category"] == "stock"
    assert data["total_value"] == 6000.0
    assert data["total_contributed"] == 5000.0
    assert data["returns"] == 1000.0
    assert data["roi_percentage"] == 20.0


def test_get_bond_stats_endpoint(test_client, test_db_session):
    """Test GET /api/investment/bond-stats endpoint"""
    # Create bond account
    account = Account(
        name="Bond Account",
        type="asset",
        category="bond",
        account_wrapper="IKZE",
        owner="Ewa",
        currency="PLN",
        purpose="general",
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create transaction
    transaction = Transaction(
        account_id=account.id,
        amount=Decimal("10000.0"),
        date=date(2024, 1, 1),
        owner="Ewa",
        is_active=True,
    )
    test_db_session.add(transaction)
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
    test_db_session.add(snapshot)
    test_db_session.commit()

    sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("10300.0"))
    test_db_session.add(sv)
    test_db_session.commit()

    # Call API
    response = test_client.get("/api/investment/bond-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["category"] == "bond"
    assert data["total_value"] == 10300.0
    assert data["total_contributed"] == 10000.0
    assert data["returns"] == 300.0
    assert data["roi_percentage"] == 3.0


def test_get_stock_stats_no_data(test_client):
    """Test GET /api/investment/stock-stats with no stock accounts"""
    response = test_client.get("/api/investment/stock-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["category"] == "stock"
    assert data["total_value"] == 0.0
    assert data["total_contributed"] == 0.0
    assert data["returns"] == 0.0
    assert data["roi_percentage"] == 0.0


def test_get_bond_stats_no_data(test_client):
    """Test GET /api/investment/bond-stats with no bond accounts"""
    response = test_client.get("/api/investment/bond-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["category"] == "bond"
    assert data["total_value"] == 0.0
    assert data["total_contributed"] == 0.0
    assert data["returns"] == 0.0
    assert data["roi_percentage"] == 0.0
