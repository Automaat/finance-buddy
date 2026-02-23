"""Tests for retirement schemas validation"""

from datetime import UTC, datetime

import pytest
from pydantic import ValidationError

from app.schemas.retirement import PPKContributionGenerateRequest, RetirementLimitCreate


class TestRetirementLimitCreateValidation:
    """Test RetirementLimitCreate schema validators"""

    def test_valid_retirement_limit_create(self):
        """Test creating valid retirement limit"""
        data = RetirementLimitCreate(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=10000.0,
            notes="Test limit",
        )
        assert data.year == 2024
        assert data.account_wrapper == "IKE"
        assert data.owner == "Marcin"
        assert data.limit_amount == 10000.0
        assert data.notes == "Test limit"

    def test_all_valid_wrappers(self):
        """Test all valid account wrappers"""
        for wrapper in ["IKE", "IKZE", "PPK"]:
            data = RetirementLimitCreate(
                year=2024, account_wrapper=wrapper, owner="Marcin", limit_amount=10000.0
            )
            assert data.account_wrapper == wrapper

    def test_any_non_empty_owner_valid(self):
        """Test any non-empty string is valid as owner"""
        for owner in ["Marcin", "Ewa", "CustomOwner"]:
            data = RetirementLimitCreate(
                year=2024, account_wrapper="IKE", owner=owner, limit_amount=10000.0
            )
            assert data.owner == owner

    def test_invalid_wrapper_fails(self):
        """Test that invalid wrapper raises validation error"""
        with pytest.raises(ValidationError, match="Input should be 'IKE', 'IKZE' or 'PPK'"):
            RetirementLimitCreate(
                year=2024, account_wrapper="INVALID", owner="Marcin", limit_amount=10000.0
            )

    def test_empty_owner_fails(self):
        """Test that empty owner raises validation error"""
        with pytest.raises(ValidationError, match="Owner cannot be empty"):
            RetirementLimitCreate(year=2024, account_wrapper="IKE", owner="", limit_amount=10000.0)

    def test_year_too_old_fails(self):
        """Test that year before 2000 raises validation error"""
        with pytest.raises(ValidationError, match="Year must be between 2000 and"):
            RetirementLimitCreate(
                year=1999, account_wrapper="IKE", owner="Marcin", limit_amount=10000.0
            )

    def test_year_too_far_future_fails(self):
        """Test that year too far in future raises validation error"""
        current_year = datetime.now(UTC).year
        future_year = current_year + 11
        with pytest.raises(ValidationError, match="Year must be between 2000 and"):
            RetirementLimitCreate(
                year=future_year, account_wrapper="IKE", owner="Marcin", limit_amount=10000.0
            )

    def test_year_current_year_plus_10_valid(self):
        """Test that current year + 10 is valid"""
        current_year = datetime.now(UTC).year
        data = RetirementLimitCreate(
            year=current_year + 10, account_wrapper="IKE", owner="Marcin", limit_amount=10000.0
        )
        assert data.year == current_year + 10

    def test_zero_limit_amount_fails(self):
        """Test that zero limit amount raises validation error"""
        with pytest.raises(ValidationError, match="Limit amount must be greater than 0"):
            RetirementLimitCreate(
                year=2024, account_wrapper="IKE", owner="Marcin", limit_amount=0.0
            )

    def test_negative_limit_amount_fails(self):
        """Test that negative limit amount raises validation error"""
        with pytest.raises(ValidationError, match="Limit amount must be greater than 0"):
            RetirementLimitCreate(
                year=2024, account_wrapper="IKE", owner="Marcin", limit_amount=-1000.0
            )

    def test_notes_optional(self):
        """Test that notes field is optional"""
        data = RetirementLimitCreate(
            year=2024, account_wrapper="IKE", owner="Marcin", limit_amount=10000.0
        )
        assert data.notes is None


class TestPPKContributionGenerateRequestValidation:
    """Test PPKContributionGenerateRequest schema validators"""

    def test_valid_ppk_contribution_generate_request(self):
        """Test creating valid PPK contribution request"""
        data = PPKContributionGenerateRequest(owner="Marcin", month=6, year=2024)
        assert data.owner == "Marcin"
        assert data.month == 6
        assert data.year == 2024

    def test_any_non_empty_owner_valid(self):
        """Test any non-empty string is valid as owner"""
        for owner in ["Marcin", "Ewa", "CustomOwner"]:
            data = PPKContributionGenerateRequest(owner=owner, month=1, year=2024)
            assert data.owner == owner

    def test_empty_owner_fails(self):
        """Test that empty owner raises validation error"""
        with pytest.raises(ValidationError, match="Owner cannot be empty"):
            PPKContributionGenerateRequest(owner="", month=1, year=2024)

    def test_month_too_low_fails(self):
        """Test that month below 1 raises validation error"""
        with pytest.raises(ValidationError, match="Month must be between 1 and 12"):
            PPKContributionGenerateRequest(owner="Marcin", month=0, year=2024)

    def test_month_too_high_fails(self):
        """Test that month above 12 raises validation error"""
        with pytest.raises(ValidationError, match="Month must be between 1 and 12"):
            PPKContributionGenerateRequest(owner="Marcin", month=13, year=2024)

    def test_all_valid_months(self):
        """Test all valid months 1-12"""
        for month in range(1, 13):
            data = PPKContributionGenerateRequest(owner="Marcin", month=month, year=2024)
            assert data.month == month

    def test_year_too_old_fails(self):
        """Test that year before 2019 raises validation error"""
        with pytest.raises(ValidationError, match="Year must be between 2019 and"):
            PPKContributionGenerateRequest(owner="Marcin", month=1, year=2018)

    def test_year_too_far_future_fails(self):
        """Test that year too far in future raises validation error"""
        current_year = datetime.now(UTC).year
        future_year = current_year + 2
        with pytest.raises(ValidationError, match="Year must be between 2019 and"):
            PPKContributionGenerateRequest(owner="Marcin", month=1, year=future_year)

    def test_year_current_year_plus_1_valid(self):
        """Test that current year + 1 is valid"""
        current_year = datetime.now(UTC).year
        data = PPKContributionGenerateRequest(owner="Marcin", month=1, year=current_year + 1)
        assert data.year == current_year + 1
