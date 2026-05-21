"""Add company_valuations table.

Stores FMV per share history for companies (typically private). Optional
low/high range encodes uncertainty. Indexed on (company, date) for the
"latest valuation per company" lookup used to compute paper value of
equity grants.

Revision ID: 0007
Revises: 0006
Create Date: 2026-05-21

"""

from collections.abc import Sequence

import sqlalchemy as sa
from sqlalchemy import inspect

from alembic import op

revision: str = "0007"
down_revision: str | None = "0006"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "company_valuations" not in existing_tables:
        op.create_table(
            "company_valuations",
            sa.Column("id", sa.Integer(), primary_key=True),
            sa.Column("company", sa.String(length=200), nullable=False),
            sa.Column("date", sa.Date(), nullable=False),
            sa.Column("currency", sa.String(length=3), nullable=False, server_default="USD"),
            sa.Column("fmv_per_share", sa.Numeric(15, 4), nullable=False),
            sa.Column("fmv_low", sa.Numeric(15, 4), nullable=True),
            sa.Column("fmv_high", sa.Numeric(15, 4), nullable=True),
            sa.Column("source", sa.String(length=30), nullable=False),
            sa.Column("common_stock_discount_pct", sa.Numeric(5, 2), nullable=True),
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
            "ix_company_valuations_company_date",
            "company_valuations",
            ["company", "date"],
        )


def downgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "company_valuations" in existing_tables:
        existing_indices = {idx["name"] for idx in inspector.get_indexes("company_valuations")}
        if "ix_company_valuations_company_date" in existing_indices:
            op.drop_index(
                "ix_company_valuations_company_date",
                table_name="company_valuations",
            )
        op.drop_table("company_valuations")
