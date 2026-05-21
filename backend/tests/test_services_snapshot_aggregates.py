"""Tests for snapshot_aggregates runtime helpers."""

from datetime import date
from decimal import Decimal

from app.models import Account, Asset, Snapshot, SnapshotValue
from app.models.snapshot_aggregate import SnapshotAggregate
from app.services.snapshot_aggregates import recompute_all, recompute_for_snapshot


def _add_snapshot(db, d: date) -> Snapshot:
    s = Snapshot(date=d)
    db.add(s)
    db.flush()
    return s


def _add_account(db, owner="Marcin", typ="asset", category="bank") -> Account:
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
    db.flush()
    return a


def _add_sv(db, snapshot_id, account_id=None, asset_id=None, value="1000") -> SnapshotValue:
    sv = SnapshotValue(
        snapshot_id=snapshot_id,
        account_id=account_id,
        asset_id=asset_id,
        value=Decimal(value),
    )
    db.add(sv)
    db.flush()
    return sv


def test_recompute_creates_aggregate_row(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 1, 1))
    acct = _add_account(test_db_session)
    _add_sv(test_db_session, snap.id, account_id=acct.id, value="5000")

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    rows = test_db_session.query(SnapshotAggregate).all()
    assert len(rows) == 1
    assert rows[0].owner == "Marcin"
    assert float(rows[0].total_assets) == 5000.0
    assert float(rows[0].total_liabilities) == 0.0
    assert float(rows[0].net_worth) == 5000.0
    assert rows[0].month == date(2024, 1, 1)


def test_recompute_is_idempotent(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 1, 1))
    acct = _add_account(test_db_session)
    _add_sv(test_db_session, snap.id, account_id=acct.id, value="1000")

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()
    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    count = test_db_session.query(SnapshotAggregate).count()
    assert count == 1


def test_recompute_empty_snapshot(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 1, 1))

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    rows = test_db_session.query(SnapshotAggregate).all()
    assert len(rows) == 1
    assert rows[0].owner == "Shared"
    assert float(rows[0].net_worth) == 0.0


def test_recompute_multi_owner(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 1, 1))
    acct_m = _add_account(test_db_session, owner="Marcin")
    acct_e = _add_account(test_db_session, owner="Ewa")
    _add_sv(test_db_session, snap.id, account_id=acct_m.id, value="3000")
    _add_sv(test_db_session, snap.id, account_id=acct_e.id, value="2000")

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    rows = test_db_session.query(SnapshotAggregate).all()
    assert len(rows) == 2
    owners = {r.owner: r for r in rows}
    assert float(owners["Marcin"].total_assets) == 3000.0
    assert float(owners["Ewa"].total_assets) == 2000.0


def test_asset_table_goes_to_shared(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 1, 1))
    asset = Asset(name="Car", is_active=True)
    test_db_session.add(asset)
    test_db_session.flush()
    _add_sv(test_db_session, snap.id, asset_id=asset.id, value="50000")

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    rows = test_db_session.query(SnapshotAggregate).all()
    assert len(rows) == 1
    assert rows[0].owner == "Shared"
    assert float(rows[0].total_assets) == 50000.0


def test_repeated_recompute_produces_identical_rows(test_db_session):
    snap = _add_snapshot(test_db_session, date(2024, 6, 1))
    acct = _add_account(test_db_session, owner="Marcin", typ="asset", category="stock")
    _add_sv(test_db_session, snap.id, account_id=acct.id, value="12345.67")

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    first = test_db_session.query(SnapshotAggregate).one()
    first_nw = float(first.net_worth)
    first_alloc = first.allocation_json

    recompute_for_snapshot(test_db_session, snap.id)
    test_db_session.commit()

    second = test_db_session.query(SnapshotAggregate).one()
    assert float(second.net_worth) == first_nw
    assert second.allocation_json == first_alloc


def test_recompute_all(test_db_session):
    s1 = _add_snapshot(test_db_session, date(2024, 1, 1))
    s2 = _add_snapshot(test_db_session, date(2024, 2, 1))
    acct = _add_account(test_db_session)
    _add_sv(test_db_session, s1.id, account_id=acct.id, value="1000")
    _add_sv(test_db_session, s2.id, account_id=acct.id, value="2000")

    recompute_all(test_db_session)
    test_db_session.commit()

    count = test_db_session.query(SnapshotAggregate).count()
    assert count == 2
