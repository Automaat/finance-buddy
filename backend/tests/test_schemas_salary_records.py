"""Tests for salary record schemas validation"""

from datetime import UTC, date, datetime, timedelta

import pytest
from pydantic import ValidationError

from app.schemas.salary_records import SalaryRecordCreate, SalaryRecordUpdate


class TestSalaryRecordCreateValidation:
    """Test SalaryRecordCreate schema validators"""

    def test_valid_salary_record_create(self):
        """Test creating valid salary record"""
        data = SalaryRecordCreate(
            date=date(2024, 1, 1),
            gross_amount=10000.0,
            contract_type="UOP",
            company="Test Company",
            owner="Marcin",
        )
        assert data.date == date(2024, 1, 1)
        assert data.gross_amount == 10000.0
        assert data.contract_type == "UOP"
        assert data.company == "Test Company"
        assert data.owner == "Marcin"

    def test_zero_gross_amount_fails(self):
        """Test that zero gross amount raises validation error"""
        with pytest.raises(ValidationError, match="Gross amount must be greater than 0"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=0.0,
                contract_type="UOP",
                company="Test Company",
                owner="Marcin",
            )

    def test_negative_gross_amount_fails(self):
        """Test that negative gross amount raises validation error"""
        with pytest.raises(ValidationError, match="Gross amount must be greater than 0"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=-1000.0,
                contract_type="UOP",
                company="Test Company",
                owner="Marcin",
            )

    def test_future_date_fails(self):
        """Test that future date raises validation error"""
        future_date = datetime.now(UTC).date() + timedelta(days=10)
        with pytest.raises(ValidationError, match="Date cannot be in the future"):
            SalaryRecordCreate(
                date=future_date,
                gross_amount=10000.0,
                contract_type="UOP",
                company="Test Company",
                owner="Marcin",
            )

    def test_empty_company_fails(self):
        """Test that empty company raises validation error"""
        with pytest.raises(ValidationError, match="Company cannot be empty"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=10000.0,
                contract_type="UOP",
                company="",
                owner="Marcin",
            )

    def test_whitespace_company_fails(self):
        """Test that whitespace-only company raises validation error"""
        with pytest.raises(ValidationError, match="Company cannot be empty"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=10000.0,
                contract_type="UOP",
                company="   ",
                owner="Marcin",
            )

    def test_invalid_owner_fails(self):
        """Test that invalid owner raises validation error"""
        with pytest.raises(ValidationError, match="Input should be 'Marcin', 'Ewa' or 'Shared'"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=10000.0,
                contract_type="UOP",
                company="Test Company",
                owner="InvalidOwner",
            )

    def test_invalid_contract_type_fails(self):
        """Test that invalid contract type raises validation error"""
        with pytest.raises(ValidationError, match="Input should be 'UOP', 'UZ', 'UoD' or 'B2B'"):
            SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=10000.0,
                contract_type="InvalidType",
                company="Test Company",
                owner="Marcin",
            )

    def test_all_valid_contract_types(self):
        """Test all valid contract types"""
        for contract_type in ["UOP", "UZ", "UoD", "B2B"]:
            data = SalaryRecordCreate(
                date=date(2024, 1, 1),
                gross_amount=10000.0,
                contract_type=contract_type,
                company="Test Company",
                owner="Marcin",
            )
            assert data.contract_type == contract_type


class TestSalaryRecordUpdateValidation:
    """Test SalaryRecordUpdate schema validators"""

    def test_valid_salary_record_update(self):
        """Test creating valid salary record update"""
        data = SalaryRecordUpdate(
            date=date(2024, 2, 1),
            gross_amount=11000.0,
            contract_type="UZ",
            company="New Company",
            owner="Ewa",
        )
        assert data.date == date(2024, 2, 1)
        assert data.gross_amount == 11000.0
        assert data.contract_type == "UZ"
        assert data.company == "New Company"
        assert data.owner == "Ewa"

    def test_partial_update(self):
        """Test partial update with only some fields"""
        data = SalaryRecordUpdate(gross_amount=11000.0)
        assert data.gross_amount == 11000.0
        assert data.date is None
        assert data.contract_type is None
        assert data.company is None
        assert data.owner is None

    def test_none_values_allowed(self):
        """Test that None values are allowed for all fields"""
        data = SalaryRecordUpdate()
        assert data.date is None
        assert data.gross_amount is None
        assert data.contract_type is None
        assert data.company is None
        assert data.owner is None

    def test_zero_gross_amount_fails(self):
        """Test that zero gross amount raises validation error"""
        with pytest.raises(ValidationError, match="Gross amount must be greater than 0"):
            SalaryRecordUpdate(gross_amount=0.0)

    def test_negative_gross_amount_fails(self):
        """Test that negative gross amount raises validation error"""
        with pytest.raises(ValidationError, match="Gross amount must be greater than 0"):
            SalaryRecordUpdate(gross_amount=-1000.0)

    def test_future_date_fails(self):
        """Test that future date raises validation error"""
        future_date = datetime.now(UTC).date() + timedelta(days=10)
        with pytest.raises(ValidationError, match="Date cannot be in the future"):
            SalaryRecordUpdate(date=future_date)

    def test_empty_company_fails(self):
        """Test that empty company raises validation error"""
        with pytest.raises(ValidationError, match="Company cannot be empty"):
            SalaryRecordUpdate(company="")

    def test_whitespace_company_fails(self):
        """Test that whitespace-only company raises validation error"""
        with pytest.raises(ValidationError, match="Company cannot be empty"):
            SalaryRecordUpdate(company="   ")

    def test_invalid_owner_fails(self):
        """Test that invalid owner raises validation error"""
        with pytest.raises(ValidationError, match="Input should be 'Marcin', 'Ewa' or 'Shared'"):
            SalaryRecordUpdate(owner="InvalidOwner")

    def test_invalid_contract_type_fails(self):
        """Test that invalid contract type raises validation error"""
        with pytest.raises(ValidationError, match="Input should be 'UOP', 'UZ', 'UoD' or 'B2B'"):
            SalaryRecordUpdate(contract_type="InvalidType")
