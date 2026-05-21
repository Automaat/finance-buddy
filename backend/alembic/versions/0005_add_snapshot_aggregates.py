"""Add snapshot_aggregates table for precomputed per-owner aggregates.

Grain: (snapshot_id, owner). The 'Shared' owner holds Asset-table
(non-account) contributions. See SnapshotAggregate model and
aggregate_spec.compute_aggregates() for full field semantics.

DB-level CASCADE on snapshot_id FK ensures aggregate rows are cleaned up
automatically when a snapshot is deleted (verify with \\d snapshot_aggregates
after upgrade).

Revision ID: 0005
Revises: 0004
Create Date: 2026-05-21
"""

from collections.abc import Sequence

import sqlalchemy as sa
from sqlalchemy import inspect

from alembic import op

revision: str = "0005"
down_revision: str | None = "0004"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    # Baseline 0001 uses Base.metadata.create_all against current metadata,
    # which already creates snapshot_aggregates on fresh databases. Skip when present.
    if inspect(op.get_bind()).has_table("snapshot_aggregates"):
        return
    op.create_table(
        "snapshot_aggregates",
        sa.Column("id", sa.Integer, primary_key=True),
        sa.Column(
            "snapshot_id",
            sa.Integer,
            sa.ForeignKey("snapshots.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("month", sa.Date, nullable=False),
        sa.Column("owner", sa.String(100), nullable=False),
        sa.Column("total_assets", sa.Numeric(15, 2), nullable=False),
        sa.Column("total_liabilities", sa.Numeric(15, 2), nullable=False),
        sa.Column("net_worth", sa.Numeric(15, 2), nullable=False),
        sa.Column("allocation_json", sa.JSON, nullable=False),
        sa.Column("computed_at", sa.DateTime(timezone=True), nullable=False),
        sa.UniqueConstraint("snapshot_id", "owner", name="uix_snapshot_agg_snapshot_owner"),
    )
    op.create_index("ix_snapshot_aggregates_month", "snapshot_aggregates", ["month"])


def downgrade() -> None:
    op.drop_index("ix_snapshot_aggregates_month", table_name="snapshot_aggregates")
    op.drop_table("snapshot_aggregates")
