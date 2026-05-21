"""Add equity_grants table.

Stores stock options and RSU grants with configurable vesting schedules.
JSON column holds optional custom (back/front-loaded) schedules. Indexed on
(owner, company) for the salaries page which groups by company.

Revision ID: 0008
Revises: 0007
Create Date: 2026-05-21

"""

from collections.abc import Sequence

import sqlalchemy as sa
from sqlalchemy import inspect

from alembic import op

revision: str = "0008"
down_revision: str | None = "0007"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "equity_grants" not in existing_tables:
        op.create_table(
            "equity_grants",
            sa.Column("id", sa.Integer(), primary_key=True),
            sa.Column("grant_date", sa.Date(), nullable=False),
            sa.Column("type", sa.String(length=20), nullable=False),
            sa.Column("company", sa.String(length=200), nullable=False),
            sa.Column("owner", sa.String(length=100), nullable=False),
            sa.Column("total_shares", sa.Integer(), nullable=False),
            sa.Column("strike_price", sa.Numeric(15, 4), nullable=True),
            sa.Column("currency", sa.String(length=3), nullable=False, server_default="USD"),
            sa.Column("vest_start_date", sa.Date(), nullable=False),
            sa.Column("vest_cliff_months", sa.Integer(), nullable=False, server_default="0"),
            sa.Column("vest_total_months", sa.Integer(), nullable=False),
            sa.Column("vest_frequency", sa.String(length=20), nullable=False),
            sa.Column("vest_custom_schedule", sa.JSON(), nullable=True),
            sa.Column(
                "requires_liquidity_event",
                sa.Boolean(),
                nullable=False,
                server_default=sa.text("false"),
            ),
            sa.Column("liquidity_event_date", sa.Date(), nullable=True),
            sa.Column(
                "tax_treatment",
                sa.String(length=30),
                nullable=False,
                server_default="capital_gains_19",
            ),
            sa.Column("notes", sa.String(length=500), nullable=True),
            sa.Column(
                "is_active",
                sa.Boolean(),
                nullable=False,
                server_default=sa.text("true"),
            ),
            sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        )
        op.create_index(
            "ix_equity_grants_owner_company",
            "equity_grants",
            ["owner", "company"],
        )


def downgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "equity_grants" in existing_tables:
        existing_indices = {idx["name"] for idx in inspector.get_indexes("equity_grants")}
        if "ix_equity_grants_owner_company" in existing_indices:
            op.drop_index("ix_equity_grants_owner_company", table_name="equity_grants")
        op.drop_table("equity_grants")
