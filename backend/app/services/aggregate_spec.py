"""Pure aggregation spec for snapshot aggregates.

No ORM or SQLAlchemy imports — safe to call from Alembic migrations
without booting the application.
"""

from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass
from datetime import date
from decimal import Decimal
from typing import Any


@dataclass
class AggregateRow:
    """One precomputed row per (snapshot_id, owner).

    'owner' is Account.owner for account-type rows, or the literal string
    'Shared' for Asset-table (non-account) entries.
    """

    snapshot_id: int
    month: date  # snapshot.date with day=1; denormalized for month-bucket grouping
    owner: str
    total_assets: Decimal
    total_liabilities: Decimal
    net_worth: Decimal
    allocation: list[dict[str, Any]]  # [{"category": str, "value": Decimal}] sorted by category


def compute_aggregates(
    snapshot: Any,
    snapshot_value_rows: Sequence[Any],
    account_rows: Sequence[Any],
    asset_rows: Sequence[Any],
) -> list[AggregateRow]:
    """Compute per-owner aggregate rows for one snapshot.

    Parameters
    ----------
    snapshot:
        Object with .id (int) and .date (date).
    snapshot_value_rows:
        Objects with .asset_id (int|None), .account_id (int|None),
        .value (Decimal|str|float).
    account_rows:
        Active Account objects with .id, .owner, .type, .category.
        Inactive accounts must be excluded by the caller.
    asset_rows:
        Active Asset objects with .id.
        Inactive assets must be excluded by the caller.

    Returns
    -------
    One AggregateRow per distinct owner. If no active rows exist, returns a
    single zero row for owner='Shared' so the month is not a gap in the chart.

    Signed-value logic mirrors dashboard/service.py:104-114:
    - Asset-table entry (asset_id in active_asset_ids) → total_assets += value, owner='Shared'
    - Account type='asset' → total_assets += value; allocation[category] += value
    - Account type='liability' → total_liabilities += value
    """
    account_map: dict[int, Any] = {a.id: a for a in account_rows}
    active_asset_ids: set[int] = {a.id for a in asset_rows}

    totals: dict[str, dict[str, Any]] = {}

    def _bucket(owner: str) -> dict[str, Any]:
        if owner not in totals:
            totals[owner] = {
                "assets": Decimal("0"),
                "liabilities": Decimal("0"),
                "alloc": {},
            }
        return totals[owner]

    for sv in snapshot_value_rows:
        raw = sv.value
        value = raw if isinstance(raw, Decimal) else Decimal(str(raw))

        if sv.asset_id is not None:
            if sv.asset_id in active_asset_ids:
                _bucket("Shared")["assets"] += value
        elif sv.account_id is not None:
            account = account_map.get(sv.account_id)
            if account is None:
                continue
            bkt = _bucket(account.owner)
            if account.type == "asset":
                bkt["assets"] += value
                cat: str = account.category
                bkt["alloc"][cat] = bkt["alloc"].get(cat, Decimal("0")) + value
            else:
                bkt["liabilities"] += value

    if not totals:
        return [
            AggregateRow(
                snapshot_id=snapshot.id,
                month=_first_of_month(snapshot.date),
                owner="Shared",
                total_assets=Decimal("0"),
                total_liabilities=Decimal("0"),
                net_worth=Decimal("0"),
                allocation=[],
            )
        ]

    result: list[AggregateRow] = []
    for owner, data in totals.items():
        allocation_sorted = [
            {"category": cat, "value": v} for cat, v in sorted(data["alloc"].items())
        ]
        result.append(
            AggregateRow(
                snapshot_id=snapshot.id,
                month=_first_of_month(snapshot.date),
                owner=owner,
                total_assets=data["assets"],
                total_liabilities=data["liabilities"],
                net_worth=data["assets"] - data["liabilities"],
                allocation=allocation_sorted,
            )
        )
    return result


def _first_of_month(d: date) -> date:
    return d.replace(day=1)
