"""Tests for account/asset mutation invalidation of snapshot_aggregates."""

from datetime import date
from decimal import Decimal

from app.models import Account, Asset, Snapshot, SnapshotValue
from app.models.snapshot_aggregate import SnapshotAggregate
from app.schemas.accounts import AccountUpdate
from app.services.accounts import delete_account, update_account
from app.services.assets import delete_asset
from app.services.snapshot_aggregates import recompute_for_snapshot


def _setup_snap_with_account(db, acct_owner="Marcin", acct_category="bank"):
    acct = Account(
        name=f"Acct-{acct_owner}",
        type="asset",
        category=acct_category,
        owner=acct_owner,
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    db.add(acct)
    db.flush()

    snap = Snapshot(date=date(2024, 1, 1))
    db.add(snap)
    db.flush()

    sv = SnapshotValue(snapshot_id=snap.id, account_id=acct.id, value=Decimal("10000"))
    db.add(sv)
    db.flush()

    recompute_for_snapshot(db, snap.id)
    db.commit()

    return snap, acct


def test_update_account_owner_recomputes_aggregate(test_db_session):
    snap, acct = _setup_snap_with_account(test_db_session, acct_owner="Marcin")

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert before.owner == "Marcin"

    update_account(test_db_session, acct.id, AccountUpdate(owner="Ewa"))

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert after.owner == "Ewa"


def test_update_account_category_recomputes_allocation(test_db_session):
    snap, acct = _setup_snap_with_account(test_db_session, acct_category="bank")

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert before.allocation_json[0]["category"] == "bank"

    update_account(test_db_session, acct.id, AccountUpdate(category="stock"))

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert after.allocation_json[0]["category"] == "stock"


def test_update_account_name_only_does_not_recompute(test_db_session):
    snap, acct = _setup_snap_with_account(test_db_session)

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    original_computed_at = before.computed_at

    update_account(test_db_session, acct.id, AccountUpdate(name="New Name"))

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    # computed_at unchanged → no recompute
    assert after.computed_at == original_computed_at


def test_delete_account_removes_its_contribution_from_aggregate(test_db_session):
    # Two accounts in same snapshot, same owner
    acct1 = Account(
        name="Keep",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    acct2 = Account(
        name="Delete",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    test_db_session.add_all([acct1, acct2])
    test_db_session.flush()

    snap = Snapshot(date=date(2024, 2, 1))
    test_db_session.add(snap)
    test_db_session.flush()

    test_db_session.add_all(
        [
            SnapshotValue(snapshot_id=snap.id, account_id=acct1.id, value=Decimal("6000")),
            SnapshotValue(snapshot_id=snap.id, account_id=acct2.id, value=Decimal("4000")),
        ]
    )
    test_db_session.flush()
    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(before.total_assets) == 10000.0

    delete_account(test_db_session, acct2.id)

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(after.total_assets) == 6000.0


def test_other_snapshots_not_touched_when_updating_account(test_db_session):
    acct_affected = Account(
        name="Affected",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    acct_unrelated = Account(
        name="Unrelated",
        type="asset",
        category="bank",
        owner="Marcin",
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    test_db_session.add_all([acct_affected, acct_unrelated])
    test_db_session.flush()

    snap1 = Snapshot(date=date(2024, 1, 1))
    snap2 = Snapshot(date=date(2024, 2, 1))
    test_db_session.add_all([snap1, snap2])
    test_db_session.flush()

    # snap1 has acct_affected; snap2 has only acct_unrelated
    test_db_session.add(
        SnapshotValue(snapshot_id=snap1.id, account_id=acct_affected.id, value=Decimal("1000"))
    )
    test_db_session.add(
        SnapshotValue(snapshot_id=snap2.id, account_id=acct_unrelated.id, value=Decimal("2000"))
    )
    test_db_session.flush()
    recompute_for_snapshot(test_db_session, snap1.id)
    recompute_for_snapshot(test_db_session, snap2.id)
    test_db_session.commit()

    s2_before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap2.id).one()
    computed_at_before = s2_before.computed_at

    # Update owner on acct_affected → only snap1 should be recomputed
    update_account(test_db_session, acct_affected.id, AccountUpdate(owner="Ewa"))

    s1_after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap1.id).one()
    assert s1_after.owner == "Ewa"

    s2_after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap2.id).one()
    assert s2_after.computed_at == computed_at_before  # snap2 not recomputed


def test_delete_asset_removes_shared_contribution(test_db_session):
    asset = Asset(name="Car", is_active=True)
    test_db_session.add(asset)
    test_db_session.flush()

    snap = Snapshot(date=date(2024, 3, 1))
    test_db_session.add(snap)
    test_db_session.flush()

    test_db_session.add(
        SnapshotValue(snapshot_id=snap.id, asset_id=asset.id, value=Decimal("30000"))
    )
    test_db_session.flush()
    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(before.total_assets) == 30000.0

    delete_asset(test_db_session, asset.id)

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(after.total_assets) == 0.0
    assert float(after.net_worth) == 0.0
