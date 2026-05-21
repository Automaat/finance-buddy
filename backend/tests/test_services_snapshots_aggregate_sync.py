"""Tests that create_snapshot and update_snapshot keep aggregates in sync."""

from datetime import date

import pytest

from app.models import Account
from app.models.snapshot_aggregate import SnapshotAggregate
from app.schemas.snapshots import SnapshotCreate, SnapshotUpdate, SnapshotValueInput
from app.services.snapshots import create_snapshot, update_snapshot


def _make_account(db, owner="Marcin", typ="asset", category="bank") -> Account:
    a = Account(
        name=f"Acct-{owner}-{category}",
        type=typ,
        category=category,
        owner=owner,
        currency="PLN",
        purpose="general",
        is_active=True,
    )
    db.add(a)
    db.commit()
    db.refresh(a)
    return a


def test_create_snapshot_builds_aggregate(test_db_session):
    acct = _make_account(test_db_session)

    data = SnapshotCreate(
        date=date(2024, 1, 1),
        values=[SnapshotValueInput(account_id=acct.id, value=10000.0)],
    )
    snap = create_snapshot(test_db_session, data)

    rows = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).all()
    assert len(rows) == 1
    assert rows[0].owner == "Marcin"
    assert float(rows[0].total_assets) == 10000.0
    assert float(rows[0].net_worth) == 10000.0


def test_create_snapshot_liability_in_aggregate(test_db_session):
    asset_acct = _make_account(test_db_session, owner="Marcin", typ="asset", category="bank")
    liab_acct = _make_account(test_db_session, owner="Marcin", typ="liability", category="mortgage")

    data = SnapshotCreate(
        date=date(2024, 2, 1),
        values=[
            SnapshotValueInput(account_id=asset_acct.id, value=50000.0),
            SnapshotValueInput(account_id=liab_acct.id, value=20000.0),
        ],
    )
    snap = create_snapshot(test_db_session, data)

    row = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(row.total_assets) == 50000.0
    assert float(row.total_liabilities) == 20000.0
    assert float(row.net_worth) == 30000.0


def test_update_snapshot_refreshes_aggregate(test_db_session):
    acct = _make_account(test_db_session)

    create_data = SnapshotCreate(
        date=date(2024, 3, 1),
        values=[SnapshotValueInput(account_id=acct.id, value=5000.0)],
    )
    snap = create_snapshot(test_db_session, create_data)

    before = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(before.net_worth) == 5000.0

    update_data = SnapshotUpdate(values=[SnapshotValueInput(account_id=acct.id, value=8000.0)])
    update_snapshot(test_db_session, snap.id, update_data)

    after = test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one()
    assert float(after.net_worth) == 8000.0


def test_update_snapshot_rollback_leaves_table_consistent(test_db_session):
    """Simulated integrity error leaves aggregates unchanged."""
    acct = _make_account(test_db_session)

    create_data = SnapshotCreate(
        date=date(2024, 4, 1),
        values=[SnapshotValueInput(account_id=acct.id, value=1000.0)],
    )
    snap = create_snapshot(test_db_session, create_data)
    original_nw = float(
        test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one().net_worth
    )

    # Try update with a non-existent account to trigger 404 (no DB change)
    from fastapi import HTTPException

    with pytest.raises(HTTPException):
        update_snapshot(
            test_db_session,
            snap.id,
            SnapshotUpdate(values=[SnapshotValueInput(account_id=99999, value=9999.0)]),
        )

    # Session is rolled back by the exception path; original aggregate intact after reset
    test_db_session.rollback()
    final_nw = float(
        test_db_session.query(SnapshotAggregate).filter_by(snapshot_id=snap.id).one().net_worth
    )
    assert final_nw == original_nw
