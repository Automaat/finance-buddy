"""Add fx_rates table.

Caches NBP table A daily rates for FX conversion of foreign-currency
compensation (USD/EUR/etc. → PLN). One row per (date, currency).

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

    if "fx_rates" not in existing_tables:
        op.create_table(
            "fx_rates",
            sa.Column("id", sa.Integer(), primary_key=True),
            sa.Column("date", sa.Date(), nullable=False),
            sa.Column("currency", sa.String(length=3), nullable=False),
            sa.Column("rate_pln", sa.Numeric(15, 6), nullable=False),
            sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
            sa.UniqueConstraint("date", "currency", name="uq_fx_rates_date_currency"),
        )
        op.create_index("ix_fx_rates_currency_date", "fx_rates", ["currency", "date"])


def downgrade() -> None:
    inspector = inspect(op.get_bind())
    existing_tables = set(inspector.get_table_names())

    if "fx_rates" in existing_tables:
        existing_indices = {idx["name"] for idx in inspector.get_indexes("fx_rates")}
        if "ix_fx_rates_currency_date" in existing_indices:
            op.drop_index("ix_fx_rates_currency_date", table_name="fx_rates")
        op.drop_table("fx_rates")
