"""Add performance indices on hot query paths.

Indices target full-table scans confirmed via EXPLAIN ANALYZE:
- transactions(account_id, date) — date-range filters in /api/transactions
- accounts(owner) — owner groupby in dashboard aggregates
- snapshot_values(asset_id) — asset lookups when computing allocation

Snapshot(date) is already covered by the existing UNIQUE index, which
PostgreSQL uses bidirectionally (backward index scan for ``ORDER BY date DESC``).

Revision ID: 0004
Revises: 0003
Create Date: 2026-05-20

"""

from collections.abc import Sequence

from sqlalchemy import inspect

from alembic import op

revision: str = "0004"
down_revision: str | None = "0003"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


_INDICES: tuple[tuple[str, str, list[str]], ...] = (
    ("ix_transactions_account_id_date", "transactions", ["account_id", "date"]),
    ("ix_accounts_owner", "accounts", ["owner"]),
    ("ix_snapshot_values_asset_id", "snapshot_values", ["asset_id"]),
)


def upgrade() -> None:
    inspector = inspect(op.get_bind())
    for name, table, cols in _INDICES:
        existing = {idx["name"] for idx in inspector.get_indexes(table)}
        if name not in existing:
            op.create_index(name, table, cols)


def downgrade() -> None:
    inspector = inspect(op.get_bind())
    for name, table, _ in _INDICES:
        existing = {idx["name"] for idx in inspector.get_indexes(table)}
        if name in existing:
            op.drop_index(name, table_name=table)
