"""Tests for investment service functions"""

from datetime import date
from decimal import Decimal

from app.models import Account, Snapshot, SnapshotValue, Transaction
from app.services.investment import get_category_stats


class TestGetCategoryStats:
    """Test get_category_stats function"""

    def test_get_category_stats_stock_success(self, test_db_session):
        """Test calculating stock stats with positive returns"""
        # Create stock accounts
        acc1 = Account(
            name="IKE Stock",
            type="asset",
            category="stock",
            account_wrapper="IKE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        acc2 = Account(
            name="Regular Stock",
            type="asset",
            category="stock",
            account_wrapper=None,
            owner="Ewa",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add_all([acc1, acc2])
        test_db_session.commit()

        # Create transactions
        t1 = Transaction(
            account_id=acc1.id,
            amount=Decimal("5000.0"),
            date=date(2024, 1, 15),
            owner="Marcin",
            is_active=True,
        )
        t2 = Transaction(
            account_id=acc2.id,
            amount=Decimal("3000.0"),
            date=date(2024, 2, 10),
            owner="Ewa",
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        # Create snapshot
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        sv1 = SnapshotValue(snapshot_id=snapshot.id, account_id=acc1.id, value=Decimal("6000.0"))
        sv2 = SnapshotValue(snapshot_id=snapshot.id, account_id=acc2.id, value=Decimal("3500.0"))
        test_db_session.add_all([sv1, sv2])
        test_db_session.commit()

        # Get stats
        result = get_category_stats(test_db_session, "stock")

        assert result.category == "stock"
        assert result.total_value == 9500.0
        assert result.total_contributed == 8000.0
        assert result.returns == 1500.0
        assert result.roi_percentage == 18.75  # (1500/8000) * 100

    def test_get_category_stats_bond_success(self, test_db_session):
        """Test calculating bond stats"""
        # Create bond account
        account = Account(
            name="IKZE Bond",
            type="asset",
            category="bond",
            account_wrapper="IKZE",
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transaction
        transaction = Transaction(
            account_id=account.id,
            amount=Decimal("10000.0"),
            date=date(2024, 6, 1),
            owner="Marcin",
            is_active=True,
        )
        test_db_session.add(transaction)
        test_db_session.commit()

        # Create snapshot
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("10500.0"))
        test_db_session.add(sv)
        test_db_session.commit()

        # Get stats
        result = get_category_stats(test_db_session, "bond")

        assert result.category == "bond"
        assert result.total_value == 10500.0
        assert result.total_contributed == 10000.0
        assert result.returns == 500.0
        assert result.roi_percentage == 5.0

    def test_get_category_stats_no_accounts(self, test_db_session):
        """Test stats when no accounts exist for category"""
        result = get_category_stats(test_db_session, "stock")

        assert result.category == "stock"
        assert result.total_value == 0.0
        assert result.total_contributed == 0.0
        assert result.returns == 0.0
        assert result.roi_percentage == 0.0

    def test_get_category_stats_negative_returns(self, test_db_session):
        """Test calculating stats with negative returns"""
        # Create stock account
        account = Account(
            name="Stock Account",
            type="asset",
            category="stock",
            account_wrapper=None,
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transaction
        transaction = Transaction(
            account_id=account.id,
            amount=Decimal("5000.0"),
            date=date(2024, 1, 1),
            owner="Marcin",
            is_active=True,
        )
        test_db_session.add(transaction)
        test_db_session.commit()

        # Create snapshot with loss
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("4000.0"))
        test_db_session.add(sv)
        test_db_session.commit()

        # Get stats
        result = get_category_stats(test_db_session, "stock")

        assert result.category == "stock"
        assert result.total_value == 4000.0
        assert result.total_contributed == 5000.0
        assert result.returns == -1000.0
        assert result.roi_percentage == -20.0  # (-1000/5000) * 100

    def test_get_category_stats_no_snapshots(self, test_db_session):
        """Test stats when accounts exist but no snapshots"""
        # Create account
        account = Account(
            name="Stock Account",
            type="asset",
            category="stock",
            account_wrapper=None,
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
            date=date(2024, 1, 1),
            owner="Marcin",
            is_active=True,
        )
        test_db_session.add(transaction)
        test_db_session.commit()

        # Get stats (no snapshot)
        result = get_category_stats(test_db_session, "stock")

        assert result.category == "stock"
        assert result.total_value == 0.0
        assert result.total_contributed == 1000.0
        assert result.returns == -1000.0
        assert result.roi_percentage == -100.0

    def test_get_category_stats_inactive_transactions_excluded(self, test_db_session):
        """Test that inactive transactions are not counted"""
        # Create account
        account = Account(
            name="Stock Account",
            type="asset",
            category="stock",
            account_wrapper=None,
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create active and inactive transactions
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("2000.0"),
            date=date(2024, 1, 1),
            owner="Marcin",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2024, 2, 1),
            owner="Marcin",
            is_active=False,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        # Create snapshot
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("2500.0"))
        test_db_session.add(sv)
        test_db_session.commit()

        # Get stats
        result = get_category_stats(test_db_session, "stock")

        assert result.total_contributed == 2000.0  # Only active transaction
        assert result.total_value == 2500.0
        assert result.returns == 500.0

    def test_get_category_stats_transactions_after_snapshot(self, test_db_session):
        """Test that transactions after latest snapshot date are excluded"""
        # Create account
        account = Account(
            name="Stock Account",
            type="asset",
            category="stock",
            account_wrapper=None,
            owner="Marcin",
            currency="PLN",
            purpose="general",
        )
        test_db_session.add(account)
        test_db_session.commit()

        # Create transactions: one before snapshot, one after
        t1 = Transaction(
            account_id=account.id,
            amount=Decimal("2000.0"),
            date=date(2024, 6, 15),
            owner="Marcin",
            is_active=True,
        )
        t2 = Transaction(
            account_id=account.id,
            amount=Decimal("3000.0"),
            date=date(2025, 1, 15),  # After snapshot date
            owner="Marcin",
            is_active=True,
        )
        test_db_session.add_all([t1, t2])
        test_db_session.commit()

        # Create snapshot on 2024-12-31
        snapshot = Snapshot(date=date(2024, 12, 31), notes="Test snapshot")
        test_db_session.add(snapshot)
        test_db_session.commit()

        sv = SnapshotValue(snapshot_id=snapshot.id, account_id=account.id, value=Decimal("2500.0"))
        test_db_session.add(sv)
        test_db_session.commit()

        # Get stats - should only count t1 (2000), not t2 (3000)
        result = get_category_stats(test_db_session, "stock")

        assert result.total_contributed == 2000.0  # Should exclude t2
        assert result.total_value == 2500.0
        assert result.returns == 500.0
        assert result.roi_percentage == 25.0  # (500/2000) * 100
