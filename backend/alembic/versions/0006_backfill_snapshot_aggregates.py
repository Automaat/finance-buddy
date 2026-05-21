"""Backfill snapshot_aggregates for all existing snapshots.

The aggregation logic is copied verbatim from services.aggregate_spec as the
frozen local function _compute_aggregates_frozen(). This duplication is
intentional: Alembic migrations must not import app code (which evolves),
but the frozen copy preserves the logic that was correct as of this revision.

If aggregate_spec.py changes after this migration is applied, write a new
migration — do NOT update this file.

Revision ID: 0006
Revises: 0005
Create Date: 2026-05-21
"""

import json
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import UTC, date, datetime
from decimal import Decimal
from typing import Any

from sqlalchemy import text

from alembic import op

revision: str = "0006"
down_revision: str | None = "0005"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


# ---------- Frozen copy of aggregate_spec.compute_aggregates ----------
# Intentionally duplicated. Migrations must not import app code.
# A parity test (test_aggregate_parity.py) asserts byte-equal output between
# this function and the live aggregate_spec.compute_aggregates().


@dataclass
class _AggRow:
    snapshot_id: int
    month: date
    owner: str
    total_assets: Decimal
    total_liabilities: Decimal
    net_worth: Decimal
    allocation: list[dict[str, Any]]


def _first_of_month_frozen(d: date) -> date:
    return d.replace(day=1)


def _compute_aggregates_frozen(
    snapshot_id: int,
    snapshot_date: date,
    sv_tuples: list[tuple[Any, Any, Any]],  # (asset_id, account_id, value)
    account_map: dict[int, tuple[str, str, str]],  # id → (owner, type, category)
    active_asset_ids: set[int],
) -> list[_AggRow]:
    """Frozen copy of aggregate_spec.compute_aggregates for migration use."""
    totals: dict[str, dict[str, Any]] = {}

    def _bucket(owner: str) -> dict[str, Any]:
        if owner not in totals:
            totals[owner] = {"assets": Decimal("0"), "liabilities": Decimal("0"), "alloc": {}}
        return totals[owner]

    for sv_asset_id, sv_account_id, sv_value in sv_tuples:
        value = sv_value if isinstance(sv_value, Decimal) else Decimal(str(sv_value))
        if sv_asset_id is not None:
            if sv_asset_id in active_asset_ids:
                _bucket("Shared")["assets"] += value
        elif sv_account_id is not None:
            acct = account_map.get(sv_account_id)
            if acct is None:
                continue
            owner, acct_type, category = acct
            bkt = _bucket(owner)
            if acct_type == "asset":
                bkt["assets"] += value
                bkt["alloc"][category] = bkt["alloc"].get(category, Decimal("0")) + value
            else:
                bkt["liabilities"] += value

    month = _first_of_month_frozen(snapshot_date)
    if not totals:
        return [_AggRow(snapshot_id, month, "Shared", Decimal("0"), Decimal("0"), Decimal("0"), [])]

    result = []
    for owner, data in totals.items():
        alloc = [{"category": c, "value": v} for c, v in sorted(data["alloc"].items())]
        result.append(
            _AggRow(
                snapshot_id,
                month,
                owner,
                data["assets"],
                data["liabilities"],
                data["assets"] - data["liabilities"],
                alloc,
            )
        )
    return result


# -----------------------------------------------------------------------


def _parse_date(raw: Any) -> date:
    """Parse a date from either a Python date object or an ISO string."""
    if isinstance(raw, date):
        return raw
    return date.fromisoformat(str(raw))


def upgrade() -> None:
    bind = op.get_bind()
    now = datetime.now(UTC)

    account_rows = bind.execute(
        text("SELECT id, owner, type, category FROM accounts WHERE is_active = TRUE")
    ).fetchall()
    account_map: dict[int, tuple[str, str, str]] = {
        r.id: (r.owner, r.type, r.category) for r in account_rows
    }

    active_asset_ids: set[int] = {
        r.id for r in bind.execute(text("SELECT id FROM assets WHERE is_active = TRUE")).fetchall()
    }

    snapshots = bind.execute(text("SELECT id, date FROM snapshots ORDER BY date")).fetchall()

    for snap in snapshots:
        snap_id: int = snap.id
        snap_date: date = _parse_date(snap.date)

        sv_rows = bind.execute(
            text(
                "SELECT asset_id, account_id, value FROM snapshot_values WHERE snapshot_id = :sid"
            ),
            {"sid": snap_id},
        ).fetchall()

        agg_rows = _compute_aggregates_frozen(
            snap_id,
            snap_date,
            [(r.asset_id, r.account_id, r.value) for r in sv_rows],
            account_map,
            active_asset_ids,
        )

        for row in agg_rows:
            alloc_json = json.dumps(
                [
                    {"category": item["category"], "value": float(item["value"])}
                    for item in row.allocation
                ]
            )
            bind.execute(
                text(
                    "INSERT INTO snapshot_aggregates "
                    "(snapshot_id, month, owner, total_assets, total_liabilities, "
                    "net_worth, allocation_json, computed_at) "
                    "VALUES (:snapshot_id, :month, :owner, :total_assets, "
                    ":total_liabilities, :net_worth, :allocation_json, :computed_at)"
                ),
                {
                    "snapshot_id": row.snapshot_id,
                    "month": row.month.isoformat(),
                    "owner": row.owner,
                    "total_assets": str(row.total_assets),
                    "total_liabilities": str(row.total_liabilities),
                    "net_worth": str(row.net_worth),
                    "allocation_json": alloc_json,
                    "computed_at": now.isoformat(),
                },
            )


def downgrade() -> None:
    op.execute(text("DELETE FROM snapshot_aggregates"))
