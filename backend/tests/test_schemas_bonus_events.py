"""Tests for bonus event schema validation."""

from datetime import UTC, date, datetime, timedelta

import pytest
from pydantic import ValidationError

from app.schemas.bonus_events import BonusEventCreate, BonusEventUpdate


class TestBonusEventCreateValidation:
    def test_valid_bonus_event_create(self):
        data = BonusEventCreate(
            date=date(2024, 12, 15),
            amount=20000.0,
            currency="PLN",
            type="annual",
            company="Test Company",
            owner="Marcin",
            contract_type="UOP",
        )
        assert data.amount == 20000.0
        assert data.currency == "PLN"
        assert data.type == "annual"
        assert data.notes is None

    def test_currency_uppercased(self):
        data = BonusEventCreate(
            date=date(2024, 12, 15),
            amount=1000.0,
            currency="usd",
            type="signon",
            company="Co",
            owner="Marcin",
            contract_type="UOP",
        )
        assert data.currency == "USD"

    def test_unknown_currency_fails(self):
        with pytest.raises(ValidationError, match="Currency must be one of"):
            BonusEventCreate(
                date=date(2024, 12, 15),
                amount=1000.0,
                currency="XYZ",
                type="annual",
                company="Co",
                owner="Marcin",
                contract_type="UOP",
            )

    def test_zero_amount_fails(self):
        with pytest.raises(ValidationError, match="Amount must be greater than 0"):
            BonusEventCreate(
                date=date(2024, 12, 15),
                amount=0.0,
                type="annual",
                company="Co",
                owner="Marcin",
                contract_type="UOP",
            )

    def test_future_date_fails(self):
        future = datetime.now(UTC).date() + timedelta(days=10)
        with pytest.raises(ValidationError, match="Date cannot be in the future"):
            BonusEventCreate(
                date=future,
                amount=1000.0,
                type="annual",
                company="Co",
                owner="Marcin",
                contract_type="UOP",
            )

    def test_empty_company_fails(self):
        with pytest.raises(ValidationError, match="Company cannot be empty"):
            BonusEventCreate(
                date=date(2024, 12, 15),
                amount=1000.0,
                type="annual",
                company="   ",
                owner="Marcin",
                contract_type="UOP",
            )

    def test_invalid_type_fails(self):
        with pytest.raises(ValidationError):
            BonusEventCreate(
                date=date(2024, 12, 15),
                amount=1000.0,
                type="bogus",
                company="Co",
                owner="Marcin",
                contract_type="UOP",
            )


class TestBonusEventUpdateValidation:
    def test_partial_update(self):
        data = BonusEventUpdate(amount=15000.0)
        assert data.amount == 15000.0
        assert data.currency is None
        assert data.company is None

    def test_invalid_amount_in_update_fails(self):
        with pytest.raises(ValidationError, match="Amount must be greater than 0"):
            BonusEventUpdate(amount=-1.0)

    def test_invalid_currency_in_update_fails(self):
        with pytest.raises(ValidationError, match="Currency must be one of"):
            BonusEventUpdate(currency="XYZ")
