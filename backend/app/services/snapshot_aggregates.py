"""Runtime helpers for maintaining snapshot_aggregates.

All helpers run inside the caller's transaction and never commit.
The caller is responsible for flushing pending ORM changes before calling
recompute (required because autoflush=False in SessionLocal).
"""

from collections.abc import Iterable
from datetime import UTC, datetime

from sqlalchemy import delete, select
from sqlalchemy.orm import Session

from app.models import Account, Asset, Snapshot, SnapshotValue
from app.models.snapshot_aggregate import SnapshotAggregate
from app.services.aggregate_spec import compute_aggregates


def recompute_for_snapshot(db: Session, snapshot_id: int) -> None:
    """Delete and reinsert aggregate rows for one snapshot.

    Idempotent. Caller must flush pending ORM state and commit when done.
    """
    db.execute(delete(SnapshotAggregate).where(SnapshotAggregate.snapshot_id == snapshot_id))

    snapshot = db.execute(select(Snapshot).where(Snapshot.id == snapshot_id)).scalar_one_or_none()
    if snapshot is None:
        return

    sv_rows = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == snapshot_id))
        .scalars()
        .all()
    )

    account_ids = [sv.account_id for sv in sv_rows if sv.account_id is not None]
    accounts = (
        db.execute(select(Account).where(Account.id.in_(account_ids), Account.is_active.is_(True)))
        .scalars()
        .all()
        if account_ids
        else []
    )

    asset_ids_in_sv = [sv.asset_id for sv in sv_rows if sv.asset_id is not None]
    assets = (
        db.execute(select(Asset).where(Asset.id.in_(asset_ids_in_sv), Asset.is_active.is_(True)))
        .scalars()
        .all()
        if asset_ids_in_sv
        else []
    )

    rows = compute_aggregates(snapshot, sv_rows, accounts, assets)
    now = datetime.now(UTC)

    for row in rows:
        db.add(
            SnapshotAggregate(
                snapshot_id=row.snapshot_id,
                month=row.month,
                owner=row.owner,
                total_assets=row.total_assets,
                total_liabilities=row.total_liabilities,
                net_worth=row.net_worth,
                allocation_json=[
                    {"category": item["category"], "value": float(item["value"])}
                    for item in row.allocation
                ],
                computed_at=now,
            )
        )


def recompute_for_snapshots(db: Session, snapshot_ids: Iterable[int]) -> None:
    """Recompute aggregates for multiple snapshots. Caller commits."""
    for sid in snapshot_ids:
        recompute_for_snapshot(db, sid)


def recompute_all(db: Session) -> None:
    """Recompute all snapshot aggregates. Used by tests and backfill only."""
    snapshot_ids = db.execute(select(Snapshot.id)).scalars().all()
    recompute_for_snapshots(db, snapshot_ids)
