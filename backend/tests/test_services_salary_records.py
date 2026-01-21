from datetime import date

import pytest
from fastapi import HTTPException

from app.models import SalaryRecord
from app.schemas.salary_records import SalaryRecordCreate, SalaryRecordUpdate
from app.services.salary_records import (
    create_salary_record,
    delete_salary_record,
    get_all_salary_records,
    get_current_salary,
    get_salary_record,
    update_salary_record,
)


def test_create_salary_record_success(test_db_session):
    """Test creating a salary record successfully"""
    data = SalaryRecordCreate(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Test Company",
        owner="Marcin",
    )

    result = create_salary_record(test_db_session, data)

    assert result.id is not None
    assert result.date == date(2024, 1, 1)
    assert result.gross_amount == 10000.0
    assert result.contract_type == "UOP"
    assert result.company == "Test Company"
    assert result.owner == "Marcin"
    assert result.is_active is True


def test_create_salary_record_duplicate(test_db_session):
    """Test creating duplicate salary record fails"""
    # Create first record
    record = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record)
    test_db_session.commit()

    # Try to create duplicate (same owner, date)
    data = SalaryRecordCreate(
        date=date(2024, 1, 1),
        gross_amount=11000.0,
        contract_type="UOP",
        company="Company B",
        owner="Marcin",
    )

    with pytest.raises(HTTPException) as exc_info:
        create_salary_record(test_db_session, data)

    assert exc_info.value.status_code == 409
    assert "already exists" in exc_info.value.detail


def test_get_all_salary_records(test_db_session):
    """Test getting all salary records"""
    record1 = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    record2 = SalaryRecord(
        date=date(2024, 6, 1),
        gross_amount=12000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record1)
    test_db_session.add(record2)
    test_db_session.commit()

    result = get_all_salary_records(test_db_session)

    assert result.total_count == 2
    assert len(result.salary_records) == 2
    assert result.salary_records[0].gross_amount == 12000.0
    assert result.salary_records[1].gross_amount == 10000.0
    assert result.current_salary_marcin == 12000.0


def test_get_all_salary_records_with_filters(test_db_session):
    """Test getting salary records with filters"""
    record1 = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    record2 = SalaryRecord(
        date=date(2024, 6, 1),
        gross_amount=8000.0,
        contract_type="UOP",
        company="Company B",
        owner="Ewa",
        is_active=True,
    )
    test_db_session.add(record1)
    test_db_session.add(record2)
    test_db_session.commit()

    result = get_all_salary_records(test_db_session, owner="Marcin")

    assert result.total_count == 1
    assert result.salary_records[0].owner == "Marcin"


def test_get_salary_record(test_db_session):
    """Test getting a single salary record"""
    record = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Test Company",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record)
    test_db_session.commit()

    result = get_salary_record(test_db_session, record.id)

    assert result.id == record.id
    assert result.gross_amount == 10000.0
    assert result.company == "Test Company"


def test_get_salary_record_not_found(test_db_session):
    """Test getting non-existent salary record fails"""
    with pytest.raises(HTTPException) as exc_info:
        get_salary_record(test_db_session, 999)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail


def test_update_salary_record(test_db_session):
    """Test updating a salary record"""
    record = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Old Company",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record)
    test_db_session.commit()

    data = SalaryRecordUpdate(gross_amount=11000.0, company="New Company")
    result = update_salary_record(test_db_session, record.id, data)

    assert result.gross_amount == 11000.0
    assert result.company == "New Company"
    assert result.date == date(2024, 1, 1)


def test_update_salary_record_duplicate(test_db_session):
    """Test updating salary record to duplicate owner+date fails"""
    # Create two records
    record1 = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    record2 = SalaryRecord(
        date=date(2024, 2, 1),
        gross_amount=11000.0,
        contract_type="UOP",
        company="Company B",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add_all([record1, record2])
    test_db_session.commit()

    # Try to update record2 to have same date as record1
    data = SalaryRecordUpdate(date=date(2024, 1, 1))

    with pytest.raises(HTTPException) as exc_info:
        update_salary_record(test_db_session, record2.id, data)

    assert exc_info.value.status_code == 409
    assert "conflicts with existing record" in exc_info.value.detail


def test_delete_salary_record(test_db_session):
    """Test soft deleting a salary record"""
    record = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Test Company",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record)
    test_db_session.commit()

    delete_salary_record(test_db_session, record.id)

    test_db_session.refresh(record)
    assert record.is_active is False


def test_get_current_salary(test_db_session):
    """Test getting current salary for owner"""
    record1 = SalaryRecord(
        date=date(2023, 1, 1),
        gross_amount=10000.0,
        contract_type="UOP",
        company="Company A",
        owner="Marcin",
        is_active=True,
    )
    record2 = SalaryRecord(
        date=date(2024, 1, 1),
        gross_amount=12000.0,
        contract_type="UOP",
        company="Company B",
        owner="Marcin",
        is_active=True,
    )
    test_db_session.add(record1)
    test_db_session.add(record2)
    test_db_session.commit()

    result = get_current_salary(test_db_session, "Marcin")

    assert result == 12000.0
