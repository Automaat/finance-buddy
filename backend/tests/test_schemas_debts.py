"""Tests for debt schemas validation"""

from datetime import UTC, date, datetime, timedelta

import pytest
from pydantic import ValidationError

from app.schemas.debts import DebtCreate, DebtUpdate


class TestDebtCreateValidation:
    """Test DebtCreate schema validators"""

    def test_valid_debt_create(self):
        """Test creating valid debt"""
        data = DebtCreate(
            name="Test Mortgage",
            debt_type="mortgage",
            start_date=date(2024, 1, 1),
            initial_amount=500000.0,
            interest_rate=3.5,
            currency="PLN",
            notes="Test notes",
        )
        assert data.name == "Test Mortgage"
        assert data.debt_type == "mortgage"

    def test_empty_name_fails(self):
        """Test that empty name raises validation error"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            DebtCreate(
                name="",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=500000.0,
                interest_rate=3.5,
            )

    def test_whitespace_name_fails(self):
        """Test that whitespace-only name raises validation error"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            DebtCreate(
                name="   ",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=500000.0,
                interest_rate=3.5,
            )

    def test_invalid_debt_type_fails(self):
        """Test that invalid debt type raises validation error"""
        with pytest.raises(
            ValidationError, match="Debt type must be 'mortgage' or 'installment_0percent'"
        ):
            DebtCreate(
                name="Test",
                debt_type="invalid_type",
                start_date=date(2024, 1, 1),
                initial_amount=500000.0,
                interest_rate=3.5,
            )

    def test_future_start_date_fails(self):
        """Test that future start date raises validation error"""
        future_date = datetime.now(UTC).date() + timedelta(days=10)
        with pytest.raises(ValidationError, match="Start date cannot be in the future"):
            DebtCreate(
                name="Test",
                debt_type="mortgage",
                start_date=future_date,
                initial_amount=500000.0,
                interest_rate=3.5,
            )

    def test_zero_initial_amount_fails(self):
        """Test that zero initial amount raises validation error"""
        with pytest.raises(ValidationError, match="Initial amount must be greater than 0"):
            DebtCreate(
                name="Test",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=0.0,
                interest_rate=3.5,
            )

    def test_negative_initial_amount_fails(self):
        """Test that negative initial amount raises validation error"""
        with pytest.raises(ValidationError, match="Initial amount must be greater than 0"):
            DebtCreate(
                name="Test",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=-1000.0,
                interest_rate=3.5,
            )

    def test_negative_interest_rate_fails(self):
        """Test that negative interest rate raises validation error"""
        with pytest.raises(
            ValidationError, match="Interest rate must be greater than or equal to 0"
        ):
            DebtCreate(
                name="Test",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=500000.0,
                interest_rate=-1.0,
            )

    def test_invalid_currency_fails(self):
        """Test that non-PLN currency raises validation error"""
        with pytest.raises(ValidationError, match="Currency must be 'PLN'"):
            DebtCreate(
                name="Test",
                debt_type="mortgage",
                start_date=date(2024, 1, 1),
                initial_amount=500000.0,
                interest_rate=3.5,
                currency="USD",
            )


class TestDebtUpdateValidation:
    """Test DebtUpdate schema validators"""

    def test_valid_debt_update(self):
        """Test creating valid debt update"""
        data = DebtUpdate(
            name="Updated Mortgage",
            debt_type="installment_0percent",
            start_date=date(2024, 2, 1),
            initial_amount=600000.0,
            interest_rate=4.0,
            currency="PLN",
            notes="Updated notes",
        )
        assert data.name == "Updated Mortgage"
        assert data.debt_type == "installment_0percent"

    def test_partial_update(self):
        """Test partial update with only some fields"""
        data = DebtUpdate(name="Updated Name")
        assert data.name == "Updated Name"
        assert data.debt_type is None

    def test_none_values_allowed(self):
        """Test that None values are allowed for all fields"""
        data = DebtUpdate()
        assert data.name is None
        assert data.debt_type is None
        assert data.start_date is None

    def test_empty_name_fails(self):
        """Test that empty name raises validation error"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            DebtUpdate(name="")

    def test_whitespace_name_fails(self):
        """Test that whitespace-only name raises validation error"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            DebtUpdate(name="   ")

    def test_invalid_debt_type_fails(self):
        """Test that invalid debt type raises validation error"""
        with pytest.raises(
            ValidationError, match="Debt type must be 'mortgage' or 'installment_0percent'"
        ):
            DebtUpdate(debt_type="invalid_type")

    def test_future_start_date_fails(self):
        """Test that future start date raises validation error"""
        future_date = datetime.now(UTC).date() + timedelta(days=10)
        with pytest.raises(ValidationError, match="Start date cannot be in the future"):
            DebtUpdate(start_date=future_date)

    def test_zero_initial_amount_fails(self):
        """Test that zero initial amount raises validation error"""
        with pytest.raises(ValidationError, match="Initial amount must be greater than 0"):
            DebtUpdate(initial_amount=0.0)

    def test_negative_initial_amount_fails(self):
        """Test that negative initial amount raises validation error"""
        with pytest.raises(ValidationError, match="Initial amount must be greater than 0"):
            DebtUpdate(initial_amount=-1000.0)

    def test_negative_interest_rate_fails(self):
        """Test that negative interest rate raises validation error"""
        with pytest.raises(
            ValidationError, match="Interest rate must be greater than or equal to 0"
        ):
            DebtUpdate(interest_rate=-1.0)

    def test_invalid_currency_fails(self):
        """Test that non-PLN currency raises validation error"""
        with pytest.raises(ValidationError, match="Currency must be 'PLN'"):
            DebtUpdate(currency="USD")
