"""Tests for investment API endpoints"""

from datetime import date

from tests.factories import (
    create_test_account,
    create_test_snapshot,
    create_test_snapshot_value,
    create_test_transaction,
)


def test_get_stock_stats_endpoint(test_client, test_db_session):
    """Test GET /api/investment/stock-stats endpoint"""
    account = create_test_account(
        test_db_session,
        name="Stock Account",
        account_type="asset",
        category="stock",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=5000.0,
        transaction_date=date(2024, 1, 1),
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 12, 31))
    create_test_snapshot_value(
        test_db_session,
        snapshot_id=snapshot.id,
        account_id=account.id,
        value=6000.0,
    )

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
    account = create_test_account(
        test_db_session,
        name="Bond Account",
        account_type="asset",
        category="bond",
        account_wrapper="IKZE",
        owner="Ewa",
    )

    create_test_transaction(
        test_db_session,
        account_id=account.id,
        amount=10000.0,
        transaction_date=date(2024, 1, 1),
        owner="Ewa",
    )

    snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 12, 31))
    create_test_snapshot_value(
        test_db_session,
        snapshot_id=snapshot.id,
        account_id=account.id,
        value=10300.0,
    )

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
