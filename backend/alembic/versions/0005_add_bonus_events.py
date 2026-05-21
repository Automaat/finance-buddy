"""Add bonus_events table.

Stores discrete compensation bonus events (annual, sign-on, spot, retention)
as part of total compensation tracking on the salaries page. Indexed on
(owner, date) since the salaries view filters and orders by owner+date.

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
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "bonus_events" not in existing_tables:
        op.create_table(
            "bonus_events",
            sa.Column("id", sa.Integer(), primary_key=True),
            sa.Column("date", sa.Date(), nullable=False),
            sa.Column("amount", sa.Numeric(15, 2), nullable=False),
            sa.Column("currency", sa.String(length=3), nullable=False, server_default="PLN"),
            sa.Column("type", sa.String(length=20), nullable=False),
            sa.Column("company", sa.String(length=200), nullable=False),
            sa.Column("owner", sa.String(length=100), nullable=False),
            sa.Column("contract_type", sa.String(length=50), nullable=False),
            sa.Column("notes", sa.String(length=500), nullable=True),
            sa.Column(
                "is_active",
                sa.Boolean(),
                nullable=False,
                server_default=sa.text("true"),
            ),
            sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        )
        op.create_index("ix_bonus_events_owner_date", "bonus_events", ["owner", "date"])


def downgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "bonus_events" in existing_tables:
        existing_indices = {idx["name"] for idx in inspector.get_indexes("bonus_events")}
        if "ix_bonus_events_owner_date" in existing_indices:
            op.drop_index("ix_bonus_events_owner_date", table_name="bonus_events")
        op.drop_table("bonus_events")
