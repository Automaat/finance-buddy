from datetime import date

from app.models.persona import Persona
from tests.factories import create_test_salary_record


def test_get_all_salary_records_empty(test_client, test_db_session):
    """Test GET /api/salaries returns empty list"""
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.add(Persona(name="Ewa"))
    test_db_session.commit()

    response = test_client.get("/api/salaries")

    assert response.status_code == 200
    data = response.json()
    assert data["salary_records"] == []
    assert data["total_count"] == 0
    assert data["current_salaries"]["Marcin"] is None
    assert data["current_salaries"]["Ewa"] is None


def test_get_all_salary_records_success(test_client, test_db_session):
    """Test GET /api/salaries returns all active salary records"""
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.add(Persona(name="Ewa"))
    test_db_session.commit()

    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Company A",
        owner="Marcin",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 6, 1),
        gross_amount=8000.0,
        company="Company B",
        owner="Ewa",
    )

    response = test_client.get("/api/salaries")

    assert response.status_code == 200
    data = response.json()
    assert len(data["salary_records"]) == 2
    assert data["total_count"] == 2
    assert data["current_salaries"]["Marcin"] == 10000.0
    assert data["current_salaries"]["Ewa"] == 8000.0


def test_get_all_salary_records_filter_by_owner(test_client, test_db_session):
    """Test GET /api/salaries?owner=X filters by owner"""
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.add(Persona(name="Ewa"))
    test_db_session.commit()

    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Company A",
        owner="Marcin",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 6, 1),
        gross_amount=8000.0,
        company="Company B",
        owner="Ewa",
    )

    response = test_client.get("/api/salaries?owner=Marcin")

    assert response.status_code == 200
    data = response.json()
    assert len(data["salary_records"]) == 1
    assert data["salary_records"][0]["owner"] == "Marcin"


def test_get_all_salary_records_filter_by_date_range(test_client, test_db_session):
    """Test GET /api/salaries with date_from and date_to filters"""
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()

    create_test_salary_record(
        test_db_session,
        salary_date=date(2023, 1, 1),
        gross_amount=9000.0,
        company="Company A",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Company A",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2025, 1, 1),
        gross_amount=11000.0,
        company="Company B",
    )

    response = test_client.get("/api/salaries?date_from=2024-01-01&date_to=2024-12-31")

    assert response.status_code == 200
    data = response.json()
    assert len(data["salary_records"]) == 1
    assert data["salary_records"][0]["date"] == "2024-01-01"


def test_create_salary_record_success(test_client):
    """Test POST /api/salaries creates new salary record"""
    payload = {
        "date": "2024-01-01",
        "gross_amount": 10000.0,
        "contract_type": "UOP",
        "company": "Test Company",
        "owner": "Marcin",
    }

    response = test_client.post("/api/salaries", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["date"] == "2024-01-01"
    assert data["gross_amount"] == 10000.0
    assert data["contract_type"] == "UOP"
    assert data["company"] == "Test Company"
    assert data["owner"] == "Marcin"
    assert data["is_active"] is True


def test_create_salary_record_validation_fails(test_client):
    """Test POST /api/salaries with invalid data fails"""
    payload = {
        "date": "2024-01-01",
        "gross_amount": -1000.0,
        "contract_type": "UOP",
        "company": "Test Company",
        "owner": "Marcin",
    }

    response = test_client.post("/api/salaries", json=payload)

    assert response.status_code == 422


def test_get_salary_record_success(test_client, test_db_session):
    """Test GET /api/salaries/{salary_id} returns salary record"""
    record = create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Test Company",
    )

    response = test_client.get(f"/api/salaries/{record.id}")

    assert response.status_code == 200
    data = response.json()
    assert data["id"] == record.id
    assert data["gross_amount"] == 10000.0
    assert data["company"] == "Test Company"


def test_get_salary_record_not_found(test_client):
    """Test GET /api/salaries/999 returns 404"""
    response = test_client.get("/api/salaries/999")

    assert response.status_code == 404


def test_update_salary_record_success(test_client, test_db_session):
    """Test PATCH /api/salaries/{salary_id} updates salary record"""
    record = create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Old Company",
    )

    payload = {"gross_amount": 11000.0, "company": "New Company"}

    response = test_client.patch(f"/api/salaries/{record.id}", json=payload)

    assert response.status_code == 200
    data = response.json()
    assert data["gross_amount"] == 11000.0
    assert data["company"] == "New Company"
    assert data["date"] == "2024-01-01"


def test_update_salary_record_not_found(test_client):
    """Test PATCH /api/salaries/999 returns 404"""
    payload = {"gross_amount": 11000.0}

    response = test_client.patch("/api/salaries/999", json=payload)

    assert response.status_code == 404


def test_delete_salary_record_success(test_client, test_db_session):
    """Test DELETE /api/salaries/{salary_id} soft deletes salary record"""
    record = create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Test Company",
    )

    response = test_client.delete(f"/api/salaries/{record.id}")

    assert response.status_code == 204

    # Verify record is soft deleted
    test_db_session.refresh(record)
    assert record.is_active is False


def test_delete_salary_record_not_found(test_client):
    """Test DELETE /api/salaries/999 returns 404"""
    response = test_client.delete("/api/salaries/999")

    assert response.status_code == 404
