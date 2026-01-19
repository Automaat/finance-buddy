"""Tests for retirement service functions"""

from datetime import date
from decimal import Decimal

import pytest
from fastapi import HTTPException

from app.models import Account, AppConfig, RetirementLimit, SalaryRecord, Snapshot, SnapshotValue, Transaction
from app.schemas.retirement import PPKContributionGenerateRequest
from app.services.retirement import (
    generate_ppk_contributions,
    get_or_create_limit,
    get_ppk_stats,
    get_yearly_stats,
    update_limit,
)


class TestGetYearlyStats:
    """Test get_yearly_stats function"""

    def test_get_yearly_stats_ike_success(self, test_db_session):
        """Test calculating IKE yearly stats with contributions"""
        # Create IKE account for Marcin
        account = Account(
            name="IKE Stocks",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

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
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2024, 1, 15),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("2000.0"),
            date=date(2024, 2, 10),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

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
        account = Account(
            name="IKZE Bonds",
            type="asset",
            category="bond",
            account_wrapper="IKZE",
            owner="Ewa",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

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
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("5000.0"),
            date=date(2024, 1, 15),
            owner="Ewa",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2024, 1, 15),
            owner="Ewa",
            transaction_type="employer",
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

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
        account = Account(
            name="IKE Stocks",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transactions - one typed, one untyped
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2024, 1, 15),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("2000.0"),
            date=date(2024, 2, 10),
            owner="Marcin",
            transaction_type=None,  # Untyped
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        result = get_yearly_stats(test_db_session, 2024)

        assert len(result) == 1
        assert result[0].total_contributed == 5000.0
        assert result[0].employee_contributed == 5000.0  # 3000 + 2000 untyped
        assert result[0].employer_contributed == 0.0

    def test_get_yearly_stats_warning_threshold(self, test_db_session):
        """Test that is_warning is True when >= 90% of limit used"""
        account = Account(
            name="IKE Stocks",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create transaction for 90% of limit
        transaction = Transaction(
            account_id=account.id,
            amount=Decimal("9000.0"),
            date=date(2024, 1, 15),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        test_db_session.add(transaction)
        test_db_session.commit()

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
        account = Account(
            name="IKE Stocks",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        result = get_yearly_stats(test_db_session, 2024)
        assert result == []

    def test_get_yearly_stats_filters_by_owner(self, test_db_session):
        """Test filtering stats by owner"""
        # Create accounts for both owners
        account_marcin = Account(
            name="IKE Marcin",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        account_ewa = Account(
            name="IKE Ewa",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Ewa",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add_all([account_marcin, account_ewa])
        test_db_session.commit()

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
        account = Account(
            name="IKE Stocks",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        limit = RetirementLimit(
            year=2024,
            account_wrapper="IKE",
            owner="Marcin",
            limit_amount=Decimal("10000.0"),
        )
        test_db_session.add(limit)
        test_db_session.commit()

        # Create active and inactive transactions
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2024, 1, 15),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("5000.0"),
            date=date(2024, 2, 10),
            owner="Marcin",
            transaction_type="employee",
            is_active=False,  # Inactive
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

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
        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transactions
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("1000.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("1500.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="employer",
            is_active=True,
        )
        t3 = Transaction(
            account_id=account.id,
            amount=Decimal("250.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="government",
            is_active=True,
        )
        test_db_session.add_all([t1, t2, t3])
        test_db_session.commit()

        # Create snapshot with current value
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        snapshot_value = SnapshotValue(
            snapshot_id=snapshot.id, account_id=account.id, value=Decimal("3500.0")
        )
        test_db_session.add(snapshot_value)
        test_db_session.commit()

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
        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transactions - some typed, some untyped
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("1000.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("500.0"),
            date=date(2024, 2, 28),
            owner="Marcin",
            transaction_type=None,  # Untyped
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        # Create snapshot
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test")
        test_db_session.add(snapshot)
        test_db_session.commit()
        snapshot_value = SnapshotValue(
            snapshot_id=snapshot.id, account_id=account.id, value=Decimal("1500.0")
        )
        test_db_session.add(snapshot_value)
        test_db_session.commit()

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
        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transaction
        transaction = Transaction(
            account_id=account.id,
            amount=Decimal("1000.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        test_db_session.add(transaction)
        test_db_session.commit()

        result = get_ppk_stats(test_db_session)

        assert len(result) == 1
        assert result[0].total_value == 0.0
        assert result[0].total_contributed == 1000.0
        assert result[0].returns == -1000.0
        assert result[0].roi_percentage == -100.0

    def test_get_ppk_stats_filters_by_owner(self, test_db_session):
        """Test filtering PPK stats by owner"""
        # Create PPK accounts for both owners
        account_marcin = Account(
            name="PPK Marcin",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        account_ewa = Account(
            name="PPK Ewa",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Ewa",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add_all([account_marcin, account_ewa])
        test_db_session.commit()

        # Create transactions
        t1 = Transaction(
            account_id=account_marcin.id,
            amount=Decimal("1000.0"),
            date=date(2024, 1, 31),
            owner="Marcin",
            transaction_type="employee",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account_ewa.id,
            amount=Decimal("2000.0"),
            date=date(2024, 1, 31),
            owner="Ewa",
            transaction_type="employee",
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        # Create snapshots
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test")
        test_db_session.add(snapshot)
        test_db_session.commit()
        sv1 = SnapshotValue(
            snapshot_id=snapshot.id, account_id=account_marcin.id, value=Decimal("1100.0")
        )
        sv2 = SnapshotValue(
            snapshot_id=snapshot.id, account_id=account_ewa.id, value=Decimal("2200.0")
        )
        test_db_session.add_all([sv1, sv2])
        test_db_session.commit()

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
        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Marcin",
        )
        test_db_session.add(salary)
        test_db_session.commit()

        # Create PPK account
        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
            is_active=True,
            receives_contributions=True,
        )
        test_db_session.add(account)
        test_db_session.commit()

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

        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
            is_active=True,
            receives_contributions=True,
        )
        test_db_session.add(account)
        test_db_session.commit()

        request = PPKContributionGenerateRequest(owner="Marcin", month=1, year=2024)

        with pytest.raises(HTTPException) as exc_info:
            generate_ppk_contributions(test_db_session, request)

        assert exc_info.value.status_code == 400
        assert "No salary record found" in exc_info.value.detail

    def test_generate_ppk_contributions_no_app_config_fails(self, test_db_session):
        """Test that missing app config raises error"""
        # Create salary and account without app config
        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Marcin",
        )
        test_db_session.add(salary)
        test_db_session.commit()

        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
            is_active=True,
            receives_contributions=True,
        )
        test_db_session.add(account)
        test_db_session.commit()

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

        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Unknown",
        )
        test_db_session.add(salary)
        test_db_session.commit()

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

        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Marcin",
        )
        test_db_session.add(salary)
        test_db_session.commit()

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

        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("10000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Marcin",
        )
        test_db_session.add(salary)
        test_db_session.commit()

        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Marcin",
            currency="PLN",
            purpose="general",
            is_active=True,
            receives_contributions=True,
        )
        test_db_session.add(account)
        test_db_session.commit()

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

        salary = SalaryRecord(
            date=date(2024, 1, 1),
            gross_amount=Decimal("8000.0"),
            contract_type="UOP",
            company="Test Company",
            owner="Ewa",
        )
        test_db_session.add(salary)
        test_db_session.commit()

        account = Account(
            name="PPK Account",
            type="asset",
            category="stock",
            account_wrapper="PPK",
            owner="Ewa",
            currency="PLN",
            purpose="general",
            is_active=True,
            receives_contributions=True,
        )
        test_db_session.add(account)
        test_db_session.commit()

        request = PPKContributionGenerateRequest(owner="Ewa", month=1, year=2024)
        result = generate_ppk_contributions(test_db_session, request)

        assert result.owner == "Ewa"
        assert result.gross_salary == 8000.0
        assert result.employee_amount == 240.0  # 8000 * 3%
        assert result.employer_amount == 200.0  # 8000 * 2.5%
        assert result.total_amount == 440.0
