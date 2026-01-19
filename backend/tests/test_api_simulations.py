from datetime import date
from decimal import Decimal

from fastapi.testclient import TestClient
from sqlalchemy.orm import Session

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.retirement_limit import RetirementLimit
from app.models.snapshot import Snapshot, SnapshotValue


def create_app_config():
    """Helper to create a standard AppConfig for tests"""
    return AppConfig(
        id=1,
        birth_date=date(1990, 1, 1),
        retirement_age=65,
        retirement_monthly_salary=Decimal("5000"),
        allocation_real_estate=20,
        allocation_stocks=50,
        allocation_bonds=20,
        allocation_gold=5,
        allocation_commodities=5,
    )


def test_simulate_retirement_success(test_client: TestClient, test_db_session: Session):
    """Test successful retirement simulation"""
    # Setup
    config = create_app_config()
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    payload = {
        "current_age": 30,
        "retirement_age": 35,
        "simulate_ike_marcin": True,
        "simulate_ike_ewa": False,
        "simulate_ikze_marcin": False,
        "simulate_ikze_ewa": False,
        "ike_marcin_balance": 10000.0,
        "ike_marcin_auto_fill": True,
        "ike_marcin_monthly": 0,
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 200
    data = response.json()

    assert "inputs" in data
    assert "simulations" in data
    assert "summary" in data
    assert len(data["simulations"]) == 1
    assert data["simulations"][0]["account_name"] == "IKE (Marcin)"
    assert data["summary"]["years_until_retirement"] == 5
    assert data["summary"]["total_final_balance"] > 0


def test_simulate_retirement_invalid_return_rate(test_client: TestClient):
    """Test validation of return rate"""
    payload = {
        "current_age": 30,
        "retirement_age": 35,
        "simulate_ike_marcin": True,
        "ike_marcin_balance": 10000.0,
        "ike_marcin_auto_fill": True,
        "annual_return_rate": 100.0,  # Invalid - exceeds 50%
        "limit_growth_rate": 5.0,
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 422


def test_simulate_retirement_missing_fields(test_client: TestClient):
    """Test missing required fields"""
    payload = {
        "current_age": 30,
        # Missing retirement_age
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 422


def test_simulate_retirement_all_accounts(test_client: TestClient, test_db_session: Session):
    """Test simulation with all accounts enabled"""
    config = create_app_config()
    test_db_session.add(config)

    for wrapper in ["IKE", "IKZE"]:
        for owner in ["Marcin", "Ewa"]:
            limit_amount = 28260.0 if wrapper == "IKE" else 11304.0
            limit = RetirementLimit(
                year=2026,
                account_wrapper=wrapper,
                owner=owner,
                limit_amount=limit_amount,
            )
            test_db_session.add(limit)

    test_db_session.commit()

    payload = {
        "current_age": 30,
        "retirement_age": 40,
        "simulate_ike_marcin": True,
        "simulate_ike_ewa": True,
        "simulate_ikze_marcin": True,
        "simulate_ikze_ewa": True,
        "ike_marcin_balance": 10000.0,
        "ike_ewa_balance": 5000.0,
        "ikze_marcin_balance": 3000.0,
        "ikze_ewa_balance": 2000.0,
        "ike_marcin_auto_fill": True,
        "ike_ewa_auto_fill": True,
        "ikze_marcin_auto_fill": True,
        "ikze_ewa_auto_fill": True,
        "marcin_tax_rate": 17.0,
        "ewa_tax_rate": 17.0,
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 200
    data = response.json()

    assert len(data["simulations"]) == 4
    assert data["summary"]["total_tax_savings"] > 0  # From IKZE accounts


def test_get_prefill_data_success(test_client: TestClient, test_db_session: Session):
    """Test prefill data endpoint"""
    config = create_app_config()
    test_db_session.add(config)
    test_db_session.commit()

    response = test_client.get("/api/simulations/prefill")

    assert response.status_code == 200
    data = response.json()

    assert "current_age" in data
    assert "retirement_age" in data
    assert "balances" in data
    assert data["retirement_age"] == 65
    assert isinstance(data["current_age"], int)
    assert data["balances"]["ike_marcin"] == 0  # No snapshot yet


def test_get_prefill_data_with_balances(test_client: TestClient, test_db_session: Session):
    """Test prefill data with actual balances from snapshot"""
    config = create_app_config()
    test_db_session.add(config)

    # Create accounts
    ike_marcin = Account(
        name="IKE Marcin",
        type="asset",
        category="retirement",
        owner="Marcin",
        currency="PLN",
        account_wrapper="IKE",
        purpose="retirement",
    )
    ikze_ewa = Account(
        name="IKZE Ewa",
        type="asset",
        category="retirement",
        owner="Ewa",
        currency="PLN",
        account_wrapper="IKZE",
        purpose="retirement",
    )
    test_db_session.add_all([ike_marcin, ikze_ewa])
    test_db_session.commit()

    # Create snapshot
    snapshot = Snapshot(date=date(2026, 1, 31), notes="Test")
    test_db_session.add(snapshot)
    test_db_session.commit()

    # Create snapshot values
    value1 = SnapshotValue(snapshot_id=snapshot.id, account_id=ike_marcin.id, value=50000.0)
    value2 = SnapshotValue(snapshot_id=snapshot.id, account_id=ikze_ewa.id, value=30000.0)
    test_db_session.add_all([value1, value2])
    test_db_session.commit()

    response = test_client.get("/api/simulations/prefill")

    assert response.status_code == 200
    data = response.json()

    assert data["balances"]["ike_marcin"] == 50000.0
    assert data["balances"]["ikze_ewa"] == 30000.0


def test_simulate_retirement_fixed_monthly_contributions(
    test_client: TestClient, test_db_session: Session
):
    """Test simulation with fixed monthly contributions"""
    config = create_app_config()
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKE", owner="Marcin", limit_amount=28260.0)
    test_db_session.add(limit)
    test_db_session.commit()

    payload = {
        "current_age": 30,
        "retirement_age": 33,
        "simulate_ike_marcin": True,
        "simulate_ike_ewa": False,
        "simulate_ikze_marcin": False,
        "simulate_ikze_ewa": False,
        "ike_marcin_balance": 0,
        "ike_marcin_auto_fill": False,
        "ike_marcin_monthly": 1000.0,
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 200
    data = response.json()

    # Verify contribution amounts
    projections = data["simulations"][0]["yearly_projections"]
    assert projections[0]["annual_contribution"] == 12000.0  # 12 * 1000


def test_simulate_retirement_yearly_projections_structure(
    test_client: TestClient, test_db_session: Session
):
    """Test structure of yearly projections"""
    config = create_app_config()
    test_db_session.add(config)

    limit = RetirementLimit(year=2026, account_wrapper="IKZE", owner="Marcin", limit_amount=11304.0)
    test_db_session.add(limit)
    test_db_session.commit()

    payload = {
        "current_age": 30,
        "retirement_age": 32,
        "simulate_ike_marcin": False,
        "simulate_ike_ewa": False,
        "simulate_ikze_marcin": True,
        "simulate_ikze_ewa": False,
        "ikze_marcin_balance": 5000.0,
        "ikze_marcin_auto_fill": True,
        "marcin_tax_rate": 17.0,
        "annual_return_rate": 7.0,
        "limit_growth_rate": 5.0,
    }

    response = test_client.post("/api/simulations/retirement", json=payload)

    assert response.status_code == 200
    data = response.json()

    projections = data["simulations"][0]["yearly_projections"]
    assert len(projections) == 2  # 2 years until retirement

    # Check structure of first projection
    p = projections[0]
    assert "year" in p
    assert "age" in p
    assert "annual_contribution" in p
    assert "balance_end_of_year" in p
    assert "cumulative_contributions" in p
    assert "cumulative_returns" in p
    assert "annual_limit" in p
    assert "limit_utilized_pct" in p
    assert "tax_savings" in p

    # IKZE should have tax savings
    assert p["tax_savings"] > 0
