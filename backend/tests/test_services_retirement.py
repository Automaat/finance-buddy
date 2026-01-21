"""Tests for retirement service functions"""

from datetime import date
from decimal import Decimal

import pytest
from fastapi import HTTPException

from app.models import (
    AppConfig,
    RetirementLimit,
    Transaction,
)
from app.schemas.retirement import PPKContributionGenerateRequest
from app.services.retirement import (
    generate_ppk_contributions,
    get_or_create_limit,
    get_ppk_stats,
    get_yearly_stats,
    update_limit,
)
from tests.factories import (
    create_test_account,
    create_test_salary_record,
    create_test_snapshot,
    create_test_snapshot_value,
    create_test_transaction,
)


class TestGetYearlyStats:
    """Test get_yearly_stats function"""

    def test_get_yearly_stats_ike_success(self, test_db_session):
        """Test calculating IKE yearly stats with contributions"""
        # Create IKE account for Marcin
        account = create_test_account(
            test_db_session,
            name="IKE Stocks",
            category="stock",
            account_wrapper="IKE",
        )

        # Create retirement limit
        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transactions
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=3000.0,
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=2000.0,
            transaction_date=date(2024, 2, 10),
            transaction_type="employee",
        )

        # Get stats
        result = get_yearly_stats(test_db_session, 2024)

        assert len(result) == 1
        assert result[0].year == 2024
        assert result[0].account_wrapper == "IKE"
        assert result[0].owner == "Marcin"
        assert result[0].limit_amount == 10000.0
        assert result[0].total_contributed == 5000.0
        assert result[0].employee_contributed == 5000.0
        assert result[0].employer_contributed == 0.0
        assert result[0].remaining == 5000.0
        assert result[0].percentage_used == 50.0
        assert result[0].is_warning is False

    def test_get_yearly_stats_ikze_with_employer_contributions(self, test_db_session):
        """Test IKZE stats with both employee and employer contributions"""
        # Create IKZE account for Ewa
        account = create_test_account(
            test_db_session,
            name="IKZE Bonds",
            category="bond",
            account_wrapper="IKZE",
            owner="Ewa",
        )

        # Create retirement limit
        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKZE",
            owner="Ewa",
            limit_amount=Decimal("20000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transactions
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=5000.0,
            owner="Ewa",
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=3000.0,
            owner="Ewa",
            transaction_type="employer",
        )

        # Get stats
        result = get_yearly_stats(test_db_session, 2024, owner="Ewa")

        assert len(result) == 1
        assert result[0].account_wrapper == "IKZE"
        assert result[0].owner == "Ewa"
        assert result[0].total_contributed == 8000.0
        assert result[0].employee_contributed == 5000.0
        assert result[0].employer_contributed == 3000.0
        assert result[0].remaining == 12000.0
        assert result[0].percentage_used == 40.0

    def test_get_yearly_stats_untyped_transactions_assigned_to_employee(self, test_db_session):
        """Test that untyped transactions are assigned to employee contributions"""
        account = create_test_account(
            test_db_session,
            name="IKE Stocks",
            category="stock",
            account_wrapper="IKE",
        )

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transactions - one typed, one untyped
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=3000.0,
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=2000.0,
            transaction_date=date(2024, 2, 10),
            transaction_type=None,  # Untyped
        )

        result = get_yearly_stats(test_db_session, 2024)

        assert len(result) == 1
        assert result[0].total_contributed == 5000.0
        assert result[0].employee_contributed == 5000.0  # 3000 + 2000 untyped
        assert result[0].employer_contributed == 0.0

    def test_get_yearly_stats_warning_threshold(self, test_db_session):
        """Test that is_warning is True when >= 90% of limit used"""
        account = create_test_account(
            test_db_session,
            name="IKE Stocks",
            category="stock",
            account_wrapper="IKE",
        )

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transaction for 90% of limit
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=9000.0,
            transaction_type="employee",
        )

        result = get_yearly_stats(test_db_session, 2024)

        assert len(result) == 1
        assert result[0].percentage_used == 90.0
        assert result[0].is_warning is True

    def test_get_yearly_stats_no_accounts_returns_empty(self, test_db_session):
        """Test that no accounts returns empty list"""
        result = get_yearly_stats(test_db_session, 2024)
        assert result == []

    def test_get_yearly_stats_no_limits_returns_empty(self, test_db_session):
        """Test that accounts without limits are not included"""
        create_test_account(
            test_db_session,
            name="IKE Stocks",
            category="stock",
            account_wrapper="IKE",
        )

        result = get_yearly_stats(test_db_session, 2024)
        assert result == []

    def test_get_yearly_stats_filters_by_owner(self, test_db_session):
        """Test filtering stats by owner"""
        # Create accounts for both owners
        create_test_account(
            test_db_session,
            name="IKE Marcin",
            category="stock",
            account_wrapper="IKE",
        )
        create_test_account(
            test_db_session,
            name="IKE Ewa",
            category="stock",
            account_wrapper="IKE",
            owner="Ewa",
        )

        # Create limits for both
        limit_marcin = RetirementLimit(
            year=2024, account_wrapper="IKE", owner="Marcin", limit_amount=Decimal("10000.0")
        )
        limit_ewa = RetirementLimit(
            year=2024, account_wrapper="IKE", owner="Ewa", limit_amount=Decimal("10000.0")
        )
        test_db_session.add_all([limit_marcin, limit_ewa])
        test_db_session.commit()

        # Get stats filtered by Marcin
        result = get_yearly_stats(test_db_session, 2024, owner="Marcin")

        assert len(result) == 1
        assert result[0].owner == "Marcin"

    def test_get_yearly_stats_inactive_transactions_excluded(self, test_db_session):
        """Test that inactive transactions are excluded from calculations"""
        account = create_test_account(
            test_db_session,
            name="IKE Stocks",
            category="stock",
            account_wrapper="IKE",
        )

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create active and inactive transactions
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=3000.0,
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=5000.0,
            transaction_date=date(2024, 2, 10),
            transaction_type="employee",
            is_active=False,  # Inactive
        )

        result = get_yearly_stats(test_db_session, 2024)

        assert len(result) == 1
        assert result[0].total_contributed == 3000.0  # Only active transaction


class TestGetOrCreateLimit:
    """Test get_or_create_limit function"""

    def test_create_new_limit(self, test_db_session):
        """Test creating a new retirement limit"""
        result = get_or_create_limit(test_db_session, 2024, "IKE", "Marcin", 10000.0, "Test notes")

        assert result.id is not None
        assert result.year == 2024
        assert result.account_wrapper == "IKE"
        assert result.owner == "Marcin"
        assert result.limit_amount == Decimal("10000.0")
        assert result.notes == "Test notes"

    def test_return_existing_limit(self, test_db_session):
        """Test that existing limit is returned instead of creating new one"""
        # Create initial limit
        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
            notes="Original notes",
        )
        test_db_session.add(limit)
        test_db_session.commit()
        limit_id = limit.id

        # Call get_or_create_limit
        result = get_or_create_limit(test_db_session, 2024, "IKE", "Marcin", 15000.0, "New notes")

        # Should return existing limit unchanged
        assert result.id == limit_id
        assert result.limit_amount == Decimal("10000.0")  # Unchanged
        assert result.notes == "Original notes"  # Unchanged


class TestUpdateLimit:
    """Test update_limit function"""

    def test_update_existing_limit(self, test_db_session):
        """Test updating an existing retirement limit"""
        # Create initial limit
        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
            notes="Original notes",
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Update the limit
        result = update_limit(test_db_session, 2024, "IKE", "Marcin", 15000.0, "Updated notes")

        assert result.limit_amount == Decimal("15000.0")
        assert result.notes == "Updated notes"

    def test_update_limit_creates_if_not_exists(self, test_db_session):
        """Test that update_limit creates new limit if it doesn't exist"""
        result = update_limit(test_db_session, 2024, "IKE", "Marcin", 10000.0, "New limit")

        assert result.id is not None
        assert result.year == 2024
        assert result.account_wrapper == "IKE"
        assert result.owner == "Marcin"
        assert result.limit_amount == Decimal("10000.0")
        assert result.notes == "New limit"

    def test_update_limit_with_none_notes_preserves_existing(self, test_db_session):
        """Test that updating with None notes preserves existing notes"""
        # Create initial limit with notes
        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
            notes="Original notes",
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Update with None notes
        result = update_limit(test_db_session, 2024, "IKE", "Marcin", 15000.0, None)

        assert result.limit_amount == Decimal("15000.0")
        assert result.notes == "Original notes"  # Preserved


class TestGetPPKStats:
    """Test get_ppk_stats function"""

    def test_get_ppk_stats_success(self, test_db_session):
        """Test calculating PPK stats with contributions and returns"""
        # Create PPK account for Marcin
        account = create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
        )

        # Create transactions
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=1000.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=1500.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="employer",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=250.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="government",
        )

        # Create snapshot with current value
        snapshot = create_test_snapshot(
            test_db_session, snapshot_date=date(2024, 12, 31), notes="Test snapshot"
        )
        create_test_snapshot_value(
            test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=3500.0
        )

        # Get stats
        result = get_ppk_stats(test_db_session, owner="Marcin")

        assert len(result) == 1
        assert result[0].owner == "Marcin"
        assert result[0].employee_contributed == 1000.0
        assert result[0].employer_contributed == 1500.0
        assert result[0].government_contributed == 250.0
        assert result[0].total_contributed == 2750.0
        assert result[0].total_value == 3500.0
        assert result[0].returns == 750.0
        assert result[0].roi_percentage == 27.27  # (750/2750) * 100 = 27.27

    def test_get_ppk_stats_untyped_transactions_assigned_to_employee(self, test_db_session):
        """Test that untyped transactions are assigned to employee contributions"""
        account = create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
        )

        # Create transactions - some typed, some untyped
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=1000.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=500.0,
            transaction_date=date(2024, 2, 28),
            transaction_type=None,  # Untyped
        )

        # Create snapshot
        snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 12, 31))
        create_test_snapshot_value(
            test_db_session, snapshot_id=snapshot.id, account_id=account.id, value=1500.0
        )

        result = get_ppk_stats(test_db_session)

        assert len(result) == 1
        assert result[0].total_contributed == 1500.0
        assert result[0].employee_contributed == 1500.0  # 1000 + 500 untyped
        assert result[0].employer_contributed == 0.0
        assert result[0].government_contributed == 0.0

    def test_get_ppk_stats_no_accounts_returns_empty(self, test_db_session):
        """Test that no PPK accounts returns empty list"""
        result = get_ppk_stats(test_db_session)
        assert result == []

    def test_get_ppk_stats_no_snapshots_returns_zero_value(self, test_db_session):
        """Test that accounts without snapshots show zero total value"""
        account = create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
        )

        # Create transaction
        create_test_transaction(
            test_db_session,
            account_id=account.id,
            amount=1000.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="employee",
        )

        result = get_ppk_stats(test_db_session)

        assert len(result) == 1
        assert result[0].total_value == 0.0
        assert result[0].total_contributed == 1000.0
        assert result[0].returns == -1000.0
        assert result[0].roi_percentage == -100.0

    def test_get_ppk_stats_filters_by_owner(self, test_db_session):
        """Test filtering PPK stats by owner"""
        # Create PPK accounts for both owners
        account_marcin = create_test_account(
            test_db_session,
            name="PPK Marcin",
            category="stock",
            account_wrapper="PPK",
        )
        account_ewa = create_test_account(
            test_db_session,
            name="PPK Ewa",
            category="stock",
            account_wrapper="PPK",
            owner="Ewa",
        )

        # Create transactions
        create_test_transaction(
            test_db_session,
            account_id=account_marcin.id,
            amount=1000.0,
            transaction_date=date(2024, 1, 31),
            transaction_type="employee",
        )
        create_test_transaction(
            test_db_session,
            account_id=account_ewa.id,
            amount=2000.0,
            transaction_date=date(2024, 1, 31),
            owner="Ewa",
            transaction_type="employee",
        )

        # Create snapshots
        snapshot = create_test_snapshot(test_db_session, snapshot_date=date(2024, 12, 31))
        create_test_snapshot_value(
            test_db_session, snapshot_id=snapshot.id, account_id=account_marcin.id, value=1100.0
        )
        create_test_snapshot_value(
            test_db_session, snapshot_id=snapshot.id, account_id=account_ewa.id, value=2200.0
        )

        # Filter by Marcin
        result = get_ppk_stats(test_db_session, owner="Marcin")

        assert len(result) == 1
        assert result[0].owner == "Marcin"
        assert result[0].total_contributed == 1000.0
        assert result[0].total_value == 1100.0


class TestGeneratePPKContributions:
    """Test generate_ppk_contributions function"""

    def test_generate_ppk_contributions_success(self, test_db_session):
        """Test successfully generating PPK contributions"""
        # Create app config
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("2.0"),
            ppk_employer_rate_ewa=Decimal("1.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        # Create salary record
        create_test_salary_record(test_db_session, company="Test Company")

        # Create PPK account
        account = create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
            receives_contributions=True,
        )

        # Generate contributions
        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)
        result = generate_ppk_contributions(test_db_session, request)

        assert result.owner == "Marcin"
        assert result.month == 1
        assert result.year == 2024
        assert result.gross_salary == 10000.0
        assert result.employee_amount == 200.0  # 10000 * 2%
        assert result.employer_amount == 150.0  # 10000 * 1.5%
        assert result.total_amount == 350.0
        assert len(result.transactions_created) == 2

        # Verify transactions were created
        transactions = (
            test_db_session.query(Transaction)
            .filter(Transaction.account_id == account.id)
            .order_by(Transaction.transaction_type)
            .all()
        )
        assert len(transactions) == 2
        assert transactions[0].transaction_type == "employee"
        assert transactions[0].amount == Decimal("200.0")
        assert transactions[0].date == date(2024, 1, 31)
        assert transactions[1].transaction_type == "employer"
        assert transactions[1].amount == Decimal("150.0")
        assert transactions[1].date == date(2024, 1, 31)

    def test_generate_ppk_contributions_no_salary_record_fails(self, test_db_session):
        """Test that missing salary record raises error"""
        # Create app config and PPK account without salary record
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("2.0"),
            ppk_employer_rate_ewa=Decimal("1.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
            receives_contributions=True,
        )

        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)

        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 400
        assert "No salary record found" in exc_info.value.detail

    def test_generate_ppk_contributions_no_app_config_fails(self, test_db_session):
        """Test that missing app config raises error"""
        # Create salary and account without app config
        create_test_salary_record(test_db_session)

        create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
            receives_contributions=True,
        )

        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)

        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 500
        assert "App configuration not found" in exc_info.value.detail

    def test_generate_ppk_contributions_unknown_owner_fails(self, test_db_session):
        """Test that unknown owner raises error"""
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("2.0"),
            ppk_employer_rate_ewa=Decimal("1.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        create_test_salary_record(test_db_session, owner="Unknown")

        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)

        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 400
        assert "No salary record found" in exc_info.value.detail

    def test_generate_ppk_contributions_no_ppk_account_fails(self, test_db_session):
        """Test that missing PPK account raises error"""
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("2.0"),
            ppk_employer_rate_ewa=Decimal("1.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        create_test_salary_record(test_db_session)

        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)

        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 404
        assert "No active PPK account found" in exc_info.value.detail

    def test_generate_ppk_contributions_duplicate_fails(self, test_db_session):
        """Test that duplicate contributions for same month raise error"""
        # Setup
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("2.0"),
            ppk_employer_rate_ewa=Decimal("1.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        create_test_salary_record(test_db_session)

        create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
            receives_contributions=True,
        )

        # Create first contribution
        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)
        generate_ppk_contributions(test_db_session, request)

        # Try to create duplicate
        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 409
        assert "Contributions already exist" in exc_info.value.detail

    def test_generate_ppk_contributions_for_ewa(self, test_db_session):
        """Test generating contributions for Ewa with different rates"""
        # Create app config with different rates for Ewa
        config = AppConfig(
            id=1,
            birth_date=date(1990, 1, 1),
            retirement_age=65,
            retirement_monthly_salary=Decimal("5000"),
            allocation_real_estate=20,
            allocation_stocks=50,
            allocation_bonds=20,
            allocation_gold=5,
            allocation_commodities=5,
            ppk_employee_rate_marcin=Decimal("2.0"),
            ppk_employer_rate_marcin=Decimal("1.5"),
            ppk_employee_rate_ewa=Decimal("3.0"),
            ppk_employer_rate_ewa=Decimal("2.5"),
        )
        test_db_session.add(config)
        test_db_session.commit()

        create_test_salary_record(test_db_session, gross_amount=8000.0, owner="Ewa")

        create_test_account(
            test_db_session,
            name="PPK Account",
            category="stock",
            account_wrapper="PPK",
            owner="Ewa",
            receives_contributions=True,
        )

        request = PPKContributionGenerateRequest(owner="Ewa", month=1, year=2024)
        result = generate_ppk_contributions(test_db_session, request)

        assert result.owner == "Ewa"
        assert result.gross_salary == 8000.0
        assert result.employee_amount == 240.0  # 8000 * 3%
        assert result.employer_amount == 200.0  # 8000 * 2.5%
        assert result.total_amount == 440.0
