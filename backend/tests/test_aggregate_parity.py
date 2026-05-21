"""Parity tests: migration frozen spec == live spec; raw path == aggregate path."""

from datetime import date
from decimal import Decimal

from app.services.aggregate_spec import compute_aggregates
from app.services.dashboard.service import _get_dashboard_data_raw, get_dashboard_data
from app.services.snapshot_aggregates import recompute_all
from tests.factories import create_test_account, create_test_snapshot, create_test_snapshot_value

# ---------------------------------------------------------------------------
# Drift guard: frozen migration spec == live spec
# ---------------------------------------------------------------------------


def test_frozen_spec_matches_live_spec():
    """Output of _compute_aggregates_frozen must equal compute_aggregates."""
    import importlib.util
    from pathlib import Path

    versions_dir = Path(__file__).parent.parent / "alembic" / "versions"
    migration_path = versions_dir / "0006_backfill_snapshot_aggregates.py"
    spec = importlib.util.spec_from_file_location("migration_0006", migration_path)
    assert spec is not None and spec.loader is not None
    migration_06 = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(migration_06)

    from dataclasses import dataclass

    @dataclass
    class _Snap:
        id: int
        date: date

    @dataclass
    class _SV:
        asset_id: int | None
        account_id: int | None
        value: Decimal

    @dataclass
    class _Account:
        id: int
        owner: str
        type: str
        category: str

    @dataclass
    class _Asset:
        id: int

    snap = _Snap(id=1, date=date(2024, 6, 15))
    account = _Account(id=1, owner="Marcin", type="asset", category="stock")
    asset = _Asset(id=5)
    svs = [
        _SV(asset_id=None, account_id=1, value=Decimal("12345.67")),
        _SV(asset_id=5, account_id=None, value=Decimal("500.00")),
    ]

    live_rows = compute_aggregates(snap, svs, [account], [asset])
    frozen_rows = migration_06._compute_aggregates_frozen(
        snap.id,
        snap.date,
        [(sv.asset_id, sv.account_id, sv.value) for sv in svs],
        {account.id: (account.owner, account.type, account.category)},
        {asset.id},
    )

    assert len(live_rows) == len(frozen_rows)
    live_by_owner = {r.owner: r for r in live_rows}
    frozen_by_owner = {r.owner: r for r in frozen_rows}

    for owner in live_by_owner:
        live_row = live_by_owner[owner]
        frozen_row = frozen_by_owner[owner]
        assert live_row.total_assets == frozen_row.total_assets, (
            f"total_assets mismatch for {owner}"
        )
        assert live_row.total_liabilities == frozen_row.total_liabilities, (
            f"total_liabilities mismatch for {owner}"
        )
        assert live_row.net_worth == frozen_row.net_worth, f"net_worth mismatch for {owner}"
        assert len(live_row.allocation) == len(frozen_row.allocation), (
            f"allocation length mismatch for {owner}"
        )
        for la, fa in zip(live_row.allocation, frozen_row.allocation, strict=True):
            assert la["category"] == fa["category"]
            assert Decimal(str(la["value"])) == Decimal(str(fa["value"]))


# ---------------------------------------------------------------------------
# Dashboard output parity: raw path == aggregate path
# ---------------------------------------------------------------------------


def _seed_db(db):
    """Seed 5 snapshots with 2 owners to exercise both code paths."""
    acct_m = create_test_account(db, name="Bank Marcin", category="bank", owner="Marcin")
    acct_e = create_test_account(db, name="Bank Ewa", category="bank", owner="Ewa")
    acct_l = create_test_account(
        db,
        name="Mortgage",
        account_type="liability",
        category="mortgage",
        owner="Shared",
    )

    for i in range(5):
        snap = create_test_snapshot(db, snapshot_date=date(2024, i + 1, 1))
        create_test_snapshot_value(
            db, snap.id, account_id=acct_m.id, value=Decimal(str(1000 * (i + 1)))
        )
        create_test_snapshot_value(
            db, snap.id, account_id=acct_e.id, value=Decimal(str(500 * (i + 1)))
        )
        create_test_snapshot_value(db, snap.id, account_id=acct_l.id, value=Decimal("200000"))

    return acct_m, acct_e, acct_l


def test_aggregate_path_matches_raw_path(test_db_session):
    """Dashboard output is identical whether read from aggregates or raw rows."""
    _seed_db(test_db_session)

    # Raw path (no aggregates yet)
    raw = _get_dashboard_data_raw(test_db_session)

    # Populate aggregates, then use aggregate path
    recompute_all(test_db_session)
    test_db_session.commit()

    agg = get_dashboard_data(test_db_session)

    # Net worth history must match in length and values
    assert len(agg.net_worth_history) == len(raw.net_worth_history)
    for agg_pt, raw_pt in zip(agg.net_worth_history, raw.net_worth_history, strict=True):
        assert abs(agg_pt.value - raw_pt.value) < 0.01, (
            f"net_worth mismatch at {raw_pt.date}: agg={agg_pt.value}, raw={raw_pt.value}"
        )

    assert abs(agg.current_net_worth - raw.current_net_worth) < 0.01
    assert abs(agg.total_assets - raw.total_assets) < 0.01
    assert abs(agg.total_liabilities - raw.total_liabilities) < 0.01

    # Allocation items (sorted by category+owner)
    agg_alloc = sorted(agg.allocation, key=lambda x: (x.category, x.owner))
    raw_alloc = sorted(raw.allocation, key=lambda x: (x.category, x.owner))
    assert len(agg_alloc) == len(raw_alloc)
    for a, r in zip(agg_alloc, raw_alloc, strict=True):
        assert a.category == r.category
        assert a.owner == r.owner
        assert abs(a.value - r.value) < 0.01
